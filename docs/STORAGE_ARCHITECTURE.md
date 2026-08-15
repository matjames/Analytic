# StatGate Shared Object & File Storage Architecture

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Enterprise File Storage Standard  
**Infrastructure:** MinIO S3-Compatible Object Cluster  
**Directives:** Section 10 Shared File Infrastructure, Elimination of Local Container File Storage

---

## 1. Executive Summary

StatGate establishes MinIO S3-compatible object storage as the standard storage tier for all institutional artifacts, research proposals, survey datasets, HelpDesk attachments, project documentation, spatial layers, and evidence files.

Storing files directly on local container file systems or ephemeral disks is prohibited for institutional assets.

---

## 2. Storage Bucket Topology & Hierarchy

All objects are organized within tenant-partitioned object namespaces:

```text
minio://statgate-files/
├── {tenant_id}/
│   ├── pms/
│   │   └── {project_id}/
│   │       └── {file_uuid}-{filename}
│   ├── rms/
│   │   └── {research_id}/
│   │       └── {file_uuid}-protocol.pdf
│   ├── statcollect/
│   │   └── {survey_id}/
│   │       └── {submission_id}/
│   │           └── media-{file_uuid}.jpg
│   ├── helpdesk/
│   │   └── {ticket_id}/
│   │       └── screenshot-{file_uuid}.png
│   ├── statspatial/
│   │   └── layers/
│   │       └── boundary-polygons-{file_uuid}.geojson
│   └── evidence/
│       └── {decision_id}/
│           └── audit-evidence-{file_uuid}.pdf
```

---

## 3. Mandatory File Governance Metadata

Every object uploaded via `statgate-lib/storage` records cryptographic verification and governance attributes:

- **ID:** Unique UUID v4.
- **Bucket:** S3 bucket target (`statgate-files`).
- **ObjectName:** Tenant and application structured key.
- **TenantID:** Tenant partition key.
- **OwnerID:** Authenticated user ID of the uploader.
- **SourceApp:** Source application identifier (`pms`, `rms`, `statcollect`, etc.).
- **AssociatedObj:** Universal URN of the associated entity (e.g. `uganda-national:pms:project:10492`).
- **ContentType:** Standard MIME type.
- **SizeBytes:** File size in bytes.
- **SHA256Checksum:** Hex-encoded SHA-256 cryptographic digest computed during upload stream.
- **Classification:** Security classification (`public`, `internal`, `confidential`, `restricted`).
- **RetentionUntil:** Regulatory data retention expiration date.
- **CreatedAt:** UTC upload timestamp.

---

## 4. Integrity & Security Verification

1. **In-Flight Checksumming:** Streamed bytes are piped through `crypto/sha256` in memory; mismatched checksums abort the upload.
2. **Access Policy:** Object retrieval validates caller tenant matching the object tenant prefix.
3. **Audit Integration:** File creation, download, and deletion trigger immutable audit events in `enterprise_audit_log`.
