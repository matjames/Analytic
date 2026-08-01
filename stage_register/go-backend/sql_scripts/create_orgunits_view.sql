-- Create flattened orgunits view
-- This view denormalizes the entire organizational hierarchy for better query performance

DROP VIEW IF EXISTS public.orgunits;

WITH RECURSIVE hierarchy AS (
    -- Get the full hierarchy path for each admin unit
    SELECT 
        au.id,
        au.mfl_uid,
        au.name,
        au.code as identifier,
        au.parent_id,
        au.level_id,
        au.path,
        au."createdAt",
        au."updatedAt",
        al.level_number as level,
        al.mfl_uid as level_mfl_uid,
        al.name as level_name,
        -- Initialize hierarchy columns
        CASE WHEN al.level_number = 1 THEN au.id ELSE NULL END as national_id,
        CASE WHEN al.level_number = 1 THEN au.name ELSE NULL END as national_name,
        CASE WHEN al.level_number = 1 THEN au.mfl_uid ELSE NULL END as national_mfl_uid,
        CASE WHEN al.level_number = 2 THEN au.id ELSE NULL END as region_id,
        CASE WHEN al.level_number = 2 THEN au.name ELSE NULL END as region_name,
        CASE WHEN al.level_number = 2 THEN au.mfl_uid ELSE NULL END as region_mfl_uid,
        CASE WHEN al.level_number = 3 THEN au.id ELSE NULL END as district_city_id,
        CASE WHEN al.level_number = 3 THEN au.name ELSE NULL END as district_city_name,
        CASE WHEN al.level_number = 3 THEN au.mfl_uid ELSE NULL END as district_city_mfl_uid,
        CASE WHEN al.level_number = 4 THEN au.id ELSE NULL END as dlg_municipality_id,
        CASE WHEN al.level_number = 4 THEN au.name ELSE NULL END as dlg_municipality_name,
        CASE WHEN al.level_number = 4 THEN au.mfl_uid ELSE NULL END as dlg_municipality_mfl_uid,
        CASE WHEN al.level_number = 5 THEN au.id ELSE NULL END as subcounty_division_id,
        CASE WHEN al.level_number = 5 THEN au.name ELSE NULL END as subcounty_division_name,
        CASE WHEN al.level_number = 5 THEN au.mfl_uid ELSE NULL END as subcounty_division_mfl_uid,
        CASE WHEN al.level_number = 6 THEN au.id ELSE NULL END as facility_au_id,
        CASE WHEN al.level_number = 6 THEN au.name ELSE NULL END as facility_au_name,
        CASE WHEN al.level_number = 6 THEN au.mfl_uid ELSE NULL END as facility_au_mfl_uid,
        CASE WHEN al.level_number = 7 THEN au.id ELSE NULL END as parish_id,
        CASE WHEN al.level_number = 7 THEN au.name ELSE NULL END as parish_name,
        CASE WHEN al.level_number = 7 THEN au.mfl_uid ELSE NULL END as parish_mfl_uid,
        CASE WHEN al.level_number = 8 THEN au.id ELSE NULL END as village_id,
        CASE WHEN al.level_number = 8 THEN au.name ELSE NULL END as village_name,
        CASE WHEN al.level_number = 8 THEN au.mfl_uid ELSE NULL END as village_mfl_uid
    FROM admin_units au
    JOIN admin_level al ON au.level_id = al.id
    WHERE au.parent_id IS NULL  -- Start with root nodes
    UNION ALL
    SELECT 
        au.id,
        au.mfl_uid,
        au.name,
        au.code as identifier,
        au.parent_id,
        au.level_id,
        au.path,
        au."createdAt",
        au."updatedAt",
        al.level_number as level,
        al.mfl_uid as level_mfl_uid,
        al.name as level_name,
        -- Propagate hierarchy from parent
        COALESCE(h.national_id, CASE WHEN al.level_number = 1 THEN au.id ELSE NULL END),
        COALESCE(h.national_name, CASE WHEN al.level_number = 1 THEN au.name ELSE NULL END),
        COALESCE(h.national_mfl_uid, CASE WHEN al.level_number = 1 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.region_id, CASE WHEN al.level_number = 2 THEN au.id ELSE NULL END),
        COALESCE(h.region_name, CASE WHEN al.level_number = 2 THEN au.name ELSE NULL END),
        COALESCE(h.region_mfl_uid, CASE WHEN al.level_number = 2 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.district_city_id, CASE WHEN al.level_number = 3 THEN au.id ELSE NULL END),
        COALESCE(h.district_city_name, CASE WHEN al.level_number = 3 THEN au.name ELSE NULL END),
        COALESCE(h.district_city_mfl_uid, CASE WHEN al.level_number = 3 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.dlg_municipality_id, CASE WHEN al.level_number = 4 THEN au.id ELSE NULL END),
        COALESCE(h.dlg_municipality_name, CASE WHEN al.level_number = 4 THEN au.name ELSE NULL END),
        COALESCE(h.dlg_municipality_mfl_uid, CASE WHEN al.level_number = 4 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.subcounty_division_id, CASE WHEN al.level_number = 5 THEN au.id ELSE NULL END),
        COALESCE(h.subcounty_division_name, CASE WHEN al.level_number = 5 THEN au.name ELSE NULL END),
        COALESCE(h.subcounty_division_mfl_uid, CASE WHEN al.level_number = 5 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.facility_au_id, CASE WHEN al.level_number = 6 THEN au.id ELSE NULL END),
        COALESCE(h.facility_au_name, CASE WHEN al.level_number = 6 THEN au.name ELSE NULL END),
        COALESCE(h.facility_au_mfl_uid, CASE WHEN al.level_number = 6 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.parish_id, CASE WHEN al.level_number = 7 THEN au.id ELSE NULL END),
        COALESCE(h.parish_name, CASE WHEN al.level_number = 7 THEN au.name ELSE NULL END),
        COALESCE(h.parish_mfl_uid, CASE WHEN al.level_number = 7 THEN au.mfl_uid ELSE NULL END),
        COALESCE(h.village_id, CASE WHEN al.level_number = 8 THEN au.id ELSE NULL END),
        COALESCE(h.village_name, CASE WHEN al.level_number = 8 THEN au.name ELSE NULL END),
        COALESCE(h.village_mfl_uid, CASE WHEN al.level_number = 8 THEN au.mfl_uid ELSE NULL END)
    FROM admin_units au
    JOIN admin_level al ON au.level_id = al.id
    JOIN hierarchy h ON au.parent_id = h.id
)
SELECT 
    h.national_id,
    h.national_name,
    h.national_mfl_uid,
    h.region_id,
    h.region_name,
    h.region_mfl_uid,
    h.district_city_id,
    h.district_city_name,
    h.district_city_mfl_uid,
    h.dlg_municipality_id,
    h.dlg_municipality_name,
    h.dlg_municipality_mfl_uid,
    h.subcounty_division_id,
    h.subcounty_division_name,
    h.subcounty_division_mfl_uid,
    h.facility_au_id,
    h.facility_au_name,
    h.facility_au_mfl_uid,
    h.parish_id,
    h.parish_name,
    h.parish_mfl_uid,
    h.village_id,
    h.village_name,
    h.village_mfl_uid,
    f.identifier as identifier,
    h.mfl_uid,
    h.name,
    f.short_name,
    f.mfl_uid as historical_id,
    h.id as admin_unit_id,
    h.parent_id,
    h.level,
    fl.mfl_uid as level_mfl_uid,
    fl.name as level_name,
    f.ownership,
    o.mfl_uid as ownership_mfl_uid,
    o.name as ownership_name,
    f.authority,
    a.mfl_uid as authority_mfl_uid,
    a.name as authority_name,
    f.status,
    f.reporting,
    f.licensed,
    f.address,
    f.contact_personemail,
    f.contact_personmobile,
    f.contact_personname,
    f.contact_persontitle,
    f.longitude,
    f.latitude,
    f.opening_date,
    f.closing_date,
    f.bed_capacity,
    f.services,
    h."createdAt",
    h."updatedAt"
FROM hierarchy h
LEFT JOIN facilities f ON h.id = f.admin_unit_id
LEFT JOIN ownership o ON f.ownership = o.mfl_uid
LEFT JOIN authority a ON f.authority = a.mfl_uid
LEFT JOIN "level" fl ON f.level = fl.mfl_uid;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_orgunits_level ON admin_units(level_id);
CREATE INDEX IF NOT EXISTS idx_orgunits_parent ON admin_units(parent_id);
CREATE INDEX IF NOT EXISTS idx_orgunits_mfl_uid ON admin_units(mfl_uid);
CREATE INDEX IF NOT EXISTS idx_facilities_admin_unit ON facilities(admin_unit_id);

-- Grant permissions
GRANT SELECT ON public.orgunits TO PUBLIC;

-- Verify view was created
SELECT COUNT(*) as total_records FROM public.orgunits;
