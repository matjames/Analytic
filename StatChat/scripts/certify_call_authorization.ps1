$ErrorActionPreference = 'Stop'

$envPath = Join-Path $PSScriptRoot '..\..\.env'
$secretLine = Get-Content $envPath | Where-Object { $_ -match '^STATGATE_REGISTRY_JWT_SECRET=' } | Select-Object -First 1
$secret = ($secretLine -split '=', 2)[1].Trim()
if (-not $secret) { throw 'STATGATE_REGISTRY_JWT_SECRET is required' }

function ConvertTo-Base64Url([byte[]]$Bytes) {
    [Convert]::ToBase64String($Bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function New-CertToken($Id, $Name, $Tenant) {
    $header = ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
    $payload = @{ sub = $Id; name = $Name; email = "$Id@example.test"; organizationId = $Tenant; role = 'member'; exp = [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() } | ConvertTo-Json -Compress
    $body = "$header.$(ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes($payload)))"
    $hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
    "$body.$(ConvertTo-Base64Url ($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))))"
}

function Invoke-CertRequest($Method, $Path, $Token, $Body = $null) {
    try {
        $arguments = @{ Uri = "http://localhost:4000$Path"; Method = $Method; Headers = @{ Authorization = "Bearer $Token" }; UseBasicParsing = $true }
        if ($null -ne $Body) { $arguments.ContentType = 'application/json'; $arguments.Body = $Body | ConvertTo-Json -Depth 6 -Compress }
        $response = Invoke-WebRequest @arguments
        [pscustomobject]@{ Status = [int]$response.StatusCode; Body = $response.Content }
    } catch {
        $status = [int]$_.Exception.Response.StatusCode
        $reader = [IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
        [pscustomobject]@{ Status = $status; Body = $reader.ReadToEnd() }
    }
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "call-cert-$stamp"
$foreignTenant = "call-foreign-$stamp"
$hostId = "call-host-$stamp"
$memberId = "call-member-$stamp"
$assistantId = "call-assistant-$stamp"
$outsiderId = "call-outsider-$stamp"
$foreignId = "call-foreign-user-$stamp"
$hostToken = New-CertToken $hostId 'Call Host' $tenant
$member = New-CertToken $memberId 'Call Member' $tenant
$assistant = New-CertToken $assistantId 'Call Assistant' $tenant
$outsider = New-CertToken $outsiderId 'Call Outsider' $tenant
$foreign = New-CertToken $foreignId 'Foreign User' $foreignTenant

$group = Invoke-CertRequest POST '/conversations/group' $hostToken @{ groupId = "call-group-$stamp"; name = 'Call authorization'; memberIds = @($hostId, $memberId, $assistantId) }
$conversationId = ($group.Body | ConvertFrom-Json).id
$call = Invoke-CertRequest POST '/v1/calls' $hostToken @{ kind = 'video'; roomName = 'Certified call'; conversationId = $conversationId }
$sessionId = ($call.Body | ConvertFrom-Json).id

$outsiderCreate = Invoke-CertRequest POST '/v1/calls' $outsider @{ kind = 'video'; roomName = 'Intrusion'; conversationId = $conversationId }
$memberJoin = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $member @{ userId = $hostId; userName = 'Spoofed Host' }
$assistantJoin = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $assistant @{}
$outsiderJoin = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $outsider @{}
$foreignJoin = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $foreign @{}
$memberDetail = Invoke-CertRequest GET "/v1/calls/$sessionId" $member
$outsiderDetail = Invoke-CertRequest GET "/v1/calls/$sessionId" $outsider
$foreignDetail = Invoke-CertRequest GET "/v1/calls/$sessionId" $foreign
$memberList = @((Invoke-CertRequest GET '/v1/calls' $member).Body | ConvertFrom-Json)
$outsiderList = @((Invoke-CertRequest GET '/v1/calls' $outsider).Body | ConvertFrom-Json)
$foreignList = @((Invoke-CertRequest GET '/v1/calls' $foreign).Body | ConvertFrom-Json)
$outsiderParticipants = Invoke-CertRequest GET "/v1/calls/$sessionId/participants" $outsider
$outsiderRecordings = Invoke-CertRequest GET "/v1/calls/$sessionId/recordings" $outsider
$memberEnd = Invoke-CertRequest POST "/v1/calls/$sessionId/end" $member
$memberRemove = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$hostId/remove" $member
$memberRoleDenied = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$assistantId/role" $member @{ role = 'moderator' }
$hostPromoteMember = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$memberId/role" $hostToken @{ role = 'moderator' }
$moderatorRemoveAssistant = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$assistantId/remove" $member
$moderatorRemoveHost = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$hostId/remove" $member
$qualityReport = Invoke-CertRequest POST "/v1/calls/$sessionId/quality" $member @{ rttMs = 420; jitterMs = 70; packetLossPct = 5; bitrateKbps = 450; quality = 'excellent'; userId = $hostId }
$invalidQuality = Invoke-CertRequest POST "/v1/calls/$sessionId/quality" $member @{ rttMs = 40; jitterMs = 5; packetLossPct = 101; bitrateKbps = 900 }
$outsiderQualityWrite = Invoke-CertRequest POST "/v1/calls/$sessionId/quality" $outsider @{ rttMs = 40; jitterMs = 5; packetLossPct = 0; bitrateKbps = 900 }
$foreignQualityRead = Invoke-CertRequest GET "/v1/calls/$sessionId/quality" $foreign
$qualityRead = @((Invoke-CertRequest GET "/v1/calls/$sessionId/quality" $member).Body | ConvertFrom-Json)

$hostRemoveMember = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$memberId/remove" $hostToken
$hostRemoveHost = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$hostId/remove" $hostToken
$qualityAfterRemoval = Invoke-CertRequest POST "/v1/calls/$sessionId/quality" $member @{ rttMs = 40; jitterMs = 5; packetLossPct = 0; bitrateKbps = 900 }
$memberLeave = Invoke-CertRequest POST "/v1/calls/$sessionId/leave" $member @{ userId = $hostId }
$qualityAfterLeave = Invoke-CertRequest POST "/v1/calls/$sessionId/quality" $member @{ rttMs = 40; jitterMs = 5; packetLossPct = 0; bitrateKbps = 900 }
$participantsAfterLeave = @((Invoke-CertRequest GET "/v1/calls/$sessionId/participants" $hostToken).Body | ConvertFrom-Json)
$hostEnd = Invoke-CertRequest POST "/v1/calls/$sessionId/end" $hostToken

$joined = $memberJoin.Body | ConvertFrom-Json
$result = [ordered]@{
    setup = $group.Status -eq 200 -and $call.Status -eq 200 -and $assistantJoin.Status -eq 200
    createRequiresConversationAccess = $outsiderCreate.Status -eq 403
    authenticatedJoinIdentity = $memberJoin.Status -eq 200 -and $joined.userId -eq $memberId -and $joined.userName -eq 'Call Member'
    joinIsolation = $outsiderJoin.Status -eq 403 -and $foreignJoin.Status -eq 403
    detailIsolation = $memberDetail.Status -eq 200 -and $outsiderDetail.Status -eq 403 -and $foreignDetail.Status -eq 403
    discoveryIsolation = $memberList.id -contains $sessionId -and -not ($outsiderList.id -contains $sessionId) -and -not ($foreignList.id -contains $sessionId)
    relatedDataIsolation = $outsiderParticipants.Status -eq 403 -and $outsiderRecordings.Status -eq 403
    hostOnlyTermination = $memberEnd.Status -eq 403 -and $hostEnd.Status -eq 200
    moderationBoundary = $memberRemove.Status -eq 403 -and $memberRoleDenied.Status -eq 403 -and $hostPromoteMember.Status -eq 200 -and $moderatorRemoveAssistant.Status -eq 200 -and $moderatorRemoveHost.Status -eq 409 -and $hostRemoveMember.Status -eq 200 -and $hostRemoveHost.Status -eq 409
    leaveCannotSpoof = $memberLeave.Status -eq 204 -and $participantsAfterLeave.userId -contains $hostId -and -not ($participantsAfterLeave.userId -contains $memberId)
    qualityWriteBoundary = $qualityReport.Status -eq 200 -and $outsiderQualityWrite.Status -eq 403 -and $qualityAfterRemoval.Status -eq 403 -and $qualityAfterLeave.Status -eq 403
    qualityValidation = $invalidQuality.Status -eq 400
    qualityReadIsolation = $foreignQualityRead.Status -eq 403
    serverClassifiedTelemetry = $qualityRead.Count -eq 1 -and $qualityRead[0].userId -eq $memberId -and $qualityRead[0].quality -eq 'fair' -and $qualityRead[0].rttMs -eq 420
}

$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
