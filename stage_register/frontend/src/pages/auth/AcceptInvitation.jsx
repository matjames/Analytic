import React, { useState } from "react";
import { useHistory, useLocation } from "react-router-dom";
import { UsersApi } from "../../helpers/api/users";
import { getRoleRoute } from "../../utils/roleRoutes";

export default function AcceptInvitation() {
  const history = useHistory();
  const token = new URLSearchParams(useLocation().search).get("token") || "";
  const [form, setForm] = useState({ first_name: "", last_name: "", username: "", password: "", confirm: "" });
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (event) => {
    event.preventDefault();
    setError("");
    if (form.password !== form.confirm) return setError("Passwords do not match.");
    setBusy(true);
    try {
      const result = await UsersApi.acceptInvitation({ token, ...form });
      localStorage.setItem("token", result.token);
      localStorage.setItem("refresh_token", result.refresh_token);
      localStorage.setItem("user", JSON.stringify(result.user));
      history.replace(getRoleRoute(result.user.role));
    } catch (err) {
      setError(err.response?.data?.error || "Unable to accept invitation.");
    } finally { setBusy(false); }
  };
  return <div className="container py-5" style={{ maxWidth: "32rem" }}><h1 className="h3">Accept invitation</h1><p className="text-muted">Create your StatGate account.</p>{error && <div className="alert alert-danger">{error}</div>}<form className="card card-body" onSubmit={submit}>{["first_name", "last_name", "username"].map((field) => <div className="mb-3" key={field}><label className="form-label" htmlFor={field}>{field.replace("_", " ")}</label><input id={field} className="form-control" required={field === "username"} value={form[field]} onChange={(e) => setForm({ ...form, [field]: e.target.value })} /></div>)}<div className="mb-3"><label className="form-label" htmlFor="invite-password">Password</label><input id="invite-password" className="form-control" type="password" required value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></div><div className="mb-3"><label className="form-label" htmlFor="invite-confirm">Confirm password</label><input id="invite-confirm" className="form-control" type="password" required value={form.confirm} onChange={(e) => setForm({ ...form, confirm: e.target.value })} /></div><button className="btn btn-primary" disabled={busy || !token} type="submit">{busy ? "Creating account..." : "Accept invitation"}</button></form></div>;
}
