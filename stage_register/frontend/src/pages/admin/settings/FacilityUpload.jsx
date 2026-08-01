import React, { useState } from "react";
import * as XLSX from "xlsx";
import { FacilitiesApi } from "../../../helpers/api";

const NAME_HEADERS = [
  "organisation unit name",
  "organisation Unit name",
  "name",
  "facility name",
];
const ID_HEADERS = [
  "organisation unit id",
  "organisation Unit id",
  "mfluid",
  "mfl_uid",
  "organisation_unit_id",
];

function findColumnIndex(headerRow, aliases) {
  const normalized = headerRow.map((h) => String(h || "").toLowerCase().trim());
  for (const alias of aliases) {
    const i = normalized.indexOf(alias.toLowerCase());
    if (i !== -1) return i;
  }
  return -1;
}

function parseFile(file) {
  return new Promise((resolve, reject) => {
    const isCsv = file.name.toLowerCase().endsWith(".csv");

    function processWorkbook(wb) {
      const firstSheet = wb.SheetNames[0];
      if (!firstSheet) {
        reject(new Error("No sheet found in file"));
        return;
      }
      const ws = wb.Sheets[firstSheet];
      const rows = XLSX.utils.sheet_to_json(ws, { header: 1, defval: "" });
      if (!rows.length) {
        reject(new Error("File is empty"));
        return;
      }
      const headerRow = rows[0].map((c) => String(c ?? "").trim());
      const nameIdx = findColumnIndex(headerRow, NAME_HEADERS);
      if (nameIdx === -1) {
        reject(
          new Error(
            "Required column not found. Expected one of: organisation Unit name, name"
          )
        );
        return;
      }
      const idIdx = findColumnIndex(headerRow, ID_HEADERS);
      const facilities = [];
      for (let i = 1; i < rows.length; i++) {
        const row = rows[i] || [];
        const name = String(row[nameIdx] ?? "").trim();
        if (!name) continue;
        const organisation_unit_id =
          idIdx >= 0 ? String(row[idIdx] ?? "").trim() : "";
        facilities.push({ name, organisation_unit_id });
      }
      resolve(facilities);
    }

    if (isCsv) {
      const textReader = new FileReader();
      textReader.onload = (ev) => {
        try {
          const wb = XLSX.read(ev.target.result, { type: "string", raw: true });
          processWorkbook(wb);
        } catch (err) {
          reject(err);
        }
      };
      textReader.onerror = () => reject(new Error("Failed to read file"));
      textReader.readAsBinaryString(file);
    } else {
      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const data = new Uint8Array(e.target.result);
          const wb = XLSX.read(data, { type: "array" });
          processWorkbook(wb);
        } catch (err) {
          reject(err);
        }
      };
      reader.onerror = () => reject(new Error("Failed to read file"));
      reader.readAsArrayBuffer(file);
    }
  });
}

function downloadCreatedRows(rows) {
  if (!rows?.length) return;
  const ws = XLSX.utils.json_to_sheet(
    rows.map((r) => ({
      identifier: r.identifier ?? "",
      mfl_uid: r.mfl_uid ?? "",
      name: r.name ?? "",
    }))
  );
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, "Uploaded facilities");
  XLSX.writeFile(
    wb,
    `facilities-uploaded-${new Date().toISOString().slice(0, 10)}.xlsx`
  );
}

export default function FacilityUpload() {
  const [uploading, setUploading] = useState(false);
  const [summary, setSummary] = useState(null);
  const [error, setError] = useState("");

  async function handleFileChange(e) {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setSummary(null);
    setError("");

    try {
      const facilities = await parseFile(file);
      if (!facilities.length) {
        setError("No valid rows found. File must have a header and at least one row with 'organisation Unit name' (or 'name').");
        setUploading(false);
        e.target.value = "";
        return;
      }

      const result = await FacilitiesApi.upload(facilities);
      setSummary(result);
    } catch (err) {
      setError(
        err?.response?.data?.error ||
          err?.message ||
          "Failed to upload. Check file format."
      );
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  }

  return (
    <>
      <div className="page-header">
        <div className="page-title">
          <h2>Upload Field Stations</h2>
          <div className="page-subtitle">
            Upload an Excel or CSV file to create field stations. Each row is assigned a unique StatGate registry UID; an unused supplied ID is retained, otherwise one is generated.
          </div>
        </div>
      </div>

      <div className="card shadow-sm">
        <div className="card-header">
          <div className="fw-semibold">Upload file</div>
          <div className="text-muted small">
            Required column: <strong>organisation Unit name</strong> (or <code>name</code>). Optional: <strong>organisation Unit id</strong> (or <code>mfl_uid</code>).
          </div>
        </div>

      <div className="card-body">
        {error && (
          <div className="alert alert-danger py-2 small mb-3">{error}</div>
        )}

        {summary && (
          <div className="alert alert-info py-2 small mb-3">
            Processed <strong>{summary.total}</strong> row(s).{" "}
            <strong>{summary.created}</strong> field station(s) created,{" "}
            <strong>{summary.failed}</strong> failed.
          </div>
        )}

        {summary?.created_rows?.length > 0 && (
          <div className="mt-3">
            <div className="d-flex justify-content-between align-items-center mb-2">
              <span className="fw-semibold small">Uploaded field stations</span>
              <button
                type="button"
                className="btn btn-sm btn-outline-primary"
                onClick={() => downloadCreatedRows(summary.created_rows)}
              >
                <i className="bi bi-download me-1" />
                Download
              </button>
            </div>
            <div className="table-responsive">
              <table className="table table-sm table-bordered mb-0">
                <thead className="table-light">
                  <tr>
                    <th>identifier</th>
                    <th>mfl_uid</th>
                    <th>name</th>
                  </tr>
                </thead>
                <tbody>
                  {summary.created_rows.map((row, idx) => (
                    <tr key={idx}>
                      <td>{row.identifier ?? "—"}</td>
                      <td>{row.mfl_uid ?? "—"}</td>
                      <td>{row.name ?? "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        <div className="mb-3 border rounded p-3 bg-light">
          <div className="d-flex justify-content-between align-items-center mb-2">
            <div className="small fw-semibold">Upload Excel or CSV</div>
            <span className="badge bg-secondary">
              organisation Unit name, organisation Unit id
            </span>
          </div>
          <div className="small text-muted mb-2">
            The file must have a header row. Supported column names:{" "}
            <code>organisation Unit name</code> (or <code>name</code>),{" "}
            <code>organisation Unit id</code> (or <code>mfl_uid</code>).
            Organisation Unit id is optional; when provided and not already
            assigned, it will be used as the field station registry UID.
          </div>
          <input
            type="file"
            accept=".xlsx,.xls,.csv"
            className="form-control form-control-sm"
            onChange={handleFileChange}
            disabled={uploading}
          />
          {uploading && (
            <div className="small text-muted mt-2">
              Uploading and processing…
            </div>
          )}
        </div>

        {summary &&
          Array.isArray(summary.errors) &&
          summary.errors.length > 0 && (
            <div className="mt-3">
              <div className="fw-semibold small mb-1">Row errors</div>
              <div className="table-responsive">
                <table className="table table-sm table-bordered mb-0">
                  <thead className="table-light">
                    <tr>
                      <th style={{ width: "4rem" }}>Row</th>
                      <th>Name</th>
                      <th>Error</th>
                    </tr>
                  </thead>
                  <tbody>
                    {summary.errors.map((row, idx) => (
                      <tr key={idx}>
                        <td>{row.row}</td>
                        <td>{row.name || "—"}</td>
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
    </>
  );
}
