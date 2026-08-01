<#+
Rewrites the checked-in CSV seed data as synthetic StatGate Field Operations data.

The importer schema is intentionally unchanged. In particular, region, district,
municipality/DLG, subcounty, and their UID columns are copied verbatim so existing
geographic filtering and hierarchy joins continue to work.
#>

[CmdletBinding()]
param(
    [string]$Root
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Split-Path -Parent $PSScriptRoot
}

function Get-Slug {
    param([string]$Value)
    $slug = ($Value -replace '[^A-Za-z0-9]', '').ToUpperInvariant()
    if ([string]::IsNullOrWhiteSpace($slug)) { return 'AREA' }
    return $slug.Substring(0, [Math]::Min(5, $slug.Length))
}

function Get-StationTier {
    param([string]$LegacyLevel)
    switch -Regex ($LegacyLevel) {
        'Hospital|Referral' { return 'Regional Operations Hub' }
        'HC IV|IV$'          { return 'District Coordination Hub' }
        'HC III|III$'        { return 'Subcounty Field Station' }
        default              { return 'Community Enumeration Point' }
    }
}

$stationsPath = Join-Path $Root 'mflupload.csv'
$hierarchyPath = Join-Path $Root 'orgunits_uploads.csv'

$stationRows = Import-Csv -LiteralPath $stationsPath
$areaCounters = @{}
$stationNamesByUid = @{}

foreach ($row in $stationRows) {
    $areaKey = "$($row.region)|$($row.district)|$($row.subcounty)"
    if (-not $areaCounters.ContainsKey($areaKey)) { $areaCounters[$areaKey] = 0 }
    $areaCounters[$areaKey]++

    $stationNumber = $areaCounters[$areaKey].ToString('000')
    $districtSlug = Get-Slug $row.district
    $stationName = "$($row.subcounty) Field Survey Station $stationNumber"

    # Keep importer headers intact while providing StatGate-oriented seed values.
    # The check makes repeat runs safe after the first conversion.
    if ($row.name -notmatch 'Field Survey Station') {
        $row.name = $stationName
        $row.shortname = "SG-$districtSlug-$stationNumber"
        $row.hflevel = Get-StationTier $row.hflevel
    }
    # The existing staging table stores this legacy importer field as BIGINT.
    # Keep it numeric and unique; the human-facing StatGate station code lives
    # in shortname.
    if ($row.nhfrid -notmatch '^\d+$') {
        $row.nhfrid = ([int64]9000000000000 + [int64]$row.id).ToString()
    }

    # These are still stored in the legacy ownership/authority fields. Their
    # values stay as existing codes/names so the current SQL lookups and filters
    # remain compatible; the React UI supplies the StatGate-facing labels.
    $stationNamesByUid[$row.uid] = $stationName
}

$stationRows | Export-Csv -LiteralPath $stationsPath -NoTypeInformation -Encoding utf8

$hierarchyRows = Import-Csv -LiteralPath $hierarchyPath
foreach ($row in $hierarchyRows) {
    if ($stationNamesByUid.ContainsKey($row.facility_uid)) {
        $row.facility = $stationNamesByUid[$row.facility_uid]
    } elseif ($row.facility -notmatch 'Field Survey Station' -or $row.facility -match 'health centre|hospital|clinic|medical centre') {
        # Some legacy hierarchy rows carry a health-facility name in the
        # subcounty column. Do not alter that geographic source field; use a
        # neutral, UID-based station name for the displayed facility column.
        $row.facility = "Field Survey Station $($row.facility_uid.Substring(0, 6).ToUpperInvariant())"
    }
}
$hierarchyRows | Export-Csv -LiteralPath $hierarchyPath -NoTypeInformation -Encoding utf8

foreach ($userFile in @('users_upload.csv', 'users_upload_prod.csv')) {
    $userPath = Join-Path $Root $userFile
    $userRows = Import-Csv -LiteralPath $userPath
    $index = 0
    foreach ($row in $userRows) {
        $index++
        $suffix = $index.ToString('000')
        if ($row.PSObject.Properties.Name -contains 'firstname') {
            $row.firstname = 'Field'
            $row.lastname = "Coordinator $suffix"
        }
        if ($row.PSObject.Properties.Name -contains 'first_name') {
            $row.first_name = 'Field'
            $row.last_name = "Coordinator $suffix"
        }
        $row.username = "field.coordinator.$suffix"
        $row.email = "field.coordinator.$suffix@statgate.example"
    }
    $userRows | Export-Csv -LiteralPath $userPath -NoTypeInformation -Encoding utf8
}

Write-Output "Cooked $($stationRows.Count) field survey stations and $($hierarchyRows.Count) hierarchy rows."
