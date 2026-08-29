import React, { useEffect, useState } from "react";

const ENTERPRISE_API = process.env.REACT_APP_ENTERPRISE_API_URL || "http://localhost:8096/api";

function enterpriseHeaders() {
  const token = localStorage.getItem("token") || localStorage.getItem("registry_jwt");
  return token ? { Authorization: `Bearer ${token}`, "Content-Type": "application/json" } : { "Content-Type": "application/json" };
}

export default function PermissionManager({ mode = "roles" }) {
  const [roles, setRoles] = useState([]);
  const [policies, setPolicies] = useState([]);
  const [form, setForm] = useState({ role: "", name: "", description: "", resource: "", actions: "read" });
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const load = async () => {
    try {
      const [rolesResponse, policiesResponse] = await Promise.all([
        fetch(`${ENTERPRISE_API}/permissions/roles`, { headers: enterpriseHeaders() }),
        fetch(`${ENTERPRISE_API}/permissions/policies`, { headers: enterpriseHeaders() }),
      ]);
      if (!rolesResponse.ok || !policiesResponse.ok) throw new Error("Permission service unavailable");
      const rolesData = await rolesResponse.json();
      const policiesData = await policiesResponse.json();
      setRoles(rolesData.roles || []);
      setPolicies(policiesData.policies || []);
    } catch (err) {
      setError(err.message || "Unable to load roles and policies.");
    }
  };

  useEffect(() => { load(); }, []);

  const create = async (event) => {
    event.preventDefault();
    setError("");
    const isRole = mode === "roles";
    const payload = isRole
      ? { role: form.role.trim(), name: form.name.trim(), description: form.description.trim() }
      : { name: form.name.trim(), description: form.description.trim(), role: form.role.trim(), resource: form.resource.trim(), actions: form.actions.split(",").map((action) => action.trim()).filter(Boolean) };
    try {
      const response = await fetch(`${ENTERPRISE_API}/permissions/${isRole ? "roles" : "policies"}`, { method: "POST", headers: enterpriseHeaders(), body: JSON.stringify(payload) });
      if (!response.ok) throw new Error((await response.json()).error || "Unable to save permission configuration.");
      setMessage(`${isRole ? "Role" : "Policy"} created.`);
      setForm({ role: "", name: "", description: "", resource: "", actions: "read" });
      load();
    } catch (err) { setError(err.message || "Unable to save permission configuration."); }
  };

  return <div className="container-fluid py-4"><h1 className="h3">{mode === "roles" ? "Roles" : "Permission policies"}</h1><p className="text-muted">Manage the platform authorization catalogue used by supported services.</p>{message && <div className="alert alert-success">{message}</div>}{error && <div className="alert alert-danger">{error}</div>}<form className="card card-body mb-4" onSubmit={create}><div className="row g-3 align-items-end"><div className="col-md-3"><label className="form-label" htmlFor="permission-name">{mode === "roles" ? "Role key" : "Policy name"}</label><input id="permission-name" className="form-control" required value={mode === "roles" ? form.role : form.name} onChange={(e) => setForm({ ...form, ...(mode === "roles" ? { role: e.target.value } : { name: e.target.value }) })} /></div>{mode === "policies" && <div className="col-md-2"><label className="form-label" htmlFor="permission-role">Role</label><input id="permission-role" className="form-control" required value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} /></div>}{mode === "policies" && <div className="col-md-2"><label className="form-label" htmlFor="permission-resource">Resource</label><input id="permission-resource" className="form-control" required value={form.resource} onChange={(e) => setForm({ ...form, resource: e.target.value })} /></div>}{mode === "policies" && <div className="col-md-2"><label className="form-label" htmlFor="permission-actions">Actions</label><input id="permission-actions" className="form-control" value={form.actions} onChange={(e) => setForm({ ...form, actions: e.target.value })} /></div>}<div className="col-md-3"><label className="form-label" htmlFor="permission-description">Description</label><input id="permission-description" className="form-control" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></div><div className="col-auto"><button type="submit" className="btn btn-primary">Create</button></div></div></form>{mode === "roles" ? <div className="table-responsive"><table className="table table-striped"><thead><tr><th>Role</th><th>Name</th><th>Description</th></tr></thead><tbody>{roles.map((item) => <tr key={item.role}><td><code>{item.role}</code></td><td>{item.name}</td><td>{item.description}</td></tr>)}</tbody></table></div> : <div className="table-responsive"><table className="table table-striped"><thead><tr><th>Name</th><th>Role</th><th>Resource</th><th>Actions</th></tr></thead><tbody>{policies.map((item) => <tr key={item.id}><td>{item.name}</td><td>{item.role}</td><td>{item.resource}</td><td>{(item.actions || []).join(", ")}</td></tr>)}</tbody></table></div>}</div>;
}
