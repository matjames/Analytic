$ErrorActionPreference = 'Stop'

$envPath = Join-Path $PSScriptRoot '..\..\.env'
$secretLine = Get-Content $envPath | Where-Object { $_ -match '^STATGATE_REGISTRY_JWT_SECRET=' } | Select-Object -First 1
$secret = ($secretLine -split '=', 2)[1].Trim()
if (-not $secret) { throw 'STATGATE_REGISTRY_JWT_SECRET is required' }

function ConvertTo-Base64Url([byte[]]$Bytes) { [Convert]::ToBase64String($Bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_') }
function New-CertToken($Id, $Name, $Tenant) {
    $header = ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
    $payload = @{ sub = $Id; name = $Name; email = "$Id@example.test"; organizationId = $Tenant; role = 'member'; exp = [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds() } | ConvertTo-Json -Compress
    $body = "$header.$(ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes($payload)))"
    $hmac = [Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($secret))
    "$body.$(ConvertTo-Base64Url ($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($body))))"
}
function Invoke-CertRequest($Method, $Path, $Token, $Body = $null) {
    $args = @{ Uri = "http://localhost:4000$Path"; Method = $Method; Headers = @{ Authorization = "Bearer $Token" }; UseBasicParsing = $true }
    if ($null -ne $Body) { $args.ContentType = 'application/json'; $args.Body = $Body | ConvertTo-Json -Depth 6 -Compress }
    try { $response = Invoke-WebRequest @args; [pscustomobject]@{ Status = [int]$response.StatusCode; Body = $response.Content } }
    catch { [pscustomobject]@{ Status = [int]$_.Exception.Response.StatusCode; Body = '' } }
}

function New-CallSocket($Token) {
    $socket = [Net.WebSockets.ClientWebSocket]::new()
    $socket.Options.AddSubprotocol("Bearer.$Token")
    [void]$socket.ConnectAsync([Uri]'ws://localhost:4000/ws', [Threading.CancellationToken]::None).GetAwaiter().GetResult()
    return $socket
}
function Send-CallSocket($Socket, $Payload) {
    $bytes = [Text.Encoding]::UTF8.GetBytes(($Payload | ConvertTo-Json -Compress))
    [void]$Socket.SendAsync([ArraySegment[byte]]::new($bytes), [Net.WebSockets.WebSocketMessageType]::Text, $true, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
}
function Receive-CallSocket($Socket, $WantedEvent, $TimeoutMs = 3000) {
    $buffer = New-Object byte[] 8192
    $cancel = [Threading.CancellationTokenSource]::new($TimeoutMs)
    try {
        while ($true) {
            $text = [Text.StringBuilder]::new()
            do {
                $result = $Socket.ReceiveAsync([ArraySegment[byte]]::new($buffer), $cancel.Token).GetAwaiter().GetResult()
                [void]$text.Append([Text.Encoding]::UTF8.GetString($buffer, 0, $result.Count))
            } while (-not $result.EndOfMessage)
            $message = $text.ToString() | ConvertFrom-Json
            if (-not $WantedEvent -or $message.event -eq $WantedEvent) { return $message }
        }
    } catch { return $null } finally { $cancel.Dispose() }
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "mute-cert-$stamp"
$hostId = "mute-host-$stamp"; $memberId = "mute-moderator-$stamp"; $participantId = "mute-participant-$stamp"
$hostToken = New-CertToken $hostId 'Mute Host' $tenant
$memberToken = New-CertToken $memberId 'Mute Moderator' $tenant
$participantToken = New-CertToken $participantId 'Mute Participant' $tenant
$group = Invoke-CertRequest POST '/conversations/group' $hostToken @{ groupId = "mute-group-$stamp"; name = 'Mute certification'; memberIds = @($hostId, $memberId, $participantId) }
$conversationId = ($group.Body | ConvertFrom-Json).id
$call = Invoke-CertRequest POST '/v1/calls' $hostToken @{ kind = 'voice'; roomName = 'Mute certification'; conversationId = $conversationId }
$sessionId = ($call.Body | ConvertFrom-Json).id
$null = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $memberToken @{}
$null = Invoke-CertRequest POST "/v1/calls/$sessionId/join" $participantToken @{}
$role = Invoke-CertRequest POST "/v1/calls/$sessionId/participants/$memberId/role" $hostToken @{ role = 'moderator' }

$hostSocket = New-CallSocket $hostToken; $memberSocket = New-CallSocket $memberToken; $participantSocket = New-CallSocket $participantToken
try {
    Send-CallSocket $hostSocket @{ action = 'join-call'; sessionId = $sessionId }
    Send-CallSocket $memberSocket @{ action = 'join-call'; sessionId = $sessionId }
    Send-CallSocket $participantSocket @{ action = 'join-call'; sessionId = $sessionId }
    Start-Sleep -Milliseconds 200

    Send-CallSocket $participantSocket @{ action = 'signal'; sessionId = $sessionId; type = 'mute-requested'; to = $hostId }
    $ordinaryDenied = Receive-CallSocket $participantSocket 'error'
    Send-CallSocket $memberSocket @{ action = 'signal'; sessionId = $sessionId; type = 'mute-requested'; to = $participantId }
    $requestReceived = Receive-CallSocket $participantSocket 'call-signal'
    Send-CallSocket $participantSocket @{ action = 'signal'; sessionId = $sessionId; type = 'mute-accepted'; to = $memberId }
    $consentReceived = Receive-CallSocket $memberSocket 'call-signal'
} finally {
    $hostSocket.Dispose(); $memberSocket.Dispose(); $participantSocket.Dispose()
}

$result = [ordered]@{
    setup = $group.Status -eq 200 -and $call.Status -eq 200 -and $role.Status -eq 200
    ordinaryParticipantCannotRequestMute = $ordinaryDenied -ne $null -and $ordinaryDenied.error -eq 'only hosts and moderators can request mute'
    moderatorRequestDelivered = $requestReceived -ne $null -and $requestReceived.payload.type -eq 'mute-requested' -and $requestReceived.payload.from -eq $memberId
    recipientConsentDelivered = $consentReceived -ne $null -and $consentReceived.payload.type -eq 'mute-accepted' -and $consentReceived.payload.from -eq $participantId
}
$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
