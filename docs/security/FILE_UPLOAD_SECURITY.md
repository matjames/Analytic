# StatGate File & Upload Security (SG-SEC-2026-08)

## Required controls

Every upload path applies:

- filename sanitization: generated storage names (uuid/timestamp prefixes),
  never raw client filenames for storage keys.
- MIME verification: content-based detection where supported; client
  Content-Type and filename are never trusted.
- maximum size enforcement (helpdesk avatars 5 MB; video and StatChat capped).
- non-executable storage: uploaded files are served from an uploads volume,
  not a webroot; no execution of uploaded content.
- authorization before download: live endpoints (enterprise /api/files/:id)
  sit behind authenticated routes.
- tenant isolation: file records carry the uploader/tenant at upload time.

## Current posture

- Helpdesk: avatar/video/uploads stored under uploads/ with multer-generated
  names; knowledge-base PDFs use UUID names. Uploads are served statically
  for browser media; all admin document endpoints are authenticated.
- Enterprise core: file records persisted via Redis with generated IDs and
  sanitized names; /api/files/* routes require a verified JWT.
- StatChat: attachment uploads enforce an allowlist of content types and
  write generated names under the configured upload directory.

## Residual risk

Static /uploads hosting of knowledge-base PDFs and avatars is intentional for
browser access; classified PDF documents should move behind an auth gate.
Rate limiting on uploads is enforced by the request-size middleware.