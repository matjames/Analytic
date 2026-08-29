# Synthetic ODK volume fixture

This fixture registers 10 multi-sector ODK-style questionnaire templates and loads
`100,001` submissions per survey by default (`1,000,010` total). Each template has
40 declared XLSForm-compatible field types. Submission metadata contains 50+ keys,
including text, integers, decimals, booleans, dates, times, datetimes, select-one,
select-multiple, GPS/geospatial values, barcode, calculated values, device metadata,
and media filenames. All values are synthetic.

Run from the repository root after the StatCollect PostgreSQL container is running:

```powershell
.\StatCollect\tests\load\load_odk_synthetic_surveys.ps1
```

If the Compose project generated another database container name:

```powershell
.\StatCollect\tests\load\load_odk_synthetic_surveys.ps1 -Container <postgres-container-name>
```

The loader is deterministic and safe to rerun: it replaces only rows whose tenant is
`synthetic-load-test` and whose instance ID starts with `uuid:synthetic-`. Use
`-RowsPerSurvey` for a larger load and `-TenantId` to isolate a test run.

To copy the already-loaded fixture into the main StatGate database, run
`copy_odk_surveys_to_main.sql` against `statgate_ml_staging`. It creates the native
`public.templates` and `public.submissions` tables when absent and uses a temporary
PostgreSQL foreign-data connection to preserve the exact JSONB and XML payloads.

For a quick validation without a million-row load, psql can run the SQL directly with
a smaller value (the PowerShell wrapper intentionally enforces the requested minimum):

```powershell
Get-Content .\StatCollect\tests\load\odk_synthetic_surveys.sql -Raw |
  docker exec -i <postgres-container-name> psql -v rows_per_survey=10 -v tenant_id=synthetic-smoke -U statcollect -d statcollect
```
