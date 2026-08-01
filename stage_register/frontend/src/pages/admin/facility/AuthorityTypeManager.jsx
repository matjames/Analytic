import React, { Fragment, useEffect, useState } from "react";
import { AuthorityTypesApi } from "../../../helpers/api";

export default function AuthorityTypeManager() {
  const [types, setTypes] = useState([]);
  const [form, setForm] = useState({ code: "", name: "", description: "" });
  const [editingId, setEditingId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const rows = await AuthorityTypesApi.list();
      setTypes(rows);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load authority types");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  function resetForm() {
    setForm({ code: "", name: "", description: "" });
    setEditingId(null);
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.code.trim() || !form.name.trim()) return;
    setSaving(true);
    setError("");
    try {
      if (editingId) {
        await AuthorityTypesApi.update(editingId, form);
      } else {
        await AuthorityTypesApi.create(form);
      }
      resetForm();
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save authority type");
    } finally {
      setSaving(false);
    }
  }

  function startEdit(type) {
    setForm({
      code: type.code || "",
      name: type.name || "",
      description: type.description || "",
    });
    setEditingId(type.id);
  }

  async function handleDelete(id) {
    if (!window.confirm("Delete this authority type? This may fail if facilities are using it.")) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await AuthorityTypesApi.remove(id);
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete authority type");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Fragment>
      <div class="page-header">
        <div class="page-title">
          <h2>Authority Types</h2>
          <div class="page-subtitle">Manage field agencies and partner organisations.</div>
        </div>
      </div>

      {error && <div className="alert alert-danger py-1 small mb-3">{error}</div>}

      <div class="filters-section">
        <form onSubmit={handleSubmit} style={{ display: "flex", gap: "0.8rem", flexWrap: "wrap", alignItems: "center", width: "100%" }}>
          <div class="filter-group">
            <label>Code:</label>
            <input
              type="text"
              placeholder="e.g. SGA"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
              required
            />
          </div>
          <div class="filter-group">
            <label>Name:</label>
            <input
              type="text"
              placeholder="e.g. StatGate Agency"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div class="filter-group">
            <label>Description:</label>
            <input
              type="text"
              placeholder="Optional description"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </div>
          <button type="submit" class="btn-action" disabled={saving}>
            {saving ? "Saving..." : editingId ? "Update" : "Add"}
          </button>
          {editingId && (
            <button
              type="button"
              class="btn-clear"
              onClick={resetForm}
            >
              Cancel
            </button>
          )}
        </form>
      </div>

      <div className="table-section">
        <div className="table-header">
          <h3>
            <i className="bi bi-building-check"></i> Authority Types ({types.length} {types.length === 1 ? 'type' : 'types'})
          </h3>
        </div>
        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "var(--text-muted)" }}>Loading authority types...</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>CODE</th>
                <th>NAME</th>
                <th>DESCRIPTION</th>
                <th>ACTIONS</th>
              </tr>
            </thead>
            <tbody>
              {types.length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
                    No authority types defined yet.
                  </td>
                </tr>
              ) : (
                types.map((t) => (
                  <tr key={t.id}>
                    <td style={{ padding: '0.85rem 1rem' }}>
                      {t.code}
                    </td>
                    <td style={{ padding: '0.85rem 1rem' }}>{t.name}</td>
                    <td style={{ padding: '0.85rem 1rem' }}>{t.description || "-"}</td>
                    <td style={{ padding: '0.85rem 1rem' }}>
                      <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                        <button
                          class="btn-action"
                          onClick={() => startEdit(t)}
                          disabled={saving}
                        >
                          Edit
                        </button>
                        <button
                          class="btn-action"
                          onClick={() => handleDelete(t.id)}
                          disabled={saving}
                          style={{ borderColor: "var(--danger)", color: "var(--danger)" }}
                          onMouseEnter={(e) => {
                            e.target.style.backgroundColor = "var(--danger)";
                            e.target.style.color = "white";
                          }}
                          onMouseLeave={(e) => {
                            e.target.style.backgroundColor = "transparent";
                            e.target.style.color = "var(--danger)";
                          }}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        )}
      </div>
    </Fragment>
  );
}
