import os

from statgate_auth import StatGateAuthenticator

# ── Authentication (SG-SEC-2026-08) ──────────────────────────────
# DummyAuthenticator with a fixed shared password was REMOVED.
c.JupyterHub.authenticator_class = StatGateAuthenticator

registry_api = os.getenv("STATGATE_REGISTRY_API_URL", "").rstrip("/")
if not registry_api:
    raise SystemExit(
        "FATAL: STATGATE_REGISTRY_API_URL must be set for JupyterHub. "
        "The Hub refuses to start without a configured identity provider."
    )
c.StatGateAuthenticator.registry_url = registry_api
c.StatGateAuthenticator.admin_roles = os.getenv(
    "STATGATE_JUPYTERHUB_ADMIN_ROLES", "admin,superadmin,platform_admin"
)

cookie_secret = os.getenv("JUPYTERHUB_COOKIE_SECRET", "").strip()
if not cookie_secret:
    raise SystemExit("FATAL: JUPYTERHUB_COOKIE_SECRET must be set. Refusing to start.")
c.JupyterHub.cookie_secret = cookie_secret.encode("utf-8")

c.Spawner.default_url = "/lab"
c.JupyterHub.ip = "0.0.0.0"
c.JupyterHub.port = 8000
