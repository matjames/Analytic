/**
 * Returns the current token only if it is still valid (JWT not expired).
 * If the token is expired, clears token and user from localStorage and returns null.
 * Use this whenever reading the token so that expiry results in a clean logout.
 */
export function getValidToken() {
  const token = localStorage.getItem("token");
  if (!token) return null;

  try {
    const parts = token.split(".");
    if (parts.length !== 3) {
      return token;
    }

    const payloadBase64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const payloadJson = atob(payloadBase64);
    const payload = JSON.parse(payloadJson);

    if (payload && typeof payload.exp === "number") {
      const nowMs = Date.now();
      const expMs = payload.exp * 1000;
      if (expMs > nowMs) {
        return token;
      }

      // Token expired: clear storage so UI and API stop using stale auth
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      window.dispatchEvent(new CustomEvent("auth-expired"));
      return null;
    }

    return token;
  } catch (e) {
    return token;
  }
}
