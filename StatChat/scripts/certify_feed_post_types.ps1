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
function JsonArray($body) {
  $value = $body | ConvertFrom-Json
  if ($value.PSObject.Properties.Name -contains 'value') { return @($value.value) }
  return @($value)
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "feed-types-$stamp"; $foreignTenant = "feed-types-foreign-$stamp"
$aliceId = "feed-types-alice-$stamp"; $bobId = "feed-types-bob-$stamp"; $foreignId = "feed-types-foreign-$stamp"
$aliceToken = Token $aliceId 'Alice Feed' $tenant
$bobToken = Token $bobId 'Bob Feed' $tenant
$foreignToken = Token $foreignId 'Foreign Feed' $foreignTenant
$null = Request GET '/users/me' $aliceToken
$null = Request GET '/users/me' $bobToken
$null = Request GET '/users/me' $foreignToken

$article = Request POST '/collaboration/posts' $aliceToken @{author='Spoofed'; type='article'; title='A useful field note'; text='Article body'}
$photo = Request POST '/collaboration/posts' $aliceToken @{type='photo'; mediaUrl='https://example.test/photo.jpg'; text='Photo caption'}
$video = Request POST '/collaboration/posts' $aliceToken @{type='video'; mediaUrl='https://example.test/video.mp4'; text='Video caption'}
$missingMedia = Request POST '/collaboration/posts' $aliceToken @{type='photo'; text='Missing media'}
$missingTitle = Request POST '/collaboration/posts' $aliceToken @{type='article'; text='Missing title'}
$invalidType = Request POST '/collaboration/posts' $aliceToken @{type='unknown'; text='Invalid type'}
$sameTenant = JsonArray (Request GET '/collaboration/posts' $bobToken).Body
$foreign = JsonArray (Request GET '/collaboration/posts' $foreignToken).Body

$articleBody = $article.Body | ConvertFrom-Json
$photoBody = $photo.Body | ConvertFrom-Json
$videoBody = $video.Body | ConvertFrom-Json
$result = [ordered]@{
  articleIdentityAndMetadata = $article.Status -eq 200 -and $articleBody.author -eq 'Alice Feed' -and $articleBody.authorId -eq $aliceId -and $articleBody.type -eq 'article' -and $articleBody.title -eq 'A useful field note'
  photoPost = $photo.Status -eq 200 -and $photoBody.type -eq 'photo' -and $photoBody.mediaUrl -eq 'https://example.test/photo.jpg'
  videoPost = $video.Status -eq 200 -and $videoBody.type -eq 'video' -and $videoBody.mediaUrl -eq 'https://example.test/video.mp4'
  validation = $missingMedia.Status -eq 400 -and $missingTitle.Status -eq 400 -and $invalidType.Status -eq 400
  tenantIsolation = ($sameTenant.id -contains $articleBody.id) -and ($sameTenant.id -contains $photoBody.id) -and ($sameTenant.id -contains $videoBody.id) -and -not ($foreign.id -contains $articleBody.id)
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
