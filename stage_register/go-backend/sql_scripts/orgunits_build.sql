-- Step 1: Create a temporary staging table from orgunits_uploads
CREATE TEMP TABLE staging_orgunits AS
SELECT 
    facility, facility_uid, 
    subcounty, subcounty_uid, 
    municipality, municipality_uid, 
    district, district_uid, 
    region, region_uid
FROM public.orgunits_uploads;

-- Step 2: Verify/Check admin_level structure
-- Level 1: MOH (Ministry of Health)
-- Level 2: Region
-- Level 3: District
-- Level 4: Municipality/DLG
-- Level 5: Subcounty
-- Level 6: Facility

-- Step 3: Insert MOH (Level 1) - Top level root node, no parent
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT 
    'MOH' AS name,
    'akV6429SUqu' AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 1 LIMIT 1) AS level_id,
    NULL AS parent_id,
    '/akV6429SUqu/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
WHERE NOT EXISTS (
    SELECT 1 FROM admin_units WHERE mfl_uid = 'akV6429SUqu'
);

-- Step 4: Insert Regions (Level 2) - Parent is MOH
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT DISTINCT
    s.region AS name,
    s.region_uid AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 2 LIMIT 1) AS level_id,
    (SELECT id FROM admin_units WHERE mfl_uid = 'akV6429SUqu') AS parent_id,
    '/akV6429SUqu/' || s.region_uid || '/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
FROM staging_orgunits s
WHERE s.region IS NOT NULL 
  AND s.region_uid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM admin_units WHERE mfl_uid = s.region_uid
  )
ORDER BY s.region;

-- Step 5: Insert Districts (Level 3) - Parent is Region
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT DISTINCT
    s.district AS name,
    s.district_uid AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 3 LIMIT 1) AS level_id,
    r.id AS parent_id,
    r.path || s.district_uid || '/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
FROM staging_orgunits s
INNER JOIN admin_units r ON r.mfl_uid = s.region_uid
WHERE s.district IS NOT NULL 
  AND s.district_uid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM admin_units WHERE mfl_uid = s.district_uid
  )
ORDER BY s.district;

-- Step 6: Insert Municipalities/DLGs (Level 4) - Parent is District
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT DISTINCT
    s.municipality AS name,
    s.municipality_uid AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 4 LIMIT 1) AS level_id,
    d.id AS parent_id,
    d.path || s.municipality_uid || '/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
FROM staging_orgunits s
INNER JOIN admin_units d ON d.mfl_uid = s.district_uid
WHERE s.municipality IS NOT NULL 
  AND s.municipality_uid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM admin_units WHERE mfl_uid = s.municipality_uid
  )
ORDER BY s.municipality;

-- Step 7: Insert Subcounties (Level 5) - Parent is Municipality
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT DISTINCT
    s.subcounty AS name,
    s.subcounty_uid AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 5 LIMIT 1) AS level_id,
    m.id AS parent_id,
    m.path || s.subcounty_uid || '/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
FROM staging_orgunits s
INNER JOIN admin_units m ON m.mfl_uid = s.municipality_uid
WHERE s.subcounty IS NOT NULL 
  AND s.subcounty_uid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM admin_units WHERE mfl_uid = s.subcounty_uid
  )
ORDER BY s.subcounty;

-- Step 8: Insert Facilities (Level 6) - Parent is Subcounty
INSERT INTO admin_units (name, mfl_uid, level_id, parent_id, path, "createdAt", "updatedAt")
SELECT DISTINCT
    s.facility AS name,
    s.facility_uid AS mfl_uid,
    (SELECT id FROM admin_level WHERE level_number = 6 LIMIT 1) AS level_id,
    sc.id AS parent_id,
    sc.path || s.facility_uid || '/' AS path,
    NOW() AS "createdAt",
    NOW() AS "updatedAt"
FROM staging_orgunits s
INNER JOIN admin_units sc ON sc.mfl_uid = s.subcounty_uid
WHERE s.facility IS NOT NULL 
  AND s.facility_uid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM admin_units WHERE mfl_uid = s.facility_uid
  )
ORDER BY s.facility;

-- Step 9: Verify the data
SELECT 
    al.level_number,
    al.name AS level_name,
    COUNT(*) AS count
FROM admin_units au
JOIN admin_level al ON au.level_id = al.id
GROUP BY al.level_number, al.name
ORDER BY al.level_number;

-- Step 10: Sample hierarchical view to verify MOH is at top
SELECT 
    au.id,
    au.name,
    au.mfl_uid,
    al.level_number,
    al.name AS level_name,
    au.path,
    parent.name AS parent_name
FROM admin_units au
JOIN admin_level al ON au.level_id = al.id
LEFT JOIN admin_units parent ON au.parent_id = parent.id
WHERE au.mfl_uid = 'akV6429SUqu' 
   OR au.path LIKE '/akV6429SUqu/%'
ORDER BY au.path
LIMIT 30;

-- Step 11: Check total counts per level
SELECT 
    CASE al.level_number
        WHEN 1 THEN 'MOH'
        WHEN 2 THEN 'Region'
        WHEN 3 THEN 'District'
        WHEN 4 THEN 'Municipality/DLG'
        WHEN 5 THEN 'Subcounty'
        WHEN 6 THEN 'Facility'
    END AS level_description,
    al.level_number,
    COUNT(*) AS total_count
FROM admin_units au
JOIN admin_level al ON au.level_id = al.id
GROUP BY al.level_number
ORDER BY al.level_number;