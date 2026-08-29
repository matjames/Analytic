$ErrorActionPreference = 'Stop'

$rootEnv = Join-Path $PSScriptRoot '..\..\.env'
$localEnv = Join-Path $PSScriptRoot '..\.env'
function Read-EnvValue($Name) {
    foreach ($path in @($localEnv, $rootEnv)) {
        $line = Get-Content $path | Where-Object { $_ -match "^$Name=" } | Select-Object -First 1
        if ($line) {
            $value = ($line -split '=', 2)[1].Trim()
            if ($value) { return $value }
        }
    }
    return ''
}

$secret = Read-EnvValue 'STATGATE_REGISTRY_JWT_SECRET'
if (-not $secret) { $secret = Read-EnvValue 'STATCHAT_JWT_SECRET' }
if (-not $secret) { throw 'A StatChat JWT secret is required' }

function ConvertTo-Base64Url([byte[]]$Bytes) {
    [Convert]::ToBase64String($Bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function New-CertToken($Id, $Name, $Role, $Tenant) {
    $header = ConvertTo-Base64Url ([Text.Encoding]::UTF8.GetBytes('{"alg":"HS256","typ":"JWT"}'))
    $payload = @{
        sub = $Id; name = $Name; email = "$Id@example.test"; organizationId = $Tenant
        role = $Role; exp = [DateTimeOffset]::UtcNow.AddHours(1).ToUnixTimeSeconds()
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
        [pscustomobject]@{ Status = [int]$response.StatusCode; Body = $response.Content; Headers = $response.Headers }
    } catch {
        $status = [int]$_.Exception.Response.StatusCode
        $reader = [IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
        [pscustomobject]@{ Status = $status; Body = $reader.ReadToEnd(); Headers = @{} }
    }
}

function Response-Array($Response) {
    if ([string]::IsNullOrWhiteSpace($Response.Body)) { return @() }
    @($Response.Body | ConvertFrom-Json)
}

$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
$tenant = "compliance-cert-$stamp"
$adminId = "compliance-admin-$stamp"
$memberId = "compliance-member-$stamp"
$outsiderId = "compliance-outsider-$stamp"
$admin = New-CertToken $adminId 'Compliance Admin' 'tenant_admin' $tenant
$member = New-CertToken $memberId 'Compliance Member' 'member' $tenant
$outsider = New-CertToken $outsiderId 'Compliance Outsider' 'member' $tenant

$defaultPolicy = Invoke-CertRequest GET '/v1/compliance/retention' $admin
$memberPolicy = Invoke-CertRequest GET '/v1/compliance/retention' $member
$invalidPolicy = Invoke-CertRequest PUT '/v1/compliance/retention' $admin @{ retentionDays = 0; enabled = $true }

$heldConversation = Invoke-CertRequest POST '/conversations/group' $admin @{ groupId = "held-$stamp"; name = 'Held certification'; memberIds = @($adminId, $memberId) }
$purgeConversation = Invoke-CertRequest POST '/conversations/group' $admin @{ groupId = "purge-$stamp"; name = 'Purge certification'; memberIds = @($adminId, $memberId) }
$exportConversation = Invoke-CertRequest POST '/conversations/group' $admin @{ groupId = "export-$stamp"; name = 'Export certification'; memberIds = @($adminId, $memberId) }
$heldId = ($heldConversation.Body | ConvertFrom-Json).id
$purgeId = ($purgeConversation.Body | ConvertFrom-Json).id
$exportId = ($exportConversation.Body | ConvertFrom-Json).id

$heldMessage = Invoke-CertRequest POST '/v1/chat/messages' $member @{ conversationId = $heldId; text = 'Aged and held evidence' }
$purgeMessage = Invoke-CertRequest POST '/v1/chat/messages' $member @{ conversationId = $purgeId; text = 'Aged and purgeable evidence' }
$exportMessage = Invoke-CertRequest POST '/v1/chat/messages' $member @{ conversationId = $exportId; text = '=formula-safe export evidence' }
$heldMessageId = ($heldMessage.Body | ConvertFrom-Json).id
$purgeMessageId = ($purgeMessage.Body | ConvertFrom-Json).id

$memberJSON = Invoke-CertRequest GET "/v1/chat/conversations/$exportId/export?format=json" $member
$memberCSV = Invoke-CertRequest GET "/v1/chat/conversations/$exportId/export?format=csv" $member
$outsiderExport = Invoke-CertRequest GET "/v1/chat/conversations/$exportId/export?format=json" $outsider
$memberDeletedExport = Invoke-CertRequest GET "/v1/chat/conversations/$exportId/export?format=json&includeDeleted=true" $member
$adminDeletedExport = Invoke-CertRequest GET "/v1/chat/conversations/$exportId/export?format=json&includeDeleted=true" $admin

$conversationHold = Invoke-CertRequest POST '/v1/compliance/legal-holds' $admin @{ name = 'Conversation evidence hold'; reason = 'Certification case C-1'; conversationId = $heldId }
$memberHold = Invoke-CertRequest POST '/v1/compliance/legal-holds' $member @{ name = 'Unauthorized hold'; reason = 'Must be denied' }
$badHold = Invoke-CertRequest POST '/v1/compliance/legal-holds' $admin @{ name = 'Bad scope'; reason = 'Unknown conversation'; conversationId = "missing-$stamp" }
$policyUpdate = Invoke-CertRequest PUT '/v1/compliance/retention' $admin @{ retentionDays = 1; enabled = $true }

$psql = 'C:\Program Files\PostgreSQL\17\bin\psql.exe'
$env:PGPASSWORD = Read-EnvValue 'STATCHAT_DB_PASSWORD'
$dbName = Read-EnvValue 'STATCHAT_DB_NAME'; if (-not $dbName) { $dbName = 'statchat' }
$dbUser = Read-EnvValue 'STATCHAT_DB_USER'; if (-not $dbUser) { $dbUser = 'Statchat' }
$backdate = "UPDATE messages SET created_at = NOW() - INTERVAL '2 days' WHERE id IN ('$heldMessageId','$purgeMessageId');"
$dbOutput = & $psql -h localhost -p 5432 -U $dbUser -d $dbName -v ON_ERROR_STOP=1 -q -c $backdate 2>&1
if ($LASTEXITCODE -ne 0) { throw "Failed to age certification messages: $dbOutput" }

$firstEnforcement = Invoke-CertRequest POST '/v1/compliance/retention/enforce' $admin
$heldAfterFirst = Response-Array (Invoke-CertRequest GET "/v1/chat/conversations/$heldId/messages" $member)
$purgedAfterFirst = Response-Array (Invoke-CertRequest GET "/v1/chat/conversations/$purgeId/messages" $member)

$conversationHoldId = ($conversationHold.Body | ConvertFrom-Json).id
$releaseConversation = Invoke-CertRequest POST "/v1/compliance/legal-holds/$conversationHoldId/release" $admin
$tenantHold = Invoke-CertRequest POST '/v1/compliance/legal-holds' $admin @{ name = 'Tenant evidence hold'; reason = 'Certification case T-1' }
$secondEnforcement = Invoke-CertRequest POST '/v1/compliance/retention/enforce' $admin
$heldUnderTenantHold = Response-Array (Invoke-CertRequest GET "/v1/chat/conversations/$heldId/messages" $member)

$tenantHoldId = ($tenantHold.Body | ConvertFrom-Json).id
$releaseTenant = Invoke-CertRequest POST "/v1/compliance/legal-holds/$tenantHoldId/release" $admin
$thirdEnforcement = Invoke-CertRequest POST '/v1/compliance/retention/enforce' $admin
$heldAfterRelease = Response-Array (Invoke-CertRequest GET "/v1/chat/conversations/$heldId/messages" $member)
$holds = Response-Array (Invoke-CertRequest GET '/v1/compliance/legal-holds' $admin)
$audit = Response-Array (Invoke-CertRequest GET '/v1/compliance/audit?limit=100' $admin)

$default = $defaultPolicy.Body | ConvertFrom-Json
$firstResult = $firstEnforcement.Body | ConvertFrom-Json
$thirdResult = $thirdEnforcement.Body | ConvertFrom-Json
$auditActions = @($audit | ForEach-Object { $_.action })
$result = [ordered]@{
    safeDefault = $defaultPolicy.Status -eq 200 -and -not $default.enabled -and $default.retentionDays -eq 365
    adminBoundary = $memberPolicy.Status -eq 403 -and $memberHold.Status -eq 403
    validation = $invalidPolicy.Status -eq 400 -and $badHold.Status -eq 400
    conversationSetup = $heldConversation.Status -eq 200 -and $purgeConversation.Status -eq 200 -and $exportConversation.Status -eq 200
    authenticatedExports = $memberJSON.Status -eq 200 -and $memberCSV.Status -eq 200 -and $memberCSV.Body -match "'=formula-safe"
    exportIsolation = $outsiderExport.Status -eq 403 -and $memberDeletedExport.Status -eq 403 -and $adminDeletedExport.Status -eq 200
    policyEnabled = $policyUpdate.Status -eq 200 -and ($policyUpdate.Body | ConvertFrom-Json).enabled
    conversationHoldProtected = $firstEnforcement.Status -eq 200 -and $firstResult.deletedMessages -eq 1 -and $heldAfterFirst.id -contains $heldMessageId -and $purgedAfterFirst.Count -eq 0
    tenantHoldProtected = $releaseConversation.Status -eq 200 -and $tenantHold.Status -eq 200 -and $secondEnforcement.Status -eq 200 -and $heldUnderTenantHold.id -contains $heldMessageId
    releasedHoldPurged = $releaseTenant.Status -eq 200 -and $thirdEnforcement.Status -eq 200 -and $thirdResult.deletedMessages -eq 1 -and $heldAfterRelease.Count -eq 0
    holdLifecycle = $holds.Count -eq 2 -and @($holds | Where-Object status -eq 'released').Count -eq 2
    immutableAudit = @('retention.policy.updated','legal_hold.created','legal_hold.released','retention.enforced','conversation.exported') | ForEach-Object { $auditActions -contains $_ } | Where-Object { -not $_ } | Measure-Object | Select-Object -ExpandProperty Count
}
$result.immutableAudit = $result.immutableAudit -eq 0

$result | ConvertTo-Json -Compress
if ($result.Values -contains $false) { exit 1 }
