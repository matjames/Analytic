# =====================================================================
# StatGate end-to-end DATA LIFECYCLE acceptance gate (plan item 2)
#
# Proves the full lifecycle against a live stack:
#   collect/ingest -> real pipeline execution -> durable storage -> analysis
#
# It is deliberately an ANTI-SIMULATION gate. It fails if:
#   * a pipeline run reports SUCCESS without moving real rows,
#   * stage counters do not match the rows actually stored,
#   * a broken pipeline does NOT fail (guaranteed-success regression),
#   * a failed VALIDATE still loads rows into the target,
#   * analytics uses request payload rows instead of persisted dataset rows.
#
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File tests/2-lifecycle/lifecycle.ps1
#   powershell ... -StatDataBase http://localhost:8307 -CoreBase http://localhost:8302
# =====================================================================
param(
    [string]$StatDataBase = "http://localhost:8107",
    [string]$CoreBase = "http://localhost:8082",
    [string]$TenantId = "tenant-alpha",
    [string]$OutFile = ""
)

$ErrorActionPreference = 'Continue'
$repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not $OutFile) { $OutFile = Join-Path $PSScriptRoot 'TEST_RESULT' }

$script:Pass = 0
$script:Fail = 0
$script:Lines = New-Object System.Collections.ArrayList

function Assert-That {
    param([string]$Name, [bool]$Condition, [string]$Detail = '')
    if ($Condition) {
        $script:Pass++
        [void]$script:Lines.Add("PASS  $Name")
        Write-Output "PASS  $Name"
    } else {
        $script:Fail++
        [void]$script:Lines.Add("FAIL  $Name  :: $Detail")
        Write-Output "FAIL  $Name  :: $Detail"
    }
}

function Assert-Eq {
    param([string]$Name, $Expected, $Actual)
    Assert-That $Name ($Expected -eq $Actual) "expected='$Expected' actual='$Actual'"
}

# ---- read required secrets from env, else the gitignored .env ----
function Get-EnvValue {
    param([string]$Key)
    $v = [Environment]::GetEnvironmentVariable($Key)
    if ($v) { return $v }
    $envFile = Join-Path $repo '.env'
    if (Test-Path $envFile) {
        $m = Select-String -Path $envFile -Pattern ("^" + [regex]::Escape($Key) + "=(.+)$") | Select-Object -First 1
        if ($m) { return $m.Matches[0].Groups[1].Value.Trim() }
    }
    return ""
}

$internalKey = Get-EnvValue 'STATGATE_INTERNAL_API_KEY'
$jwtSecret = Get-EnvValue 'STATGATE_REGISTRY_JWT_SECRET'
if (-not $internalKey) { Write-Output 'FATAL: STATGATE_INTERNAL_API_KEY not set and not found in .env.'; exit 2 }
if (-not $jwtSecret) { Write-Output 'FATAL: STATGATE_REGISTRY_JWT_SECRET not set and not found in .env.'; exit 2 }

function B64Url([byte[]]$b) {
    return [Convert]::ToBase64String($b).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

# Registry-contract HS256 token (statgate-lib validator: userId/role/tenant_id/iss/aud)
$now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$headerJson = '{"alg":"HS256","typ":"JWT"}'
$payloadJson = '{"userId":"lifecycle-runner","role":"analyst","tenant_id":"' + $TenantId +
    '","email":"lifecycle@example.com","iss":"statgate-registry","aud":"statgate","iat":' +
    $now + ',"exp":' + ($now + 900) + '}'
$h = B64Url ([Text.Encoding]::UTF8.GetBytes($headerJson))
$p = B64Url ([Text.Encoding]::UTF8.GetBytes($payloadJson))
$sig = B64Url ([System.Security.Cryptography.HMACSHA256]::new(
    [Text.Encoding]::UTF8.GetBytes($jwtSecret)).ComputeHash([Text.Encoding]::UTF8.GetBytes("$h.$p")))
$token = "$h.$p.$sig"

# ---- HTTP helpers ----
function ConvertFrom-JsonSafe {
    param($Text)
    if (-not $Text) { return $null }
    try { return $Text | ConvertFrom-Json } catch { return $null }
}

function Invoke-Api {
    param([string]$Method, [string]$Url, [hashtable]$Headers, $Body = $null)
    $req = @{ Uri = $Url; Method = $Method; UseBasicParsing = $true; TimeoutSec = 30; Headers = $Headers }
    if ($null -ne $Body) {
        $req.ContentType = 'application/json'
        $req.Body = ($Body | ConvertTo-Json -Depth 12 -Compress)
    }
    try {
        $r = Invoke-WebRequest @req
        return @{ Status = [int]$r.StatusCode; Json = (ConvertFrom-JsonSafe $r.Content); Raw = $r.Content }
    } catch {
        $status = 0; $content = ''
        if ($_.Exception.Response) {
            try { $status = [int]$_.Exception.Response.StatusCode.value__ } catch { }
            try {
                $sr = New-Object IO.StreamReader($_.Exception.Response.GetResponseStream())
                $content = $sr.ReadToEnd()
            } catch { }
        }
        return @{ Status = $status; Json = (ConvertFrom-JsonSafe $content); Raw = $content }
    }
}

$dataHeaders = @{
    'Authorization' = "Bearer $token"
    'X-Tenant-ID'   = $TenantId
    'X-User-ID'     = 'lifecycle-runner'
}
$coreHeaders = @{
    'X-StatGate-Internal-Key' = $internalKey
    'X-Tenant-ID'             = $TenantId
    'X-User-Role'             = 'analyst'
    'X-User-Clearance'        = '4'
}

# ---- preflight: the stack must be up and protected ----
Write-Output "=== StatGate lifecycle gate | statdata=$StatDataBase core=$CoreBase tenant=$TenantId ==="

$sdHealth = Invoke-Api 'GET' "$StatDataBase/health" @{}
Assert-Eq 'preflight: statdata /health' 200 $sdHealth.Status
$coreHealth = Invoke-Api 'GET' "$CoreBase/health" @{}
Assert-Eq 'preflight: statgate-core /health' 200 $coreHealth.Status
if ($sdHealth.Status -ne 200 -or $coreHealth.Status -ne 200) {
    Write-Output 'FATAL: stack not reachable; start the Compose stack (or pass -StatDataBase/-CoreBase).'
    exit 2
}

$noAuth = Invoke-Api 'GET' "$StatDataBase/api/v1/data/catalog/datasets" @{}
Assert-Eq 'preflight: catalog requires auth (no token -> 401)' 401 $noAuth.Status
$noKey = Invoke-Api 'POST' "$CoreBase/api/v1/statistics/tabulate" @{ 'Content-Type' = 'application/json' } @{ row_variable = 'x' }
Assert-Eq 'preflight: core tabulate requires internal key (-> 401)' 401 $noKey.Status

# =====================================================================
# STAGE 1 - COLLECT / INGEST: datasets must hold real rows
# =====================================================================
$stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()

$srcCreate = Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets" $dataHeaders @{
    id             = "ds-lifecycle-src-$stamp"
    name           = "Lifecycle Source $stamp"
    domain         = "demographics"
    classification = "INTERNAL"
    owner_team     = "Surveys"
}
Assert-Eq 'stage1: create source dataset' 201 $srcCreate.Status
$srcId = $srcCreate.Json.id
if (-not $srcId) { $srcId = "ds-lifecycle-src-$stamp" }
Assert-That 'stage1: source dataset id present' ([bool]$srcId) "raw=$($srcCreate.Raw)"
$srcRecords = "$StatDataBase/api/v1/data/catalog/datasets/$srcId/records"

# Anti-simulation control: an empty batch must be refused, not silently "succeed".
$emptyPush = Invoke-Api 'POST' $srcRecords $dataHeaders @{ records = @() }
Assert-Eq 'stage1: empty record batch refused (400)' 400 $emptyPush.Status

$sourceRows = @(
    @{ district = 'North'; population = 52000; note = 'x' },
    @{ district = 'South'; population = 31000; note = 'y' },
    @{ district = 'East';  population = 8400;  note = 'z' },
    @{ district = 'West';  population = 47000; note = 'w' }
)
$push = Invoke-Api 'POST' $srcRecords $dataHeaders @{ records = $sourceRows }
Assert-Eq 'stage1: store 4 source rows' 201 $push.Status
Assert-Eq 'stage1: records_stored reported' 4 $push.Json.records_stored
Assert-Eq 'stage1: total_rows reflects storage' 4 $push.Json.total_rows
Assert-That 'stage1: bytes_written > 0' (([int64]$push.Json.bytes_written) -gt 0) "bytes_written=$($push.Json.bytes_written)"

$readback = Invoke-Api 'GET' ($srcRecords + '?limit=100') $dataHeaders
Assert-Eq 'stage1: stored rows are readable' 4 $readback.Json.count

$tgtCreate = Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets" $dataHeaders @{
    id             = "ds-lifecycle-tgt-$stamp"
    name           = "Lifecycle Target $stamp"
    domain         = "demographics"
    classification = "INTERNAL"
    owner_team     = "Data Platform"
}
Assert-Eq 'stage1: create target dataset' 201 $tgtCreate.Status
$tgtId = $tgtCreate.Json.id
if (-not $tgtId) { $tgtId = "ds-lifecycle-tgt-$stamp" }
$tgtRecords = "$StatDataBase/api/v1/data/catalog/datasets/$tgtId/records"

# =====================================================================
# STAGE 2 - DEFINE: quality rule and a real ETL pipeline DAG
# =====================================================================
$rule = Invoke-Api 'POST' "$StatDataBase/api/v1/data/quality/rules" $dataHeaders @{
    id           = "qr-lifecycle-$stamp"
    dataset_id   = $srcId
    rule_name    = 'District Not Null'
    rule_type    = 'NOT_NULL'
    target_field = 'district'
    severity     = 'ERROR'
}
Assert-Eq 'stage2: create NOT_NULL quality rule' 201 $rule.Status

$pipelineBody = @{
    id                = "pipe-lifecycle-$stamp"
    name              = "Lifecycle ETL $stamp"
    pipeline_type     = 'ETL'
    source_dataset_id = $srcId
    target_dataset_id = $tgtId
    stages            = @(
        @{ id = 'stg-ext'; name = 'Extract'; stage_type = 'EXTRACT' },
        # Validate the raw extracted rows BEFORE transformation: the NOT_NULL
        # rule targets 'district', which the rename below removes. This ordering
        # is deliberate and is itself part of what the gate proves.
        @{ id = 'stg-val'; name = 'Validate'; stage_type = 'VALIDATE' },
        @{ id = 'stg-xf'; name = 'Filter and Rename'; stage_type = 'TRANSFORM'; config = @{
                operations = @(
                    @{ type = 'filter'; field = 'population'; op = 'gte'; value = 10000 },
                    @{ type = 'rename'; from = 'district'; to = 'region' },
                    @{ type = 'remove_fields'; fields = @('note') }
                )
            }
        },
        @{ id = 'stg-load'; name = 'Load'; stage_type = 'LOAD' }
    )
}
$pipeCreate = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines" $dataHeaders $pipelineBody
Assert-Eq 'stage2: create pipeline' 201 $pipeCreate.Status
$pipeId = $pipeCreate.Json.id
if (-not $pipeId) { $pipeId = "pipe-lifecycle-$stamp" }

# =====================================================================
# STAGE 3 - PROCESS: execute the pipeline and demand real evidence
# =====================================================================
$run = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines/$pipeId/run" $dataHeaders @{}
Assert-Eq 'stage3: pipeline run accepted' 200 $run.Status
Assert-Eq 'stage3: run status SUCCESS' 'SUCCESS' $run.Json.status
Assert-Eq 'stage3: records_read equals stored source rows' 4 $run.Json.records_read
Assert-Eq 'stage3: records_written equals loaded rows' 3 $run.Json.records_written
Assert-Eq 'stage3: records_rejected equals filtered rows' 1 $run.Json.records_rejected
Assert-Eq 'stage3: four stages executed' 4 @($run.Json.stage_runs).Count

$stages = @($run.Json.stage_runs)
$extract = $stages | Where-Object { $_.stage_id -eq 'stg-ext' }
$transform = $stages | Where-Object { $_.stage_id -eq 'stg-xf' }
$validate = $stages | Where-Object { $_.stage_id -eq 'stg-val' }
$load = $stages | Where-Object { $_.stage_id -eq 'stg-load' }

Assert-Eq 'stage3: extract stage SUCCESS' 'SUCCESS' $extract.status
Assert-Eq 'stage3: extract summary records_extracted' 4 $extract.output_summary.records_extracted
Assert-Eq 'stage3: transform summary records_in' 4 $transform.output_summary.records_in
Assert-Eq 'stage3: transform summary records_out' 3 $transform.output_summary.records_out
Assert-Eq 'stage3: validate summary quality_status' 'PASSED' $validate.output_summary.quality_status
Assert-That 'stage3: validate evaluated a real rule' (([int]$validate.output_summary.rules_total) -ge 1) "rules_total=$($validate.output_summary.rules_total)"
Assert-Eq 'stage3: load summary records_loaded' 3 $load.output_summary.records_loaded
Assert-That 'stage3: load summary bytes_written > 0' (([int64]$load.output_summary.bytes_written) -gt 0) "bytes_written=$($load.output_summary.bytes_written)"
Assert-Eq 'stage3: load summary total_rows in storage' 3 $load.output_summary.total_rows

# =====================================================================
# STAGE 4 - STORE: the target must physically hold transformed rows
# =====================================================================
$tgtRead = Invoke-Api 'GET' ($tgtRecords + '?limit=100') $dataHeaders
Assert-Eq 'stage4: target holds transformed rows' 3 $tgtRead.Json.count
$row0 = @($tgtRead.Json.records)[0]
Assert-That 'stage4: transformed row carries region' ($null -ne $row0.region) "row=$($tgtRead.Raw)"
Assert-That 'stage4: district renamed away' ($null -eq $row0.district) "row=$($tgtRead.Raw)"
Assert-That 'stage4: note removed' ($null -eq $row0.note) "row=$($tgtRead.Raw)"

$tgtMeta = Invoke-Api 'GET' "$StatDataBase/api/v1/data/catalog/datasets/$tgtId" $dataHeaders
Assert-Eq 'stage4: catalog row_count reflects storage' 3 $tgtMeta.Json.row_count
Assert-That 'stage4: catalog size_bytes > 0' (([int64]$tgtMeta.Json.size_bytes) -gt 0) "size_bytes=$($tgtMeta.Json.size_bytes)"

# A second run must APPEND: counters are derived from storage, not constants.
$run2 = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines/$pipeId/run" $dataHeaders @{}
Assert-Eq 'stage4: second run SUCCESS' 'SUCCESS' $run2.Json.status
$load2 = @($run2.Json.stage_runs) | Where-Object { $_.stage_id -eq 'stg-load' }
Assert-Eq 'stage4: second run reports 6 rows in storage' 6 $load2.output_summary.total_rows

$cleared = Invoke-Api 'DELETE' $tgtRecords $dataHeaders
Assert-Eq 'stage4: record reset deletes every row' 6 $cleared.Json.deleted_rows
$afterClear = Invoke-Api 'GET' ($tgtRecords + '?limit=100') $dataHeaders
Assert-Eq 'stage4: target empty after reset' 0 $afterClear.Json.count

# =====================================================================
# STAGE 5 - FAIL HONESTLY: broken pipelines must NOT report SUCCESS
# =====================================================================
$ghostId = "pipe-lifecycle-ghost-$stamp"
$ghost = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines" $dataHeaders @{
    id                = $ghostId
    name              = "Ghost $stamp"
    pipeline_type     = 'ETL'
    source_dataset_id = "ds-does-not-exist-$stamp"
    target_dataset_id = $tgtId
    stages            = @(
        @{ id = 'stg-ext'; name = 'Extract'; stage_type = 'EXTRACT' },
        @{ id = 'stg-load'; name = 'Load'; stage_type = 'LOAD' }
    )
}
Assert-Eq 'stage5: ghost pipeline created' 201 $ghost.Status
$ghostRun = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines/$ghostId/run" $dataHeaders @{}
Assert-Eq 'stage5: missing source reports FAILED (no guaranteed SUCCESS)' 'FAILED' $ghostRun.Json.status
Assert-That 'stage5: failure carries a real message' (-not [string]::IsNullOrWhiteSpace([string]$ghostRun.Json.error_message)) "raw=$($ghostRun.Raw)"
Assert-Eq 'stage5: failed run wrote nothing' 0 ([int64]$ghostRun.Json.records_written)
$ghostTarget = Invoke-Api 'GET' ($tgtRecords + '?limit=100') $dataHeaders
Assert-Eq 'stage5: target untouched by the failed run' 0 $ghostTarget.Json.count

# A failed QUALITY gate must stop the run before LOAD touches the target.
$dirtyId = "ds-lifecycle-dirty-$stamp"
$cleanId = "ds-lifecycle-clean-$stamp"
[void](Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets" $dataHeaders @{
    id = $dirtyId; name = "Dirty $stamp"; domain = 'demographics'; classification = 'INTERNAL'; owner_team = 'Surveys' })
$cleanCreate = Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets" $dataHeaders @{
    id = $cleanId; name = "Clean $stamp"; domain = 'demographics'; classification = 'INTERNAL'; owner_team = 'Data Platform' }
Assert-Eq 'stage5: clean target created' 201 $cleanCreate.Status
[void](Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets/$dirtyId/records" $dataHeaders @{
    records = @(
        @{ district = 'North'; population = 100 },
        @{ district = $null;   population = 200 }
    ) })
[void](Invoke-Api 'POST' "$StatDataBase/api/v1/data/quality/rules" $dataHeaders @{
    id = "qr-lifecycle-dirty-$stamp"; dataset_id = $dirtyId; rule_name = 'District Not Null'
    rule_type = 'NOT_NULL'; target_field = 'district'; severity = 'ERROR' })

$qviolId = "pipe-lifecycle-qviol-$stamp"
[void](Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines" $dataHeaders @{
    id = $qviolId; name = "Quality Gate $stamp"; pipeline_type = 'ETL'
    source_dataset_id = $dirtyId; target_dataset_id = $cleanId
    stages = @(
        @{ id = 'stg-ext'; name = 'Extract'; stage_type = 'EXTRACT' },
        @{ id = 'stg-val'; name = 'Validate'; stage_type = 'VALIDATE' },
        @{ id = 'stg-load'; name = 'Load'; stage_type = 'LOAD' }
    )
})
$qviolRun = Invoke-Api 'POST' "$StatDataBase/api/v1/data/pipelines/$qviolId/run" $dataHeaders @{}
Assert-Eq 'stage5: quality violation fails the run' 'FAILED' $qviolRun.Json.status
$qviolStages = @($qviolRun.Json.stage_runs)
$qviolValidate = $qviolStages | Where-Object { $_.stage_id -eq 'stg-val' }
Assert-Eq 'stage5: validate stage is the failing stage' 'FAILED' $qviolValidate.status
Assert-That 'stage5: failing validate names the violated rule' (([int]$qviolValidate.output_summary.rules_failed) -ge 1) "rules_failed=$($qviolValidate.output_summary.rules_failed)"
Assert-Eq 'stage5: load never ran after failure' 2 $qviolStages.Count
$cleanAfter = Invoke-Api 'GET' "$StatDataBase/api/v1/data/catalog/datasets/$cleanId/records?limit=100" $dataHeaders
Assert-Eq 'stage5: clean target stayed empty' 0 $cleanAfter.Json.count

# =====================================================================
# STAGE 6 - ANALYZE: tabulation must read persisted rows, not payload
# =====================================================================
$tabId = "ds-lifecycle-tab-$stamp"
$tabCreate = Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets" $dataHeaders @{
    id = $tabId; name = "Lifecycle Tab $stamp"; domain = 'demographics'
    classification = 'INTERNAL'; owner_team = 'Analytics'
}
Assert-Eq 'stage6: create tabulation dataset' 201 $tabCreate.Status

$tabRows = @(
    @{ region = 'North'; sex = 'F'; count = 30 },
    @{ region = 'North'; sex = 'M'; count = 20 },
    @{ region = 'South'; sex = 'F'; count = 10 },
    @{ region = 'South'; sex = 'M'; count = 40 }
)
$tabPush = Invoke-Api 'POST' "$StatDataBase/api/v1/data/catalog/datasets/$tabId/records" $dataHeaders @{ records = $tabRows }
Assert-Eq 'stage6: tabulation rows stored' 4 $tabPush.Json.total_rows

$tab = Invoke-Api 'POST' "$CoreBase/api/v1/statistics/tabulate" $coreHeaders @{
    title = 'Lifecycle tabulation'; dataset_id = $tabId
    row_variable = 'region'; col_variable = 'sex'; aggregation = 'COUNT'
}
Assert-Eq 'stage6: dataset-bound tabulation accepted' 200 $tab.Status
Assert-Eq 'stage6: tabulation used persisted rows' 4 $tab.Json.total_record_count
Assert-Eq 'stage6: North row total from storage' 2 $tab.Json.row_totals.North

# Payload rows must NOT override storage when dataset_id is set.
$bogus = @()
1..5 | ForEach-Object { $bogus += @{ region = 'Bogus'; sex = 'X'; count = 1 } }
$tabBogus = Invoke-Api 'POST' "$CoreBase/api/v1/statistics/tabulate" $coreHeaders @{
    title = 'Anti-simulation'; dataset_id = $tabId; row_variable = 'region'
    col_variable = 'sex'; aggregation = 'COUNT'; data_records = $bogus
}
Assert-Eq 'stage6: payload rows ignored when dataset_id is set' 4 $tabBogus.Json.total_record_count
Assert-That 'stage6: no payload category leaked into results' ($null -eq $tabBogus.Json.row_totals.Bogus) "row_totals=$($tabBogus.Json.row_totals | ConvertTo-Json -Compress)"

# ---- cleanup (datasets only; pipeline definitions have no DELETE route) ----
foreach ($id in @($srcId, $tgtId, $tabId, $dirtyId, $cleanId)) {
    if ($id) { [void](Invoke-Api 'DELETE' "$StatDataBase/api/v1/data/catalog/datasets/$id" $dataHeaders) }
}

# ---- evidence ----
Write-Output ''
Write-Output "=== lifecycle gate: $script:Pass passed, $script:Fail failed ==="
$evidence = New-Object System.Collections.ArrayList
[void]$evidence.Add('StatGate end-to-end data lifecycle acceptance gate')
[void]$evidence.Add('run_at_utc: ' + [DateTime]::UtcNow.ToString('u'))
[void]$evidence.Add("statdata=$StatDataBase core=$CoreBase tenant=$TenantId")
[void]$evidence.Add("source_dataset=$srcId target_dataset=$tgtId pipeline=$pipeId")
[void]$evidence.Add("passed=$script:Pass failed=$script:Fail")
[void]$evidence.Add('')
foreach ($line in $script:Lines) { [void]$evidence.Add($line) }
$evidence | Set-Content -Path $OutFile -Encoding UTF8
Write-Output "evidence: $OutFile"
if ($script:Fail -gt 0) { exit 1 }
exit 0