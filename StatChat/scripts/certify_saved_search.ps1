$ErrorActionPreference = 'Stop'

$envPath = Join-Path $PSScriptRoot '..\..\.env'
$secretLine = Get-Content $envPath | Where-Object { $_ -match '^STATGATE_REGISTRY_JWT_SECRET=' } | Select-Object -First 1
$secret = ($secretLine -split '=', 2)[1].Trim()
if (-not $secret) { throw 'STATGATE_REGISTRY_JWT_SECRET is required' }

function ConvertTo-Base64Url([byte[]]$Bytes) {
    [Convert]::ToBase64String($Bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function New-CertToken($Id, $Name) {
    $header = ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
    $payload = @{
        sub = $Id; name = $Name; email = "$Id@example.test"; organizationId = 'saved-cert-tenant'
        role = 'member'; exp = [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds()
    } | ConvertTo-Json -Compress
    $body = "$header.$(ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes($payload)))"
    $hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
    "$body.$(ConvertTo-Base64Url ($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))))"
}

function Invoke-CertRequest($Method, $Path, $Token, $Body = $null) {
    try {
        $arguments = @{ Uri = "http://localhost:4000$Path"; Method = $Method; Headers = @{ Authorization = "Bearer $Token" }; UseBasicParsing = $true }
        if ($null -ne $Body) {
            $arguments.ContentType = 'application/json'
            $arguments.Body = $Body | ConvertTo-Json -Depth 8 -Compress
        }
        $response = Invoke-WebRequest @arguments
        [pscustomobject]@{ Status = [int]$response.StatusCode; Body = $response.Content }
    } catch {
        $status = [int]$_.Exception.Response.StatusCode
        $reader = [IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
        [pscustomobject]@{ Status = $status; Body = $reader.ReadToEnd() }
    }
}

function ConvertTo-Array($Response) {
    if ([string]::IsNullOrWhiteSpace($Response.Body)) { return @() }
    @($Response.Body | ConvertFrom-Json)
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$keyword = "saved-cert-$stamp"
$ownerId = "save-owner-$stamp"
$memberId = "save-member-$stamp"
$outsiderId = "save-outsider-$stamp"
$owner = New-CertToken $ownerId 'Saved Owner'
$member = New-CertToken $memberId 'Saved Member'
$outsider = New-CertToken $outsiderId 'Saved Outsider'

$source = Invoke-CertRequest POST '/conversations/group' $owner @{ groupId = "saved-source-$stamp"; name = 'Saved source'; memberIds = @($ownerId, $memberId) }
$sourceId = ($source.Body | ConvertFrom-Json).id
$other = Invoke-CertRequest POST '/conversations/group' $owner @{ groupId = "saved-other-$stamp"; name = 'Saved other'; memberIds = @($ownerId, $memberId) }
$otherId = ($other.Body | ConvertFrom-Json).id
$textSend = Invoke-CertRequest POST '/v1/chat/messages' $member @{ conversationId = $sourceId; text = "$keyword primary report" }
$message = $textSend.Body | ConvertFrom-Json
[void](Invoke-CertRequest POST '/v1/chat/messages' $owner @{ conversationId = $otherId; text = "$keyword secondary report" })
$mediaSend = Invoke-CertRequest POST '/v1/chat/media/send' $member @{ conversationId = $sourceId; mediaId = 'approved' }
$mediaMessage = $mediaSend.Body | ConvertFrom-Json

$saveOwner = Invoke-CertRequest POST "/v1/chat/messages/$($message.id)/saved" $owner
$saveOwnerAgain = Invoke-CertRequest POST "/v1/chat/messages/$($message.id)/saved" $owner
$saveMember = Invoke-CertRequest POST "/v1/chat/messages/$($message.id)/saved" $member
$ownerList = ConvertTo-Array (Invoke-CertRequest GET '/v1/chat/saved' $owner)
$outsiderList = ConvertTo-Array (Invoke-CertRequest GET '/v1/chat/saved' $outsider)
$outsiderSave = Invoke-CertRequest POST "/v1/chat/messages/$($message.id)/saved" $outsider

$savedMessages = @(((Invoke-CertRequest GET "/v1/search?savedOnly=true&q=$keyword" $owner).Body | ConvertFrom-Json).messages)
$outsiderSavedMessages = @(((Invoke-CertRequest GET "/v1/search?savedOnly=true&q=$keyword" $outsider).Body | ConvertFrom-Json).messages)
$conversationMessages = @(((Invoke-CertRequest GET "/v1/search?q=$keyword&conversationId=$sourceId" $owner).Body | ConvertFrom-Json).messages)
$sender = [uri]::EscapeDataString('Saved Member')
$senderMessages = @(((Invoke-CertRequest GET "/v1/search?q=$keyword&sender=$sender" $owner).Body | ConvertFrom-Json).messages)
$today = [DateTime]::UtcNow.ToString('yyyy-MM-dd')
$dateMessages = @(((Invoke-CertRequest GET "/v1/search?q=$keyword&from=$today&to=$today" $owner).Body | ConvertFrom-Json).messages)
$futureMessages = @(((Invoke-CertRequest GET "/v1/search?q=$keyword&from=2099-01-01" $owner).Body | ConvertFrom-Json).messages)
$attachmentMessages = @(((Invoke-CertRequest GET "/v1/search?conversationId=$sourceId&hasAttachment=true" $owner).Body | ConvertFrom-Json).messages)
$withoutAttachment = @(((Invoke-CertRequest GET "/v1/search?conversationId=$sourceId&hasAttachment=false&q=$keyword" $owner).Body | ConvertFrom-Json).messages)
$badDate = Invoke-CertRequest GET '/v1/search?from=bad-date' $owner
$badBoolean = Invoke-CertRequest GET '/v1/search?hasAttachment=perhaps' $owner
$inaccessible = Invoke-CertRequest GET "/v1/search?conversationId=$sourceId" $outsider

$unsaveMember = Invoke-CertRequest DELETE "/v1/chat/messages/$($message.id)/saved" $member
$ownerAfterMember = ConvertTo-Array (Invoke-CertRequest GET '/v1/chat/saved' $owner)
[void](Invoke-CertRequest POST "/v1/chat/messages/$($message.id)/saved" $member)
$removeMember = Invoke-CertRequest DELETE "/v1/chat/conversations/$sourceId/members/$memberId" $owner
$memberAfterRemoval = ConvertTo-Array (Invoke-CertRequest GET '/v1/chat/saved' $member)
$unsaveOwner = Invoke-CertRequest DELETE "/v1/chat/messages/$($message.id)/saved" $owner
$ownerFinal = ConvertTo-Array (Invoke-CertRequest GET '/v1/chat/saved' $owner)

$result = [ordered]@{
    groupCreate = $source.Status -eq 200 -and $other.Status -eq 200
    sends = $textSend.Status -eq 200 -and $mediaSend.Status -eq 201
    privateIdempotentSaves = $saveOwner.Status -eq 201 -and $saveOwnerAgain.Status -eq 201 -and $saveMember.Status -eq 201
    privateLists = $ownerList.id -contains $message.id -and -not ($outsiderList.id -contains $message.id)
    outsiderSaveDenied = $outsiderSave.Status -eq 403
    savedOnlyIsolation = $savedMessages.id -contains $message.id -and -not ($outsiderSavedMessages.id -contains $message.id)
    conversationFilter = $conversationMessages.Count -eq 1 -and $conversationMessages[0].conversationId -eq $sourceId
    senderFilter = $senderMessages.id -contains $message.id
    dateFilters = $dateMessages.id -contains $message.id -and $futureMessages.Count -eq 0
    attachmentFilters = $attachmentMessages.id -contains $mediaMessage.id -and $withoutAttachment.id -contains $message.id
    invalidFiltersRejected = $badDate.Status -eq 400 -and $badBoolean.Status -eq 400
    inaccessibleConversationDenied = $inaccessible.Status -eq 403
    independentUnsave = $unsaveMember.Status -eq 204 -and $ownerAfterMember.id -contains $message.id
    accessRevocation = $removeMember.Status -eq 204 -and -not ($memberAfterRemoval.id -contains $message.id)
    ownerUnsave = $unsaveOwner.Status -eq 204 -and -not ($ownerFinal.id -contains $message.id)
}

$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
