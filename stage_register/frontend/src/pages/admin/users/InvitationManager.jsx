import React, { useEffect, useState } from "react";
import { UsersApi } from "../../../helpers/api/users";

export default function InvitationManager() {
  const [invitations, setInvitations] = useState([]);
  const [form, setForm] = useState({ email: "", role: "viewer", organisation: "", expires_in_days: 7 });
  const [created, setCreated] = useState(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const load = () => UsersApi.listInvitations().then(setInvitations).catch(() => setError("Unable to load invitations."));
  useEffect(() => { load(); }, []);

  const submit = async (event) => {
    event.preventDefault();
    setError("");
    setMessage("");
    try {
      const result = await UsersApi.createInvitation(form);
      setCreated(result);
      setForm({ ...form, email: "" });
      setMessage("Invitation created. Share the acceptance link with the invitee.");
      load();
    } catch (err) {
      setError(err.response?.data?.error || "Unable to create invitation.");
    }
  };

  const revoke = async (id) => {
    try {
      await UsersApi.revokeInvitation(id);
      setMessage("Invitation revoked.");
      load();
    } catch (err) {
      setError(err.response?.data?.error || "Unable to revoke invitation.");
    }
  };

  return (
    <div className="container-fluid py-4">
      <h1 className="h3 mb-1">User invitations</h1>
      <p className="text-muted">Invite people into an approved role without opening privileged public registration.</p>
      {message && <div className="alert alert-success">{message}</div>}
      {error && <div className="alert alert-danger">{error}</div>}
      {created && (
        <div className="alert alert-info">
          <strong>Acceptance link</strong>
          <code className="d-block mt-2 text-break">{created.accept_url}</code>
        </div>
      )}
      <form className="card card-body mb-4" onSubmit={submit}>
        <div className="row g-3 align-items-end">
          <div className="col-md-4"><label className="form-label" htmlFor="invite-email">Email</label><input id="invite-email" className="form-control" type="email" required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
          <div className="col-md-2"><label className="form-label" htmlFor="invite-role">Role</label><select id="invite-role" className="form-select" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}><option>viewer</option><option>analyst</option><option>editor</option><option>operator</option><option>manager</option><option>agent</option><option>district</option><option>district_admin</option><option>tenant_admin</option><option>governance_officer</option></select></div>
          <div className="col-md-3"><label className="form-label" htmlFor="invite-org">Organisation</label><input id="invite-org" className="form-control" value={form.organisation} onChange={(e) => setForm({ ...form, organisation: e.target.value })} /></div>
          <div className="col-md-2"><label className="form-label" htmlFor="invite-days">Expires in days</label><input id="invite-days" className="form-control" type="number" min="1" max="30" value={form.expires_in_days} onChange={(e) => setForm({ ...form, expires_in_days: Number(e.target.value) })} /></div>
          <div className="col-md-1"><button className="btn btn-primary w-100" type="submit">Invite</button></div>
        </div>
      </form>
      <div className="table-responsive"><table className="table table-striped align-middle"><thead><tr><th>Email</th><th>Role</th><th>Expires</th><th>Status</th><th /></tr></thead><tbody>{invitations.map((item) => { const active = !item.accepted_at && new Date(item.expires_at) > new Date(); return <tr key={item.id}><td>{item.email}</td><td>{item.role}</td><td>{new Date(item.expires_at).toLocaleString()}</td><td>{item.accepted_at ? "Accepted" : active ? "Pending" : "Expired"}</td><td>{active && <button className="btn btn-sm btn-outline-danger" type="button" onClick={() => revoke(item.id)}>Revoke</button>}</td></tr>; })}</tbody></table></div>
    </div>
  );
}
