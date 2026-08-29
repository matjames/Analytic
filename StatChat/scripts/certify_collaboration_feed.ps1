$ErrorActionPreference = 'Stop'
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
    $args = @{ Uri="http://localhost:4000$path"; Method=$method; Headers=@{Authorization="Bearer $token"}; UseBasicParsing=$true }
    if ($null -ne $body) { $args.ContentType='application/json'; $args.Body=$body | ConvertTo-Json -Compress }
    $response = Invoke-WebRequest @args
    [pscustomobject]@{Status=[int]$response.StatusCode; Body=$response.Content}
  } catch { [pscustomobject]@{Status=[int]$_.Exception.Response.StatusCode; Body=''} }
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "feed-cert-$stamp"; $foreignTenant = "feed-foreign-$stamp"
$authorId = "feed-author-$stamp"; $peerId = "feed-peer-$stamp"; $foreignId = "feed-foreign-user-$stamp"
$authorToken = Token $authorId 'Verified Author' $tenant; $foreignToken = Token $foreignId 'Foreign User' $foreignTenant
$peerToken = Token $peerId 'Verified Peer' $tenant
$null = Request GET '/collaboration/posts' $peerToken
$null = Request GET '/collaboration/posts' $foreignToken
$post = Request POST '/collaboration/posts' $authorToken @{author='Spoofed'; role='admin'; org='Foreign'; text='Tenant-scoped collaboration post'; likes=999}
$created = $post.Body | ConvertFrom-Json
$authorPosts = @((Request GET '/collaboration/posts' $authorToken).Body | ConvertFrom-Json)
$foreignPosts = @((Request GET '/collaboration/posts' $foreignToken).Body | ConvertFrom-Json)
$foreignLike = Request POST "/collaboration/posts/$($created.id)/like" $foreignToken @{}
$foreignShare = Request POST "/collaboration/posts/$($created.id)/share" $foreignToken @{}
$foreignComment = Request POST "/collaboration/posts/$($created.id)/comments" $foreignToken @{text='cross-tenant'}
$comment = Request POST "/collaboration/posts/$($created.id)/comments" $authorToken @{author='Spoofed'; text='Authenticated comment'}
$createdComment = $comment.Body | ConvertFrom-Json
$sameTenantConnection = Request POST '/collaboration/connections' $authorToken @{targetUserId=$peerId}
$sameTenantRequest = $sameTenantConnection.Body | ConvertFrom-Json
$acceptedConnection = Request POST "/collaboration/connection-requests/$($sameTenantRequest.id)/accept" $peerToken
$selfConnection = Request POST '/collaboration/connections' $authorToken @{targetUserId=$authorId}
$foreignConnection = Request POST '/collaboration/connections' $authorToken @{targetUserId=$foreignId}
$connections = @((Request GET '/collaboration/connections' $authorToken).Body | ConvertFrom-Json)
$result = [ordered]@{
  authenticatedAuthor = $post.Status -eq 200 -and $created.author -eq 'Verified Author' -and $created.authorId -eq $authorId -and $created.org -eq $tenant -and $created.likes -eq 0
  tenantReadIsolation = ($authorPosts.id -contains $created.id) -and -not ($foreignPosts.id -contains $created.id)
  tenantWriteIsolation = $foreignLike.Status -eq 404 -and $foreignShare.Status -eq 404 -and $foreignComment.Status -eq 404
  authenticatedComment = $comment.Status -eq 200 -and $createdComment.author -eq 'Verified Author' -and $createdComment.authorId -eq $authorId
  connectionBoundary = $sameTenantConnection.Status -eq 200 -and $sameTenantRequest.status -eq 'pending' -and $acceptedConnection.Status -eq 200 -and $selfConnection.Status -eq 400 -and $foreignConnection.Status -eq 404 -and ($connections.connectedToId -contains $peerId)
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
