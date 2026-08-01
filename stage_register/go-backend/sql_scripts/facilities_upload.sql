-- Script to populate public.facilities table from public.mflupload
-- This script maps columns from mflupload to facilities table structure

-- First, verify the mflupload table has data
-- SELECT COUNT(*) FROM public.mflupload;

-- TRUNCATE TABLE public.facilities RESTART IDENTITY CASCADE;

-- Insert data from mflupload into facilities
INSERT INTO public.facilities (
    identifier,
    mfl_uid,
    "name",
    short_name,
    historical_id,
    admin_unit_id,
    "level",
    ownership,
    authority,
    status,
    reporting,
    licensed,
    address,
    contact_personemail,
    contact_personmobile,
    contact_personname,
    contact_persontitle,
    longitude,
    latitude,
    opening_date,
    closing_date,
    bed_capacity,
    services,
    "createdAt",
    "updatedAt",
    user_id
)
SELECT 
    m.nhfrid AS identifier,              
    m.uid AS mfl_uid,                 
    m."name",                         
    m.shortname AS short_name,       
    m.uid AS historical_id,        
    au.id AS admin_unit_id,           
    l.mfl_uid AS "level",            
    o.mfl_uid AS ownership,           
    a.mfl_uid AS authority,           
    m.status,                        
    CASE 
        WHEN LOWER(m.report) IN ('reporting', 'yes', 'true', '1') THEN TRUE
        WHEN LOWER(m.report) IN ('not reporting', 'no', 'false', '0') THEN FALSE
        ELSE NULL
    END AS reporting,                 
    NULL AS licensed,                 
    NULL AS address,                  
    NULL AS contact_personemail,      
    NULL AS contact_personmobile,     
    NULL AS contact_personname,       
    NULL AS contact_persontitle,      
    CASE 
        WHEN m."longtitude" ~ '^-?[0-9]+\.?[0-9]*$' THEN CAST(m."longtitude" AS NUMERIC)
        ELSE NULL 
    END AS longitude,                 
    CASE 
        WHEN m."latitude" ~ '^-?[0-9]+\.?[0-9]*$' THEN CAST(m."latitude" AS NUMERIC)
        ELSE NULL 
    END AS latitude,                  
    NULL AS opening_date,             
    NULL AS closing_date,             
    NULL AS bed_capacity,             
    NULL AS services,                 
    CURRENT_TIMESTAMP AS "createdAt", 
    CURRENT_TIMESTAMP AS "updatedAt", 
    NULL AS user_id                   
FROM public.mflupload m
LEFT JOIN public."level" l ON l.code = m.hflevel OR l.name = m.hflevel OR l.mfl_uid = m.hflevel
LEFT JOIN public.ownership o ON o.code = m.ownership OR o.name = m.ownership OR o.mfl_uid = m.ownership
LEFT JOIN public.authority a ON a.code = m.authority OR a.name = m.authority OR a.mfl_uid = m.authority
LEFT JOIN public.admin_units au ON au.mfl_uid = m.uid OR au.name = m.name
WHERE m.uid IS NOT NULL               
ORDER BY m.id;
