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
  } catch { [pscustomobject]@{Status=[int]($_.Exception.Response.StatusCode.value__); Body=''} }
}
function RequestNoAuth($method, $path) {
  $status = curl.exe --silent --output NUL --write-out '%{http_code}' --request $method "$baseUrl$path"
  [pscustomobject]@{Status=[int]$status; Body=''}
}
function JsonArray($body) {
  $value = $body | ConvertFrom-Json
  if ($value.PSObject.Properties.Name -contains 'value') { return @($value.value) }
  return @($value)
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "wellness-cert-$stamp"; $foreignTenant = "wellness-foreign-$stamp"
$aliceId = "wellness-alice-$stamp"; $bobId = "wellness-bob-$stamp"; $foreignId = "wellness-foreign-$stamp"
$aliceToken = Token $aliceId 'Alice Wellness' $tenant
$bobToken = Token $bobId 'Bob Wellness' $tenant
$foreignToken = Token $foreignId 'Foreign Wellness' $foreignTenant
$null = Request GET '/users/me' $aliceToken
$null = Request GET '/users/me' $bobToken
$null = Request GET '/users/me' $foreignToken

$aliceInitial = JsonArray (Request GET '/wellness/posts' $aliceToken).Body
$foreignInitial = JsonArray (Request GET '/wellness/posts' $foreignToken).Body
$create = Request POST '/wellness/posts' $aliceToken @{author='Spoofed Author'; handle='@spoof'; category='Mindfulness'; text='A tenant-safe wellness note'; tags=@('#Wellness')}
$created = $create.Body | ConvertFrom-Json
$bobFeed = JsonArray (Request GET '/wellness/posts' $bobToken).Body
$foreignFeed = JsonArray (Request GET '/wellness/posts' $foreignToken).Body

$likeOne = Request POST "/wellness/posts/$($created.id)/like" $aliceToken
$likeTwo = Request POST "/wellness/posts/$($created.id)/like" $aliceToken
$bookmarkOne = Request POST "/wellness/posts/$($created.id)/bookmark" $aliceToken
$bookmarkTwo = Request POST "/wellness/posts/$($created.id)/bookmark" $aliceToken
$share = Request POST "/wellness/posts/$($created.id)/share" $bobToken
$comment = Request POST "/wellness/posts/$($created.id)/comments" $bobToken @{text='Thank you for sharing this.'}
$comments = JsonArray (Request GET "/wellness/posts/$($created.id)/comments" $aliceToken).Body
$foreignTarget = Request POST "/wellness/posts/$($created.id)/like" $foreignToken
$unauthenticated = RequestNoAuth GET '/wellness/posts'

$result = [ordered]@{
  seededContentVisible = ($aliceInitial.id -contains 'w1') -and ($foreignInitial.id -contains 'w1')
  authenticatedAuthorAndTenant = $create.Status -eq 200 -and $created.author -eq 'Alice Wellness' -and $created.authorId -eq $aliceId -and $created.tenantId -eq $tenant
  sameTenantReadAndForeignIsolation = ($bobFeed.id -contains $created.id) -and -not ($foreignFeed.id -contains $created.id)
  idempotentLike = (($likeOne.Body | ConvertFrom-Json).liked -eq $true) -and (($likeTwo.Body | ConvertFrom-Json).liked -eq $false)
  idempotentBookmark = (($bookmarkOne.Body | ConvertFrom-Json).bookmarked -eq $true) -and (($bookmarkTwo.Body | ConvertFrom-Json).bookmarked -eq $false)
  durableShareAndComment = $share.Status -eq 200 -and (($share.Body | ConvertFrom-Json).shares -eq 1) -and $comment.Status -eq 200 -and ($comments.author -contains 'Bob Wellness')
  boundaryAndAuthentication = $foreignTarget.Status -eq 404 -and $unauthenticated.Status -eq 401
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
