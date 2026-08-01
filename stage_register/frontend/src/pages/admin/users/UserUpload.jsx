import React, { useState } from "react";
import { UsersApi } from "../../../helpers/api";

export default function UserUpload() {
  const [uploading, setUploading] = useState(false);
  const [summary, setSummary] = useState(null);
  const [error, setError] = useState("");

  async function handleFileChange(e) {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setSummary(null);
    setError("");

    const formData = new FormData();
    formData.append("file", file);

    try {
      const result = await UsersApi.uploadCsv(formData);
      setSummary(result);
    } catch (ex) {
      setError(
        ex?.response?.data?.error ||
          ex?.message ||
          "Failed to upload CSV. Please check the format."
      );
    } finally {
      setUploading(false);
      // reset file input so the same file can be selected again
      e.target.value = "";
    }
  }

  return (
    <div className="card shadow-sm">
      <div className="card-header">
        <div className="fw-semibold">Bulk User Upload</div>
        <div className="text-muted small">
          Upload a CSV file to create multiple users at once. All users will be
          created with the default password{" "}
          <code>biostat@2026</code>.
        </div>
      </div>

      <div className="card-body">
        {error && (
          <div className="alert alert-danger py-1 small mb-3">{error}</div>
        )}

        {summary && (
          <div className="alert alert-info py-1 small mb-3">
            Processed <strong>{summary.total}</strong> rows.{" "}
            <strong>{summary.created}</strong> users created,{" "}
            <strong>{summary.failed}</strong> failed.
          </div>
        )}

        <div className="mb-3 border rounded p-3 bg-light">
          <div className="d-flex justify-content-between align-items-center mb-2">
            <div className="small fw-semibold">Upload CSV</div>
            <span className="badge bg-secondary">
              email, username, [first_name], [last_name], [role], [organisation], [phoneno], [district_id]
            </span>
          </div>
          <div className="small text-muted mb-2">
            The CSV file must include a header row with at least{" "}
            <code>email</code> and <code>username</code> columns. Optional
            columns are <code>first_name</code>, <code>last_name</code>,{" "}
            <code>role</code>, <code>organisation</code>,{" "}
            <code>phoneno</code>, and <code>district_id</code>. Passwords
            are not read from the file; all accounts will use the default
            password.
          </div>
          <input
            type="file"
            accept=".csv,text/csv"
            className="form-control form-control-sm"
            onChange={handleFileChange}
            disabled={uploading}
          />
          {uploading && (
            <div className="small text-muted mt-2">Uploading and processing...</div>
          )}
        </div>

        {summary && Array.isArray(summary.errors) && summary.errors.length > 0 && (
          <div className="mt-3">
            <div className="fw-semibold small mb-1">Row errors</div>
            <div className="table-responsive">
              <table className="table table-sm table-bordered mb-0">
                <thead className="table-light">
                  <tr>
                    <th style={{ width: "4rem" }}>Row</th>
                    <th>Email</th>
                    <th>Username</th>
                    <th>Error</th>
                  </tr>
                </thead>
                <tbody>
                  {summary.errors.map((row) => (
                    <tr key={row.row}>
                      <td>{row.row}</td>
                      <td>{row.email || "-"}</td>
                      <td>{row.username || "-"}</td>
                      <td>{row.error}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

