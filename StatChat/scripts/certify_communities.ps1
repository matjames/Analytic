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
$tenant = "community-cert-$stamp"; $foreignTenant = "community-foreign-$stamp"
$ownerId = "community-owner-$stamp"; $memberId = "community-member-$stamp"; $secondMemberId = "community-second-member-$stamp"; $foreignId = "community-foreign-user-$stamp"
$ownerToken = Token $ownerId 'Community Owner' $tenant
$memberToken = Token $memberId 'Community Member' $tenant
$secondMemberToken = Token $secondMemberId 'Second Community Member' $tenant
$foreignToken = Token $foreignId 'Foreign Member' $foreignTenant
$null = Request GET '/users/me' $ownerToken
$null = Request GET '/users/me' $memberToken
$null = Request GET '/users/me' $secondMemberToken
$null = Request GET '/users/me' $foreignToken

$publicCommunityResponse = Request POST '/v1/chat/communities' $ownerToken @{name="Field Methods $stamp"; description='Practice group for field protocols'; visibility='public'}
$privateCommunityResponse = Request POST '/v1/chat/communities' $ownerToken @{name="Private Methods $stamp"; description='Invite-only planning'; visibility='private'}
$publicCommunity = $publicCommunityResponse.Body | ConvertFrom-Json
$privateCommunity = $privateCommunityResponse.Body | ConvertFrom-Json

$tenantCommunities = @((Request GET '/v1/chat/communities' $memberToken).Body | ConvertFrom-Json)
$foreignCommunities = @((Request GET '/v1/chat/communities' $foreignToken).Body | ConvertFrom-Json)
$joinPublic = Request POST "/v1/chat/communities/$($publicCommunity.id)/join" $memberToken
$joinPrivate = Request POST "/v1/chat/communities/$($privateCommunity.id)/join" $memberToken
$topicBeforeJoin = Request POST "/v1/chat/communities/$($privateCommunity.id)/topics" $memberToken @{title='Should not post'; body='No membership'}
$ownerInvitePrivate = Request POST "/v1/chat/communities/$($privateCommunity.id)/members" $ownerToken @{userId=$memberId}
$privateMembers = @((Request GET "/v1/chat/communities/$($privateCommunity.id)/members" $memberToken).Body | ConvertFrom-Json)
$memberAddAttempt = Request POST "/v1/chat/communities/$($privateCommunity.id)/members" $memberToken @{userId=$secondMemberId}
$foreignInvite = Request POST "/v1/chat/communities/$($privateCommunity.id)/members" $ownerToken @{userId=$foreignId}
$removeOwner = Request DELETE "/v1/chat/communities/$($privateCommunity.id)/members/$ownerId" $ownerToken
$topicResponse = Request POST "/v1/chat/communities/$($publicCommunity.id)/topics" $memberToken @{title='How should we validate field enumerator checklists?'; body='Please share county-level practices.'}
$topic = $topicResponse.Body | ConvertFrom-Json
$foreignTopic = Request POST "/v1/chat/communities/$($publicCommunity.id)/topics" $foreignToken @{title='Foreign write'; body='No tenant access'}
$replyResponse = Request POST "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies" $ownerToken @{body='Use pilot observations plus supervisor sign-off.'}
$reply = $replyResponse.Body | ConvertFrom-Json
$memberReplyResponse = Request POST "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies" $memberToken @{body='We can pilot this in two counties.'}
$memberReply = $memberReplyResponse.Body | ConvertFrom-Json
$memberDeletesOwnerReply = Request DELETE "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies/$($reply.id)" $memberToken
$ownerDeletesMemberReply = Request DELETE "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies/$($memberReply.id)" $ownerToken
$ownerAddsSecondMember = Request POST "/v1/chat/communities/$($privateCommunity.id)/members" $ownerToken @{userId=$secondMemberId}
$transferOwner = Request POST "/v1/chat/communities/$($privateCommunity.id)/owner" $ownerToken @{userId=$secondMemberId}
$formerOwnerAddAttempt = Request POST "/v1/chat/communities/$($privateCommunity.id)/members" $ownerToken @{userId=$foreignId}
$newOwnerMembers = @((Request GET "/v1/chat/communities/$($privateCommunity.id)/members" $secondMemberToken).Body | ConvertFrom-Json)
$foreignReplies = Request GET "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies" $foreignToken
$topics = @((Request GET "/v1/chat/communities/$($publicCommunity.id)/topics" $memberToken).Body | ConvertFrom-Json)
$replies = @((Request GET "/v1/chat/communities/$($publicCommunity.id)/topics/$($topic.id)/replies" $memberToken).Body | ConvertFrom-Json)

$result = [ordered]@{
  publicCommunityCreated = $publicCommunityResponse.Status -eq 201 -and $publicCommunity.createdBy -eq $ownerId -and $publicCommunity.joined -eq $true
  privateCommunityScoped = ($tenantCommunities.id -contains $publicCommunity.id) -and -not ($tenantCommunities.id -contains $privateCommunity.id) -and -not ($foreignCommunities.id -contains $publicCommunity.id)
  membershipBoundary = $joinPublic.Status -eq 200 -and $joinPrivate.Status -eq 404 -and $topicBeforeJoin.Status -eq 404
  privateInvite = $ownerInvitePrivate.Status -eq 200 -and ($privateMembers.userId -contains $memberId)
  ownerOnlyMembership = $memberAddAttempt.Status -eq 403 -and $foreignInvite.Status -eq 404 -and $removeOwner.Status -eq 409
  authenticatedTopic = $topicResponse.Status -eq 201 -and $topic.authorId -eq $memberId -and $topic.author -eq 'Community Member'
  tenantWriteIsolation = $foreignTopic.Status -eq 404 -and $foreignReplies.Status -eq 404
  authenticatedReply = $replyResponse.Status -eq 201 -and $reply.authorId -eq $ownerId -and ($replies.id -contains $reply.id)
  moderationBoundary = $memberDeletesOwnerReply.Status -eq 403 -and $ownerDeletesMemberReply.Status -eq 204
  ownerTransfer = $ownerAddsSecondMember.Status -eq 200 -and $transferOwner.Status -eq 204 -and $formerOwnerAddAttempt.Status -eq 403 -and (($newOwnerMembers | Where-Object { $_.userId -eq $secondMemberId }).role -eq 'owner')
  topicReadBack = ($topics.id -contains $topic.id) -and (($topics | Where-Object { $_.id -eq $topic.id }).replyCount -ge 1)
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
