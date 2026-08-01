import React, { useEffect, useMemo, useState } from "react";
import { LevelsApi, UnitsApi } from "../../../helpers/api";

export default function UnitsManager() {
  const [levels, setLevels] = useState([]);
  const [unitsByLevel, setUnitsByLevel] = useState({});
  const [form, setForm] = useState({
    name: "",
    levelId: "",
    parentId: "",
  });
  const [list, setList] = useState([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [uploading, setUploading] = useState(false);
  const [uploadSummary, setUploadSummary] = useState(null);
  const [filterLevelId, setFilterLevelId] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [total, setTotal] = useState(0);

  async function loadLevels() {
    try {
      const l = await LevelsApi.list();
      setLevels(l);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load levels");
    }
  }

  async function loadUnitsForLevel(levelId, parentId) {
    return UnitsApi.list({
      ...(levelId && { levelId }),
      ...(parentId && { parentId }),
    });
  }

  async function loadPage(nextPage = page, nextFilterLevelId = filterLevelId) {
    setLoading(true);
    setError("");
    try {
      const res = await UnitsApi.listPaged({
        levelId: nextFilterLevelId || undefined,
        page: nextPage,
        pageSize,
      });
      setList(res.rows || []);
      setTotal(res.total || 0);
      setPage(res.page || nextPage);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load units");
    } finally {
      setLoading(false);
    }
  }

  async function handleUploadFile(e) {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setUploadSummary(null);
    setError("");

    const formData = new FormData();
    formData.append("file", file);

    try {
      const result = await UnitsApi.uploadCsv(formData);
      setUploadSummary(result);
      await loadPage(page, filterLevelId);
    } catch (ex) {
      setError(
        ex?.response?.data?.error ||
          ex?.message ||
          "Failed to upload CSV. Please check the format."
      );
    } finally {
      setUploading(false);
      // reset file input so same file can be selected again
      e.target.value = "";
    }
  }

  useEffect(() => {
    loadLevels();
  }, []);

  useEffect(() => {
    async function bootstrap() {
      if (levels.length === 0) return;
      const map = {};
      for (const lvl of levels) {
        map[lvl.id] = await loadUnitsForLevel(lvl.id);
      }
      setUnitsByLevel(map);
      await loadPage(1, filterLevelId);
    }
    bootstrap();
  }, [levels]);

  const parentLevelId = useMemo(() => {
    const lvl = levels.find((l) => l.id === Number(form.levelId));
    if (!lvl) return undefined;
    const parentLevel = levels.find(
      (x) => x.level_number === lvl.level_number - 1
    );
    return parentLevel?.id;
  }, [form.levelId, levels]);

  const parentOptions = useMemo(() => {
    if (!parentLevelId) return [];
    return (unitsByLevel[parentLevelId] || []).slice().sort((a, b) => {
      return a.name.localeCompare(b.name);
    });
  }, [parentLevelId, unitsByLevel]);

  async function submit(e) {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      await UnitsApi.create({
        name: form.name.trim(),
        code: undefined,
        levelId: Number(form.levelId),
        parentId: form.parentId ? Number(form.parentId) : null,
      });
      setForm({ name: "", levelId: "", parentId: "" });
      await loadPage(1, filterLevelId);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to create unit");
    } finally {
      setSaving(false);
    }
  }

  async function remove(id) {
    if (!window.confirm("Delete this unit? Children must be moved or deleted first, or use cascade.")) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await UnitsApi.remove(id);
      await loadPage(page, filterLevelId);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete unit");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="card shadow-sm">
      <div className="card-header">
        <div className="fw-semibold">Administrative Units</div>
        <div className="text-muted small">
          Create and manage units within each level; parent is required from the
          level above.
        </div>
      </div>

      <div className="card-body">
        {error && (
          <div className="alert alert-danger py-1 small mb-3">{error}</div>
        )}
        {uploadSummary && (
          <div className="alert alert-info py-1 small mb-3">
            Uploaded {uploadSummary.total} rows:{" "}
            <strong>{uploadSummary.success}</strong> succeeded,{" "}
            <strong>{uploadSummary.failed}</strong> failed.
          </div>
        )}

        <div className="mb-3 border rounded p-2 bg-light">
          <div className="d-flex justify-content-between align-items-center mb-1">
            <div className="small fw-semibold">Bulk upload (CSV)</div>
            <span className="badge bg-secondary">name, level_id, parent_mfl_uid</span>
          </div>
          <div className="small text-muted mb-1">
            Upload a CSV file with a header row containing at least{" "}
            <code>name</code> and <code>level_id</code>. Optionally include{" "}
            <code>parent_mfl_uid</code>. MFL UID is auto-generated.
          </div>
          <input
            type="file"
            accept=".csv,text/csv"
            className="form-control form-control-sm"
            onChange={handleUploadFile}
            disabled={uploading || saving}
          />
        </div>

        <form className="row g-2 mb-3" onSubmit={submit}>
          <div className="col-md-5">
            <label className="form-label form-label-sm">Name</label>
            <input
              className="form-control form-control-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="col-md-3">
            <label className="form-label form-label-sm">Level</label>
            <select
              className="form-select form-select-sm"
              value={form.levelId}
              onChange={(e) =>
                setForm({
                  ...form,
                  levelId: e.target.value,
                  parentId: "",
                })
              }
              required
            >
              <option value="">Select level…</option>
              {levels.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </div>
          <div className="col-md-3">
            <label className="form-label form-label-sm">
              Parent{" "}
              <span className="text-muted">
                ({parentLevelId ? "required" : "N/A for top level"})
              </span>
            </label>
            <select
              className="form-select form-select-sm"
              value={form.parentId}
              onChange={(e) =>
                setForm({ ...form, parentId: e.target.value || "" })
              }
              disabled={!parentLevelId}
              required={!!parentLevelId}
            >
              <option value="">
                {parentLevelId ? "Select parent…" : "—"}
              </option>
              {parentOptions.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
          <div className="col-12 text-end">
            <button
              type="submit"
              className="btn btn-primary btn-sm px-4"
              disabled={saving || uploading}
            >
              {saving ? "Saving..." : "Add unit"}
            </button>
          </div>
        </form>

        <div className="d-flex justify-content-between align-items-center mb-2 small">
          <div>
            <label className="me-2">Filter by level:</label>
            <select
              className="form-select form-select-sm d-inline-block"
              style={{ width: "220px" }}
              value={filterLevelId}
              onChange={(e) => {
                const val = e.target.value;
                setFilterLevelId(val);
                setPage(1);
                loadPage(1, val);
              }}
            >
              <option value="">All levels</option>
              {levels.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            {total > 0 && (
              <span className="text-muted">
                Page {page} of {Math.max(1, Math.ceil(total / pageSize))} ({total} total)
              </span>
            )}
          </div>
        </div>

        <div className="table-responsive">
          <table className="table table-sm align-middle mb-0">
            <thead className="table-light">
              <tr>
                <th>Name</th>
                <th>MFL UID</th>
                <th>Level</th>
                <th>Parent UID</th>
                <th style={{ width: "6rem" }} className="text-end">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={5} className="text-muted small">
                    Loading units...
                  </td>
                </tr>
              ) : list.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-muted small">
                    No units yet.
                  </td>
                </tr>
              ) : (
                list.map((u) => (
                  <tr key={u.id}>
                    <td>{u.name}</td>
                    <td>
                      {u.mfl_uid ? (
                        <code>{u.mfl_uid}</code>
                      ) : (
                        <span className="text-muted">—</span>
                      )}
                    </td>
                    <td>{u.admin_level?.name}</td>
                    <td>
                      {u.parent_id ? (
                        <code>{u.parent_id}</code>
                      ) : (
                        <span className="text-muted">—</span>
                      )}
                    </td>
                    <td className="text-end">
                      <button
                        type="button"
                        className="btn btn-outline-danger btn-sm"
                        onClick={() => remove(u.id)}
                        disabled={saving}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="d-flex justify-content-between align-items-center mt-2 small">
          <div>
            Showing{" "}
            {total === 0 ? 0 : (page - 1) * pageSize + 1} -{" "}
            {Math.min(total, page * pageSize)} of {total}
          </div>
          <div className="btn-group btn-group-sm">
            <button
              type="button"
              className="btn btn-outline-secondary"
              disabled={page <= 1 || loading}
              onClick={() => loadPage(page - 1, filterLevelId)}
            >
              Previous
            </button>
            <button
              type="button"
              className="btn btn-outline-secondary"
              disabled={page * pageSize >= total || loading}
              onClick={() => loadPage(page + 1, filterLevelId)}
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}