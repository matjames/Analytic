import React, { useEffect, useState } from "react";
import { UsersApi } from "../../../helpers/api/users";

const ENTERPRISE_API = process.env.REACT_APP_ENTERPRISE_API_URL || "http://localhost:8096/api";

function headers() {
  const token = localStorage.getItem("token") || localStorage.getItem("registry_jwt");
  return token ? { Authorization: `Bearer ${token}`, "Content-Type": "application/json" } : {};
}

export default function WorkspaceManager() {
  const [workspaces, setWorkspaces] = useState([]);
  const [members, setMembers] = useState([]);
  const [users, setUsers] = useState([]);
  const [selected, setSelected] = useState("");
  const [form, setForm] = useState({ name: "", slug: "", description: "" });
  const [member, setMember] = useState({ user_id: "", role: "member" });
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const loadWorkspaces = async () => {
    const response = await fetch(`${ENTERPRISE_API}/workspaces`, { headers: headers() });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Unable to load workspaces.");
    setWorkspaces(data.workspaces || []);
    if (!selected && data.workspaces?.[0]) setSelected(data.workspaces[0].id);
  };

  const loadMembers = async (workspaceID) => {
    if (!workspaceID) return setMembers([]);
    const response = await fetch(`${ENTERPRISE_API}/workspaces/${workspaceID}/members`, { headers: headers() });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Unable to load workspace members.");
    setMembers(data.members || []);
  };

  // Load the directory and memberships once when entering workspace administration.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    Promise.all([loadWorkspaces(), UsersApi.list()]).then(([, directory]) => setUsers(directory.users || directory || [])).catch((err) => setError(err.message || "Unable to load workspace administration."));
  }, []);

  useEffect(() => { loadMembers(selected).catch((err) => setError(err.message)); }, [selected]);

  const create = async (event) => {
    event.preventDefault(); setError("");
    try {
      const response = await fetch(`${ENTERPRISE_API}/workspaces`, { method: "POST", headers: headers(), body: JSON.stringify(form) });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Unable to create workspace.");
      setForm({ name: "", slug: "", description: "" }); setMessage("Workspace created."); await loadWorkspaces(); setSelected(data.workspace.id);
    } catch (err) { setError(err.message); }
  };

  const addMember = async (event) => {
    event.preventDefault(); setError("");
    try {
      const response = await fetch(`${ENTERPRISE_API}/workspaces/${selected}/members`, { method: "POST", headers: headers(), body: JSON.stringify(member) });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Unable to add member.");
      setMessage("Workspace member saved."); setMember({ user_id: "", role: "member" }); await loadMembers(selected);
    } catch (err) { setError(err.message); }
  };

  return <div className="container-fluid py-4"><h1 className="h3">Workspaces</h1><p className="text-muted">Create tenant workspaces and manage their active members.</p>{message && <div className="alert alert-success">{message}</div>}{error && <div className="alert alert-danger">{error}</div>}<div className="row g-4"><div className="col-lg-5"><form className="card card-body" onSubmit={create}><h2 className="h5">Create workspace</h2><input className="form-control mb-2" placeholder="Name" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /><input className="form-control mb-2" placeholder="slug-example" pattern="[a-z0-9][a-z0-9-]{1,62}" required value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} /><textarea className="form-control mb-3" placeholder="Description" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /><button className="btn btn-primary" type="submit">Create workspace</button></form><div className="card card-body mt-4"><h2 className="h5">Your workspaces</h2>{workspaces.length === 0 ? <p className="text-muted mb-0">No workspaces available.</p> : <select className="form-select" value={selected} onChange={(e) => setSelected(e.target.value)}>{workspaces.map((workspace) => <option key={workspace.id} value={workspace.id}>{workspace.name} ({workspace.role})</option>)}</select>}</div></div><div className="col-lg-7"><form className="card card-body mb-4" onSubmit={addMember}><h2 className="h5">Add or restore member</h2><div className="row g-2"><div className="col-md-7"><select className="form-select" required value={member.user_id} onChange={(e) => setMember({ ...member, user_id: e.target.value })}><option value="">Select user</option>{users.map((user) => <option key={user.id} value={user.id}>{user.username || user.email} ({user.id})</option>)}</select></div><div className="col-md-3"><select className="form-select" value={member.role} onChange={(e) => setMember({ ...member, role: e.target.value })}><option value="member">Member</option><option value="admin">Admin</option></select></div><div className="col-md-2"><button className="btn btn-primary w-100" disabled={!selected} type="submit">Save</button></div></div></form><div className="card card-body"><h2 className="h5">Members</h2><div className="table-responsive"><table className="table table-sm"><thead><tr><th>User</th><th>Role</th><th>Status</th></tr></thead><tbody>{members.map((item) => <tr key={item.user_id}><td>{item.user_id}</td><td>{item.role}</td><td>{item.status}</td></tr>)}</tbody></table></div></div></div></div></div>;
}
