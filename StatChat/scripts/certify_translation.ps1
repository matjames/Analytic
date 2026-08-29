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

$token = Token 'translation-cert-user' 'Translation Cert User' 'translation-cert-tenant'
$languagesResponse = Request GET '/v1/chat/translation/languages' $token
$languages = $languagesResponse.Body | ConvertFrom-Json
$swahili = $languages | Where-Object { $_.code -eq 'sw' }
$french = $languages | Where-Object { $_.code -eq 'fr' }
$translatedResponse = Request POST '/v1/chat/translation/translate' $token @{text='Hello team'; sourceLanguage='en'; targetLanguage='sw'}
$translated = $translatedResponse.Body | ConvertFrom-Json
$identityResponse = Request POST '/v1/chat/translation/translate' $token @{text='Keep this text'; sourceLanguage='en'; targetLanguage='en'}
$identity = $identityResponse.Body | ConvertFrom-Json
$unsupported = Request POST '/v1/chat/translation/translate' $token @{text='Hello'; sourceLanguage='en'; targetLanguage='xx'}
$empty = Request POST '/v1/chat/translation/translate' $token @{text=''; sourceLanguage='en'; targetLanguage='sw'}
$unauthenticated = Request GET '/v1/chat/translation/languages' ''

$result = [ordered]@{
  authenticatedLanguageDiscovery = $languagesResponse.Status -eq 200 -and $null -ne $swahili -and $null -ne $french
  glossaryTranslation = $translatedResponse.Status -eq 200 -and $translated.text -eq 'habari timu' -and $translated.provider -eq 'local-glossary'
  sameLanguageIdentity = $identityResponse.Status -eq 200 -and $identity.text -eq 'Keep this text' -and $identity.provider -eq 'identity'
  unsupportedLanguageRejected = $unsupported.Status -eq 400
  emptyTextRejected = $empty.Status -eq 400
  authenticationRequired = $unauthenticated.Status -eq 401
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
