import React, { Fragment, useEffect, useState } from "react";
import { LevelsApi } from "../../../helpers/api";

export default function LevelsManager() {
  const [levels, setLevels] = useState([]);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [editingId, setEditingId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const rows = await LevelsApi.list();
      setLevels(rows);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load levels");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  function resetForm() {
    setName("");
    setCode("");
    setEditingId(null);
  }

  async function add(e) {
    e.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    setError("");
    try {
      if (editingId) {
        await LevelsApi.update(editingId, { name: name.trim(), code: code.trim() || undefined });
      } else {
        await LevelsApi.create({ name: name.trim(), code: code.trim() || undefined });
      }
      resetForm();
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save level");
    } finally {
      setSaving(false);
    }
  }

  function startEdit(level) {
    setName(level.name || "");
    setCode(level.code || "");
    setEditingId(level.id);
  }

  async function moveLevel(index, direction) {
    const newIndex = index + direction;
    if (newIndex < 0 || newIndex >= levels.length) return;
    const reordered = [...levels];
    const [moved] = reordered.splice(index, 1);
    reordered.splice(newIndex, 0, moved);
    try {
      // Optimistic UI: update order locally
      setLevels(reordered);
      // Persist + refresh from server so level_number stays correct
      const updated = await LevelsApi.reorder(reordered.map((l) => l.id));
      if (Array.isArray(updated) && updated.length > 0) {
        setLevels(updated);
      }
    } catch (e) {
      // revert on error
      setError(e?.response?.data?.error || e.message || "Failed to reorder levels");
      load();
    }
  }

  async function removeLevel(id) {
    if (!window.confirm("Delete this level? This is only allowed if no units use it.")) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await LevelsApi.remove(id);
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete level");
    } finally {
      setSaving(false);
    }
  }

  async function seedBaseLevels() {
    if (
      !window.confirm(
        "Seed default levels (National, Region, District, Subcounty, Field Survey Station)? Existing levels will remain."
      )
    ) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await LevelsApi.seedDefaults();
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to seed base levels");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Fragment>
      <div class="page-header">
        <div class="page-title">
          <h2>Hierarchy Levels</h2>
          <div class="page-subtitle">Define and order the administrative levels (e.g. Region, District, Field Survey Station).</div>
        </div>
        <button class="btn-action" onClick={seedBaseLevels} disabled={saving}>
          Seed default levels
        </button>
      </div>

      {error && <div className="alert alert-danger py-1 small mb-3">{error}</div>}

      <div class="filters-section">
        <form onSubmit={add} style={{ display: "flex", gap: "0.8rem", flexWrap: "wrap", alignItems: "center", width: "100%" }}>
          <div class="filter-group">
            <label>Name:</label>
            <input
              type="text"
              placeholder="e.g. District"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={saving}
            />
          </div>
          <div class="filter-group">
            <label>Code:</label>
            <input
              type="text"
              placeholder="e.g. DIST (optional)"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              disabled={saving}
            />
          </div>
          <button type="submit" class="btn-action" disabled={saving || !name.trim()}>
            {editingId ? "Update" : "Add Level"}
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
            <i className="bi bi-list-ol"></i> Hierarchy Levels ({levels.length} {levels.length === 1 ? 'level' : 'levels'})
          </h3>
        </div>
        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "var(--text-muted)" }}>Loading levels...</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>ORDER</th>
                <th>NAME</th>
                <th>LEVEL NO</th>
                <th>ACTIONS</th>
              </tr>
            </thead>
            <tbody>
              {levels.length === 0 ? (
                <tr>
                  <td colSpan={4} style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
                    No levels defined yet.
                  </td>
                </tr>
              ) : (
                levels.map((level, index) => (
                  <tr key={level.id}>
                    <td style={{ padding: '0.85rem 1rem' }}>
                      <div class="order-controls">
                        <button
                          class="order-btn"
                          onClick={() => moveLevel(index, -1)}
                          disabled={index === 0 || saving}
                        >
                          <i class="bi bi-caret-up-fill"></i>
                        </button>
                        <button
                          class="order-btn"
                          onClick={() => moveLevel(index, 1)}
                          disabled={index === levels.length - 1 || saving}
                        >
                          <i class="bi bi-caret-down-fill"></i>
                        </button>
                      </div>
                    </td>
                    <td style={{ padding: '0.85rem 1rem' }}>{level.name}</td>
                    <td style={{ padding: '0.85rem 1rem', textAlign: 'center' }}>{level.level_number || "-"}</td>
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
                          onClick={() => removeLevel(level.id)}
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
