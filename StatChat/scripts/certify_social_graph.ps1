$ErrorActionPreference = 'Stop'
$baseUrl = if ($env:STATCHAT_CERT_BASE_URL) { $env:STATCHAT_CERT_BASE_URL.TrimEnd('/') } else { 'http://localhost:4000' }
$secretLine = Get-Content (Join-Path $PSScriptRoot '..\..\.env') | Where-Object { $_ -match '^STATGATE_REGISTRY_JWT_SECRET=' } | Select-Object -First 1
$secret = ($secretLine -split '=', 2)[1].Trim()
function B64([byte[]]$bytes) { [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_') }
function Token($id, $name, $tenant) {
  $header = B64 ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
  $payload = @{ sub=$id; name=$name; email="$id@example.test"; organizationId=$tenant; role='member'; exp=[DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() } | ConvertTo-Json -Compress
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
$tenant = "social-cert-$stamp"; $foreignTenant = "social-foreign-$stamp"
$aliceId = "social-alice-$stamp"; $bobId = "social-bob-$stamp"; $carolId = "social-carol-$stamp"; $foreignId = "social-foreign-$stamp"
$aliceToken = Token $aliceId 'Alice Network' $tenant
$bobToken = Token $bobId 'Bob Network' $tenant
$carolToken = Token $carolId 'Carol Network' $tenant
$foreignToken = Token $foreignId 'Foreign Network' $foreignTenant
$null = Request GET '/users/me' $aliceToken
$null = Request GET '/users/me' $bobToken
$null = Request GET '/users/me' $carolToken
$null = Request GET '/users/me' $foreignToken

$requestOne = Request POST '/collaboration/connections' $aliceToken @{targetUserId=$bobId}
$requestBody = $requestOne.Body | ConvertFrom-Json
$requestTwo = Request POST '/collaboration/connections' $aliceToken @{targetUserId=$bobId}
$incomingBefore = JsonArray (Request GET '/collaboration/connection-requests' $bobToken).Body
$wrongResponder = Request POST "/collaboration/connection-requests/$($requestBody.id)/accept" $aliceToken
$accept = Request POST "/collaboration/connection-requests/$($requestBody.id)/accept" $bobToken
$aliceConnections = JsonArray (Request GET '/collaboration/connections' $aliceToken).Body
$bobConnections = JsonArray (Request GET '/collaboration/connections' $bobToken).Body

$declineRequest = Request POST '/collaboration/connections' $carolToken @{targetUserId=$bobId}
$declineBody = $declineRequest.Body | ConvertFrom-Json
$decline = Request POST "/collaboration/connection-requests/$($declineBody.id)/decline" $bobToken
$bobAfterDecline = JsonArray (Request GET '/collaboration/connection-requests' $bobToken).Body
$self = Request POST '/collaboration/connections' $aliceToken @{targetUserId=$aliceId}
$foreign = Request POST '/collaboration/connections' $aliceToken @{targetUserId=$foreignId}
$remove = Request DELETE '/collaboration/connections' $aliceToken @{targetUserId=$bobId}
$aliceAfterRemove = JsonArray (Request GET '/collaboration/connections' $aliceToken).Body
$bobAfterRemove = JsonArray (Request GET '/collaboration/connections' $bobToken).Body

$result = [ordered]@{
  pendingRequest = $requestOne.Status -eq 200 -and $requestBody.status -eq 'pending' -and $requestBody.direction -eq 'outgoing'
  pendingIsIdempotent = $requestTwo.Status -eq 200 -and (($requestTwo.Body | ConvertFrom-Json).id -eq $requestBody.id) -and $incomingBefore.id -contains $requestBody.id
  recipientOnlyAccept = $wrongResponder.Status -eq 404 -and $accept.Status -eq 200
  reciprocalAcceptance = ($aliceConnections.connectedToId -contains $bobId) -and ($bobConnections.connectedToId -contains $aliceId)
  declineRemovesPending = $decline.Status -eq 200 -and -not ($bobAfterDecline.id -contains $declineBody.id)
  boundaryValidation = $self.Status -eq 400 -and $foreign.Status -eq 404
  reciprocalRemoval = $remove.Status -eq 204 -and -not ($aliceAfterRemove.connectedToId -contains $bobId) -and -not ($bobAfterRemove.connectedToId -contains $aliceId)
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
