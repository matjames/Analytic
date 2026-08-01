import React, { Fragment, useEffect, useState } from "react";
import { FacilityLevelsApi } from "../../../helpers/api";

export default function FacilityLevelManager() {
  const [levels, setLevels] = useState([]);
  const [form, setForm] = useState({ code: "", name: "", description: "" });
  const [editingId, setEditingId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const rows = await FacilityLevelsApi.list();
      setLevels(rows);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load station tiers");
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
        await FacilityLevelsApi.update(editingId, form);
      } else {
        await FacilityLevelsApi.create(form);
      }
      resetForm();
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save station tier");
    } finally {
      setSaving(false);
    }
  }

  function startEdit(level) {
    setForm({
      code: level.code || "",
      name: level.name || "",
      description: level.description || "",
    });
    setEditingId(level.id);
  }

  async function handleDelete(id) {
    if (!window.confirm("Delete this station tier? This may fail if field stations are using it.")) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await FacilityLevelsApi.remove(id);
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete station tier");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Fragment>
      <div class="page-header">
        <div class="page-title">
          <h2>Station Tiers</h2>
          <div class="page-subtitle">Manage field-operation tiers (e.g., Community Enumeration Point, District Coordination Hub).</div>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger py-1 small mb-3">{error}</div>
      )}

      <div class="filters-section">
        <form onSubmit={handleSubmit} style={{ display: "flex", gap: "0.8rem", flexWrap: "wrap", alignItems: "center", width: "100%" }}>
          <div class="filter-group">
            <label>Code:</label>
            <input
              type="text"
              placeholder="e.g. DISTRICT_HUB"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
              disabled={saving}
              required
            />
          </div>
          <div class="filter-group">
            <label>Name:</label>
            <input
              type="text"
              placeholder="e.g. District Coordination Hub"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              disabled={saving}
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
              disabled={saving}
            />
          </div>
          <button type="submit" class="btn-action" disabled={saving || !form.code.trim() || !form.name.trim()}>
            {editingId ? "Update" : "Add"}
          </button>
          {editingId && (
            <button
              type="button"
              class="btn-clear"
              onClick={resetForm}
              disabled={saving}
            >
              Cancel
            </button>
          )}
        </form>
      </div>

      <div className="table-section">
        <div className="table-header">
          <h3>
            <i className="bi bi-layers"></i> Station Tiers ({levels.length} {levels.length === 1 ? 'tier' : 'tiers'})
          </h3>
        </div>
        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "var(--text-muted)" }}>Loading...</div>
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
              {levels.length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
                    No station tiers found
                  </td>
                </tr>
              ) : (
                levels.map((level) => (
                  <tr key={level.id}>
                    <td style={{ padding: '0.85rem 1rem' }}>
                      {level.code || "-"}
                    </td>
                    <td style={{ padding: '0.85rem 1rem' }}>{level.name}</td>
                    <td style={{ padding: '0.85rem 1rem' }}>{level.description || "-"}</td>
                    <td style={{ padding: '0.85rem 1rem' }}>
                      <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                        <button
                          class="btn-action"
                          onClick={() => startEdit(level)}
                          disabled={saving}
                        >
                          Edit
                        </button>
                        <button
                          class="btn-action"
                          onClick={() => handleDelete(level.id)}
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
