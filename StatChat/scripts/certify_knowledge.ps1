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
$tenant = "knowledge-cert-$stamp"; $foreignTenant = "knowledge-foreign-$stamp"
$authorId = "knowledge-author-$stamp"; $peerId = "knowledge-peer-$stamp"; $foreignId = "knowledge-foreign-user-$stamp"
$authorToken = Token $authorId 'Knowledge Author' $tenant
$peerToken = Token $peerId 'Knowledge Peer' $tenant
$foreignToken = Token $foreignId 'Foreign Knowledge User' $foreignTenant
$null = Request GET '/users/me' $authorToken
$null = Request GET '/users/me' $peerToken
$null = Request GET '/users/me' $foreignToken

$postResponse = Request POST '/knowledge/posts' $authorToken @{title='Tenant knowledge protocol'; author='Spoofed Author'; createdBy='spoof'; category='Research'; content='Use a review owner, evidence links, and publication date on every protocol page.'}
$post = $postResponse.Body | ConvertFrom-Json
$tenantPosts = JsonArray (Request GET '/knowledge/posts' $peerToken).Body
$foreignPosts = JsonArray (Request GET '/knowledge/posts' $foreignToken).Body

$articles = JsonArray (Request GET '/knowledge/articles' $authorToken).Body
$articleWithContent = $articles | Where-Object { $_.content -and $_.excerpt -and $_.content.Length -gt $_.excerpt.Length } | Select-Object -First 1

$expertsBefore = JsonArray (Request GET '/knowledge/experts' $authorToken).Body
$expert = $expertsBefore | Select-Object -First 1
$followOne = Request POST "/knowledge/experts/$($expert.id)/follow" $authorToken
$followTwo = Request POST "/knowledge/experts/$($expert.id)/follow" $authorToken
$followOneBody = $followOne.Body | ConvertFrom-Json
$followTwoBody = $followTwo.Body | ConvertFrom-Json
$expertsAfter = JsonArray (Request GET '/knowledge/experts' $authorToken).Body
$followedExpert = $expertsAfter | Where-Object { $_.id -eq $expert.id } | Select-Object -First 1

$ideasBefore = JsonArray (Request GET '/knowledge/ideas' $authorToken).Body
$idea = $ideasBefore | Select-Object -First 1
$voteOne = Request POST "/knowledge/ideas/$($idea.id)/upvote" $authorToken
$voteTwo = Request POST "/knowledge/ideas/$($idea.id)/upvote" $authorToken
$voteOneBody = $voteOne.Body | ConvertFrom-Json
$voteTwoBody = $voteTwo.Body | ConvertFrom-Json
$ideasAfter = JsonArray (Request GET '/knowledge/ideas' $authorToken).Body
$votedIdea = $ideasAfter | Where-Object { $_.id -eq $idea.id } | Select-Object -First 1

$peerExperts = JsonArray (Request GET '/knowledge/experts' $peerToken).Body
$peerExpertState = $peerExperts | Where-Object { $_.id -eq $expert.id } | Select-Object -First 1
$peerIdeas = JsonArray (Request GET '/knowledge/ideas' $peerToken).Body
$peerIdeaState = $peerIdeas | Where-Object { $_.id -eq $idea.id } | Select-Object -First 1

$missingExpert = Request POST '/knowledge/experts/not-real/follow' $authorToken
$missingIdea = Request POST '/knowledge/ideas/not-real/upvote' $authorToken

$result = [ordered]@{
  authenticatedKnowledgePost = $postResponse.Status -eq 201 -and $post.author -eq 'Knowledge Author' -and $post.authorId -eq $authorId -and $post.createdBy -eq $authorId -and $post.org -eq $tenant
  tenantPostIsolation = ($tenantPosts.id -contains $post.id) -and -not ($foreignPosts.id -contains $post.id)
  articleContentAvailable = $null -ne $articleWithContent
  followIdempotent = $followOne.Status -eq 200 -and $followTwo.Status -eq 200 -and $followedExpert.following -eq $true -and ([int]$followTwoBody.followers -eq [int]$followOneBody.followers)
  upvoteIdempotent = $voteOne.Status -eq 200 -and $voteTwo.Status -eq 200 -and $votedIdea.upvoted -eq $true -and ([int]$voteTwoBody.votes -eq [int]$voteOneBody.votes)
  interactionStateIsPerUser = $peerExpertState.following -ne $true -and $peerIdeaState.upvoted -ne $true
  missingTargetsReturnNotFound = $missingExpert.Status -eq 404 -and $missingIdea.Status -eq 404
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
