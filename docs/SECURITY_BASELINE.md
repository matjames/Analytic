# StatGate Sovereign Security Baseline & Zero-Default Policy

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Enterprise Security Directive (SG-SEC-ZERO-DEFAULT)  
**Scope:** Universal Security Controls Across All StatGate Backends & Frontends

---

## 1. Zero-Default Security Architecture

StatGate enforces strict zero-default security across all operational environments. The platform rejects insecure fallbacks, hardcoded development passwords, and mock authentication bypasses.

### Mandatory Rules:
1. **No Insecure Fallbacks:** If `STATGATE_REGISTRY_JWT_SECRET` is missing in an enterprise deployment, the service MUST fail startup immediately (`log.Fatalf`).
2. **No Mock Identities:** Mock authentication strings (`demo_`, `mock_`, `bypass_`) are rejected by middleware with `401 Unauthorized`.
3. **No Hardcoded Passwords:** Database passwords and MinIO storage keys must be injected strictly via environment variables or Docker secrets.
4. **Tenant Isolation:** Requests containing an `X-Tenant-ID` header that does not match the cryptographic `tenant_id` claim in the verified JWT token are terminated with `403 Forbidden`.

---

## 2. Cryptographic Identity & JWT Validation Standard

- **Algorithm:** HMAC-SHA256 (`HS256`).
- **Minimum Secret Length:** 32 high-entropy characters (256-bit).
- **Mandatory Claims:**
  - `userId` / `sub`: Unique principal identifier.
  - `email`: Authenticated institutional email address.
  - `tenant_id`: Multi-tenant boundary slug.
  - `org_id`: Institutional organization unit.
  - `role`: Canonical role (e.g. `admin`, `manager`, `editor`, `operator`, `analyst`, `viewer`, `agent`, `district_admin`).
  - `exp`: Expiration timestamp (maximum 24-hour lifetime).
  - `iss`: Issuer (`statgate-registry`).
  - `aud`: Audience (`statgate`).

---

## 3. Standard Role & Permission Matrix (RBAC)

| Role | Read | Write | Execute / Workflow | Approve | Publish | Audit | Spatial Access | Admin |
|---|---|---|---|---|---|---|---|---|
| `viewer` | YES | NO | NO | NO | NO | NO | NO | NO |
| `analyst` | YES | NO | NO | NO | NO | NO | YES | NO |
| `editor` | YES | YES | NO | NO | NO | NO | YES | NO |
| `agent` | YES | YES | NO | NO | NO | NO | NO | NO |
| `operator` | YES | YES | YES | NO | NO | NO | YES | NO |
| `manager` | YES | YES | YES | YES | NO | NO | YES | NO |
| `district_admin` | YES | YES | YES | YES | NO | NO | YES | YES |
| `governance_officer` | YES | NO | YES | YES | NO | YES | NO | NO |
| `admin` | YES | YES | YES | YES | YES | YES | YES | YES |
| `platform_admin` | YES | YES | YES | YES | YES | YES | YES | YES |

---

## 4. Immutable Enterprise Audit Policy

All state-modifying operations (create, update, delete, transition, approve, reject) must record structured immutable audit entries in `enterprise_audit_log`:
- **Who:** Authenticated `user_id` and role.
- **What:** Action identifier (e.g. `pms.project.created`).
- **When:** UTC timestamp (`time.Now().UTC()`).
- **Where:** Component host, path, and service identifier.
- **Tenant:** Multi-tenant slug.
- **Object:** Target universal identifier (`tenant:app:type:id`).
- **States:** `previous_state` and `new_state` JSON diffs.
- **Network Context:** Client IP and User-Agent.
- **Correlation:** Request ID and correlation trace token.
