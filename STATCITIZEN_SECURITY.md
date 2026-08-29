# StatCitizen — Security Architecture & Threat Model

## 1. Security Architecture Principles

StatCitizen adheres to strict sovereign security standards:
- **Zero Hardcoded Secrets:** All credentials sourced from environment variables.
- **Fail-Closed Validation:** Startup halts in production if required cryptographic secrets or database credentials are missing.
- **Defense in Depth:** Rate limiting, payload size limits, strict security headers, and MIME/size attachment validations.
- **Tenant Isolation:** Multi-tenancy enforced at both database query and routing layers.

---

## 2. Cryptographic Controls

1. **Citizen Session Tokens:** Generated via CSPRNG (`crypto/rand`) yielding 256-bit cryptographically random URL-safe tokens.
2. **PII Anonymization:** Client IP addresses, emails, and phone numbers are hashed using SHA-256 HMAC keyed by `STATCITIZEN_CITIZEN_SESSION_SECRET`.
3. **Enterprise Authentication:** Institutional admin endpoints require valid JWTs verified against `STATGATE_REGISTRY_JWT_SECRET` or internal pre-shared keys (`X-Internal-API-Key`).

---

## 3. Threat Mitigation Matrix

| Threat Vector | Countermeasure | Implementation |
|---|---|---|
| **Bot Flooding & DDoS** | Token Bucket Rate Limiting per IP | `middleware.go` |
| **Data Scraping** | Governed Publication Gate | Internal records isolated from public routes |
| **Payload Overflow** | Max Request Body (10MB limit) | `requestSizeMiddleware` |
| **Cross-Site Attacks** | Strict CSP, HSTS, X-Frame-Options: DENY, nosniff | `securityHeadersMiddleware` |
| **Malicious Uploads** | Size cap (10MB), scan status tracking | `handleUploadAttachment` |
| **Tenant Data Leakage** | Explicit Tenant ID scoping on all SQL queries | `persistence.go` |
