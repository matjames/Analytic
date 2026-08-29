import React, { useEffect, useState } from "react";

const ENTERPRISE_API = process.env.REACT_APP_ENTERPRISE_API_URL || "http://localhost:8096/api";

function headers() {
  const token = localStorage.getItem("token") || localStorage.getItem("registry_jwt");
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export default function AuditLogViewer() {
  const [entries, setEntries] = useState([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [filters, setFilters] = useState({ action: "", resource: "" });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const pageSize = 50;

  const load = async (nextOffset = offset) => {
    setLoading(true);
    setError("");
    const params = new URLSearchParams({ limit: String(pageSize), offset: String(nextOffset) });
    if (filters.action.trim()) params.set("action", filters.action.trim());
    if (filters.resource.trim()) params.set("resource", filters.resource.trim());
    try {
      const response = await fetch(`${ENTERPRISE_API}/apis/audit?${params}`, { headers: headers() });
      const data = await response.json();
      if (!response.ok) throw new Error(data.message || data.error || "Unable to load audit log.");
      setEntries(data.entries || []);
      setTotal(Number(data.total || 0));
      setOffset(nextOffset);
    } catch (err) {
      setError(err.message || "Unable to load audit log.");
    } finally {
      setLoading(false);
    }
  };

  // Load the first page once; subsequent loads are explicit refresh/filter actions.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load(0); }, []);

  const submit = (event) => {
    event.preventDefault();
    load(0);
  };

  return (
    <div className="container-fluid py-4">
      <div className="d-flex justify-content-between align-items-center mb-3">
        <div><h1 className="h3 mb-1">Audit log</h1><p className="text-muted mb-0">Review activity for the active tenant.</p></div>
        <button type="button" className="btn btn-outline-primary" onClick={() => load(offset)} disabled={loading}>Refresh</button>
      </div>
      {error && <div className="alert alert-danger">{error}</div>}
      <form className="card card-body mb-4" onSubmit={submit}>
        <div className="row g-3 align-items-end">
          <div className="col-md-4"><label className="form-label" htmlFor="audit-action">Action contains</label><input id="audit-action" className="form-control" value={filters.action} onChange={(e) => setFilters({ ...filters, action: e.target.value })} /></div>
          <div className="col-md-4"><label className="form-label" htmlFor="audit-resource">Resource</label><input id="audit-resource" className="form-control" value={filters.resource} onChange={(e) => setFilters({ ...filters, resource: e.target.value })} /></div>
          <div className="col-auto"><button type="submit" className="btn btn-primary" disabled={loading}>Apply filters</button></div>
        </div>
      </form>
      <div className="table-responsive">
        <table className="table table-striped align-middle">
          <thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Resource</th><th>Outcome</th><th>Request</th></tr></thead>
          <tbody>
            {!loading && entries.length === 0 && <tr><td colSpan="6" className="text-center text-muted py-4">No audit events found.</td></tr>}
            {entries.map((entry, index) => <tr key={`${entry.created_at}-${entry.request_id || index}`}><td>{entry.created_at ? new Date(entry.created_at).toLocaleString() : "-"}</td><td><code>{entry.actor || "-"}</code></td><td>{entry.action}</td><td>{entry.resource}{entry.resource_id ? ` / ${entry.resource_id}` : ""}</td><td>{entry.outcome || "-"}</td><td><code>{entry.request_id || "-"}</code></td></tr>)}
          </tbody>
        </table>
      </div>
      <div className="d-flex justify-content-between align-items-center mt-3"><small className="text-muted">Showing {entries.length ? offset + 1 : 0}-{offset + entries.length} of {total}</small><div className="btn-group"><button type="button" className="btn btn-outline-secondary" onClick={() => load(Math.max(0, offset - pageSize))} disabled={loading || offset === 0}>Previous</button><button type="button" className="btn btn-outline-secondary" onClick={() => load(offset + pageSize)} disabled={loading || offset + pageSize >= total}>Next</button></div></div>
    </div>
  );
}
