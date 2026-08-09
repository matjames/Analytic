@echo off
echo ============================================
echo  StatCollect Questionnaire Test Suite
echo  Community Health Survey
echo ============================================
echo.

set BASE=http://localhost:8080
set API_KEY=changeme

echo [1/8] Submitting Sample 1 - Kampala...
curl -s -X POST -H "X-API-Key: %API_KEY%" -F "xml_submission_file=@%~dp0submissions\sample1_kampala.xml" %BASE%/submission
echo.

echo [2/8] Submitting Sample 2 - Wakiso...
curl -s -X POST -H "X-API-Key: %API_KEY%" -F "xml_submission_file=@%~dp0submissions\sample2_wakiso.xml" %BASE%/submission
echo.

echo [3/8] Submitting Sample 3 - Gulu...
curl -s -X POST -H "X-API-Key: %API_KEY%" -F "xml_submission_file=@%~dp0submissions\sample3_gulu.xml" %BASE%/submission
echo.

echo [4/8] Testing idempotency - resubmit Sample 1 (should say "already received")...
curl -s -X POST -H "X-API-Key: %API_KEY%" -F "xml_submission_file=@%~dp0submissions\sample1_kampala.xml" %BASE%/submission
echo.

echo [5/8] Listing all submissions...
curl -s -H "X-API-Key: %API_KEY%" "%BASE%/admin/submissions?per_page=10&page=1"
echo.

echo [6/8] Approving Sample 1...
curl -s -X POST -H "X-API-Key: %API_KEY%" "%BASE%/admin/submission/validate?instance_id=uuid:health-survey-001-kampala&status=approved&notes=All+data+verified"
echo.

echo [7/8] Rejecting Sample 2...
curl -s -X POST -H "X-API-Key: %API_KEY%" "%BASE%/admin/submission/validate?instance_id=uuid:health-survey-002-wakiso&status=rejected&notes=Missing+consent+documentation"
echo.

echo [8/8] Viewing event log...
curl -s -H "X-API-Key: %API_KEY%" "%BASE%/admin/events?limit=10"
echo.

echo.
echo ============================================
echo  Test Suite Complete
echo ============================================
echo.
echo Open the admin console at http://localhost:8080/admin/
echo to view the submissions in the browser.