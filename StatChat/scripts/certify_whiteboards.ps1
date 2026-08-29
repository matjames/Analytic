$ErrorActionPreference = 'Stop'
$baseUrl = if ($env:STATCHAT_CERT_BASE_URL) { $env:STATCHAT_CERT_BASE_URL.TrimEnd('/') } else { 'http://localhost:4000' }
$secretLine = Get-Content (Join-Path $PSScriptRoot '..\..\.env') | Where-Object { $_ -match '^STATGATE_REGISTRY_JWT_SECRET=' } | Select-Object -First 1
$secret = ($secretLine -split '=', 2)[1].Trim()
function B64([byte[]]$bytes) { [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_') }
function Token($id, $name, $tenant) {
  $header = B64 ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
  $payload = @{ sub=$id; name=$name; email="$id@example.test"; organizationId=$tenant; role='analyst'; exp=[DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() } | ConvertTo-Json -Compress
  $body = "$header.$(B64 ([Text.Encoding]::UTF8.GetBytes($payload)))"
  $hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
  "$body.$(B64 ($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))))"
}
function Request($method, $path, $token, $body = $null) {
  try {
    $args = @{ Uri="$baseUrl$path"; Method=$method; Headers=@{Authorization="Bearer $token"}; UseBasicParsing=$true }
    if ($null -ne $body) { $args.ContentType='application/json'; $args.Body=$body | ConvertTo-Json -Compress }
    $response = Invoke-WebRequest @args
    [pscustomobject]@{Status=[int]$response.StatusCode; Body=$response.Content}
  } catch { [pscustomobject]@{Status=[int]$_.Exception.Response.StatusCode; Body=''} }
}
function JsonArray($body) {
  $value = $body | ConvertFrom-Json
  if ($value.PSObject.Properties.Name -contains 'value') { return @($value.value) }
  return @($value)
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "whiteboard-cert-$stamp"; $foreignTenant = "whiteboard-foreign-$stamp"
$ownerId = "whiteboard-owner-$stamp"; $editorId = "whiteboard-editor-$stamp"; $viewerId = "whiteboard-viewer-$stamp"; $foreignId = "whiteboard-foreign-user-$stamp"
$ownerToken = Token $ownerId 'Whiteboard Owner' $tenant
$editorToken = Token $editorId 'Whiteboard Editor' $tenant
$viewerToken = Token $viewerId 'Whiteboard Viewer' $tenant
$foreignToken = Token $foreignId 'Foreign Whiteboard User' $foreignTenant
$null = Request GET '/users/me' $ownerToken
$null = Request GET '/users/me' $editorToken
$null = Request GET '/users/me' $viewerToken
$null = Request GET '/users/me' $foreignToken

$createResponse = Request POST '/v1/chat/whiteboards' $ownerToken @{title='Shared design board'; data='[{"color":"#165c92","width":3,"points":[{"x":4,"y":6}]}]'; createdBy='spoofed-owner'}
$board = $createResponse.Body | ConvertFrom-Json
$ownerBoards = JsonArray (Request GET '/v1/chat/whiteboards' $ownerToken).Body
$foreignBoards = JsonArray (Request GET '/v1/chat/whiteboards' $foreignToken).Body
$foreignRead = Request GET "/v1/chat/whiteboards/$($board.id)" $foreignToken

$editorAdd = Request POST "/v1/chat/whiteboards/$($board.id)/members" $ownerToken @{userId=$editorId; role='editor'}
$viewerAdd = Request POST "/v1/chat/whiteboards/$($board.id)/members" $ownerToken @{userId=$viewerId; role='viewer'}
$editorMembers = JsonArray (Request GET "/v1/chat/whiteboards/$($board.id)/members" $editorToken).Body
$foreignMembers = Request GET "/v1/chat/whiteboards/$($board.id)/members" $foreignToken

$editorUpdate = Request PUT "/v1/chat/whiteboards/$($board.id)" $editorToken @{title='Shared design board v2'; data='[{"color":"#c44b36","width":3,"points":[{"x":10,"y":12},{"x":30,"y":42}]}]'; version=1}
$updated = $editorUpdate.Body | ConvertFrom-Json
$staleUpdate = Request PUT "/v1/chat/whiteboards/$($board.id)" $ownerToken @{title='Stale owner write'; data='[]'; version=1}
$viewerUpdate = Request PUT "/v1/chat/whiteboards/$($board.id)" $viewerToken @{title='Viewer write'; data='[]'; version=2}
$editorManage = Request POST "/v1/chat/whiteboards/$($board.id)/members" $editorToken @{userId=$foreignId; role='viewer'}
$foreignTarget = Request POST "/v1/chat/whiteboards/$($board.id)/members" $ownerToken @{userId=$foreignId; role='viewer'}
$revisions = JsonArray (Request GET "/v1/chat/whiteboards/$($board.id)/revisions" $viewerToken).Body
$editorDelete = Request DELETE "/v1/chat/whiteboards/$($board.id)" $editorToken
$ownerDelete = Request DELETE "/v1/chat/whiteboards/$($board.id)" $ownerToken
$deletedRead = Request GET "/v1/chat/whiteboards/$($board.id)" $ownerToken

$result = [ordered]@{
  authenticatedOwner = $createResponse.Status -eq 201 -and $board.createdBy -eq $ownerId -and $board.role -eq 'owner' -and $board.canEdit -eq $true
  tenantListIsolation = ($ownerBoards.id -contains $board.id) -and -not ($foreignBoards.id -contains $board.id)
  tenantReadIsolation = $foreignRead.Status -eq 404
  ownerMemberManagement = $editorAdd.Status -eq 201 -and $viewerAdd.Status -eq 201
  memberReadAccess = ($editorMembers.userId -contains $editorId) -and ($editorMembers.userId -contains $viewerId)
  memberTenantIsolation = $foreignMembers.Status -eq 404
  editorUpdate = $editorUpdate.Status -eq 200 -and $updated.version -eq 2 -and $updated.updatedBy -eq $editorId
  optimisticConflict = $staleUpdate.Status -eq 409
  viewerWriteDenied = $viewerUpdate.Status -eq 403
  ownerOnlyManagement = $editorManage.Status -eq 403
  crossTenantMemberDenied = $foreignTarget.Status -eq 404
  revisionHistory = $revisions.Count -ge 2 -and ($revisions.version -contains 1) -and ($revisions.version -contains 2)
  ownerOnlyDelete = $editorDelete.Status -eq 403 -and $ownerDelete.Status -eq 204 -and $deletedRead.Status -eq 404
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
