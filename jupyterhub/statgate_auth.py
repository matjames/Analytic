"""StatGate JupyterHub authenticator.

Replaces the removed DummyAuthenticator ("password" / "password").

Design (SG-SEC-2026-08):
- JupyterHub users authenticate through the StatGate Registry identity
  service (the platform's identity authority).
- The authenticator fails CLOSED: if the Registry URL or credentials are
  missing/unreachable, authentication is refused (never silently succeeds).
- Role-based admin grants come from the Registry user record.

The Hub itself only runs in the "analytics-labs" profile of the platform.
"""

import os

import requests
from jupyterhub.auth import Authenticator
from traitlets import Int, Unicode


class StatGateAuthenticator(Authenticator):
    """Authenticates a user against the StatGate Registry."""

    registry_url = Unicode(
        config=True,
        help="Base URL of the StatGate Registry API (e.g. http://statgate-registry-api:9090/api).",
    )

    admin_roles = Unicode(
        config=True,
        default_value="admin,superadmin,platform_admin",
        help="Registry role values that grant JupyterHub administrative rights.",
    )

    hub_timeout = Int(
        config=True,
        default_value=15,
        help="Timeout in seconds for Registry authentication calls.",
    )

    async def authenticate(self, handler, data):
        registry = (self.registry_url or os.getenv("STATGATE_REGISTRY_API_URL", "")).rstrip("/")
        if not registry:
            self.log.error("StatGateAuthenticator: STATGATE_REGISTRY_API_URL is not configured")
            return None

        username = (data.get("username") or "").strip()
        password = data.get("password") or ""
        if not username or not password:
            return None

        internal_key = os.getenv("STATGATE_INTERNAL_API_KEY", "")
        headers = {"Content-Type": "application/json"}
        if internal_key:
            headers["X-StatGate-Internal-Key"] = internal_key

        payload = {"emailOrUsername": username, "password": password}
        try:
            resp = await self._post_async(
                registry + "/users/login",
                json=payload,
                headers=headers,
                timeout=self.hub_timeout,
            )
        except Exception as exc:  # noqa: BLE001 - fail closed on any error
            self.log.info("StatGateAuthenticator: registry login call failed: %s", exc)
            return None

        if resp.status_code not in (200, 201):
            self.log.info(
                "StatGateAuthenticator: registry rejected login for %s (status %s)",
                username,
                resp.status_code,
            )
            return None

        try:
            body = resp.json()
        except Exception:  # noqa: BLE001
            return None

        user = body.get("user")
        if not isinstance(user, dict):
            return None

        normalized_name = str(user.get("username") or username)
        role = str(user.get("role") or "").lower()
        admin = role in [r.strip().lower() for r in self.admin_roles.split(",") if r.strip()]
        auth_state = {
            "statgate_user": user,
            "access_token": body.get("token"),
        }
        return {
            "name": normalized_name,
            "admin": admin,
            "auth_state": auth_state,
        }

    def _post_async(self, url, json=None, headers=None, timeout=15):
        import asyncio

        async def _run():
            loop = asyncio.get_event_loop()
            return await loop.run_in_executor(
                None,
                lambda: requests.post(url, json=json, headers=headers, timeout=timeout),
            )

        return _run()