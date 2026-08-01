import React, { Fragment, useEffect, useState } from "react";
import { DocumentsApi } from "../../../helpers/api";

export default function DocumentManager() {
  const [documents, setDocuments] = useState([]);
  const [form, setForm] = useState({ title: "", description: "", category: "SOP", document: null });
  const [editingId, setEditingId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showModal, setShowModal] = useState(false);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const rows = await DocumentsApi.list();
      setDocuments(Array.isArray(rows) ? rows : []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load documents");
      setDocuments([]); // Set empty array on error to prevent crashes
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  function resetForm() {
    setForm({ title: "", description: "", category: "SOP", document: null });
    setEditingId(null);
    setError("");
    setSuccess("");
    setShowModal(false);
    // Reset file input
    const fileInput = document.getElementById("document-file");
    if (fileInput) fileInput.value = "";
  }

  function openModal() {
    setShowModal(true);
    setError("");
    setSuccess("");
  }

  function closeModal() {
    resetForm();
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.title.trim() || !form.category) {
      setError("Title and category are required");
      return;
    }
    if (!editingId && !form.document) {
      setError("Please select a PDF or image file to upload");
      return;
    }

    setSaving(true);
    setError("");
    setSuccess("");

    try {
      const formData = new FormData();
      formData.append("title", form.title);
      formData.append("description", form.description || "");
      formData.append("category", form.category);
      if (form.document) {
        formData.append("document", form.document);
      }

      if (editingId) {
        await DocumentsApi.update(editingId, {
          title: form.title,
          description: form.description || "",
          category: form.category,
        });
        setSuccess("Document updated successfully");
      } else {
        await DocumentsApi.upload(formData);
        setSuccess("Document uploaded successfully");
      }
      await load();
      // Close modal after a short delay to show success message
      setTimeout(() => {
        resetForm();
      }, 1000);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save document");
    } finally {
      setSaving(false);
    }
  }

  function startEdit(doc) {
    setForm({
      title: doc.title || "",
      description: doc.description || "",
      category: doc.category || "SOP",
      document: null,
    });
    setEditingId(doc.id);
    setError("");
    setSuccess("");
    setShowModal(true);
  }

  async function handleDelete(id) {
    if (!window.confirm("Delete this document? This action cannot be undone.")) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      await DocumentsApi.delete(id);
      setSuccess("Document deleted successfully");
      await load();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to delete document");
    } finally {
      setSaving(false);
    }
  }

  async function handleDownload(id, originalFilename) {
    try {
      const blob = await DocumentsApi.download(id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = originalFilename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to download document");
    }
  }

  function formatFileSize(bytes) {
    if (!bytes) return "-";
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + " KB";
    return (bytes / (1024 * 1024)).toFixed(2) + " MB";
  }

  function formatDate(dateString) {
    if (!dateString) return "-";
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
  }

  function getCategoryStyle(category) {
    switch (category) {
      case "SOP":
        return { background: "#dbeafe", color: "#1e40af" };
      case "Manual":
        return { background: "#dcfce7", color: "#15803d" };
      case "Training":
        return { background: "#fef3c7", color: "#92400e" };
      default:
        return { background: "#f3f4f6", color: "#374151" };
    }
  }

  return (
    <Fragment>
      <div className="page-header">
        <div className="page-title">
          <h2>Public Documents</h2>
          <div className="page-subtitle">
            Upload and manage PDF documents that will be displayed on the public portal under SOPs & Manuals.
          </div>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger py-1 small mb-3">{error}</div>
      )}

      {success && !showModal && (
        <div className="alert alert-success py-1 small mb-3">{success}</div>
      )}

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
        <div></div>
        <button
          type="button"
          className="btn-action"
          onClick={openModal}
          style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}
        >
          <i className="bi bi-plus-circle" /> Upload Document
        </button>
      </div>

      <div className="table-container">
        {loading ? (
          <div style={{ padding: "2rem", textAlign: "center", color: "var(--text-muted)" }}>Loading...</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th style={{ width: "35%", maxWidth: "350px" }}>Document</th>
                <th style={{ width: "12%" }}>Category</th>
                <th style={{ width: "53%", minWidth: "350px", textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {documents.length === 0 ? (
                <tr>
                  <td colSpan="3" style={{ textAlign: "center", padding: "2rem", color: "var(--text-muted)" }}>
                    No documents found. Upload your first document above.
                  </td>
                </tr>
              ) : (
                documents.map((doc) => (
                  <tr key={doc.id}>
                    <td style={{ maxWidth: "350px", wordWrap: "break-word" }}>
                      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                        <i className="bi bi-file-pdf-fill" style={{ color: "#dc2626", fontSize: "1.2rem", flexShrink: 0 }} />
                        <div style={{ minWidth: 0, flex: 1 }}>
                          <div style={{ fontWeight: "500", wordBreak: "break-word" }}>{doc.title}</div>
                          {doc.description && (
                            <div style={{ fontSize: "0.875rem", color: "var(--text-muted)", wordBreak: "break-word", marginTop: "0.25rem" }}>
                              {doc.description}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span
                        className="badge-category"
                        style={getCategoryStyle(doc.category)}
                      >
                        {doc.category}
                      </span>
                    </td>
                    <td style={{ width: "53%", minWidth: "350px", textAlign: "right" }}>
                      <div style={{ display: "flex", gap: "0.5rem", flexWrap: "nowrap", justifyContent: "flex-end" }}>
                        <button
                          className="btn-action"
                          onClick={() => handleDownload(doc.id, doc.original_filename)}
                          style={{ fontSize: "0.875rem", padding: "0.4rem 0.75rem", whiteSpace: "nowrap" }}
                          title="Download"
                        >
                          <i className="bi bi-download" /> Download
                        </button>
                        <button
                          className="btn-action"
                          onClick={() => startEdit(doc)}
                          disabled={saving}
                          style={{ fontSize: "0.875rem", padding: "0.4rem 0.75rem", whiteSpace: "nowrap" }}
                          title="Edit"
                        >
                          <i className="bi bi-pencil" /> Edit
                        </button>
                        <button
                          className="btn-action"
                          onClick={() => handleDelete(doc.id)}
                          disabled={saving}
                          style={{
                            fontSize: "0.875rem",
                            padding: "0.4rem 0.75rem",
                            borderColor: "var(--danger)",
                            color: "var(--danger)",
                            whiteSpace: "nowrap",
                          }}
                          onMouseEnter={(e) => {
                            e.target.style.backgroundColor = "var(--danger)";
                            e.target.style.color = "white";
                          }}
                          onMouseLeave={(e) => {
                            e.target.style.backgroundColor = "transparent";
                            e.target.style.color = "var(--danger)";
                          }}
                          title="Delete"
                        >
                          <i className="bi bi-trash" /> Delete
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

      {/* Document Upload/Edit Modal */}
      {showModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)", zIndex: 1050 }}
          onClick={closeModal}
        >
          <div
            className="modal-dialog modal-lg modal-dialog-scrollable"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">
                  {editingId ? "Edit Document" : "Upload Document"}
                </h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={closeModal}
                  aria-label="Close"
                />
              </div>
              <form onSubmit={handleSubmit}>
                <div className="modal-body">
                  {error && (
                    <div className="alert alert-danger py-2 small mb-3" role="alert">
                      {error}
                    </div>
                  )}

                  {success && (
                    <div className="alert alert-success py-2 small mb-3" role="alert">
                      {success}
                    </div>
                  )}

                  <div style={{ display: "flex", flexDirection: "column", gap: "1.5rem" }}>
                    <div style={{ display: "flex", gap: "1rem", flexWrap: "wrap" }}>
                      <div className="form-group" style={{ flex: "1 1 300px", minWidth: "250px" }}>
                        <label className="form-label">
                          Title: <span className="required">*</span>
                        </label>
                        <input
                          type="text"
                          className="form-control"
                          placeholder="e.g. Field Station Registration SOP"
                          value={form.title}
                          onChange={(e) => setForm({ ...form, title: e.target.value })}
                          disabled={saving}
                          required
                        />
                      </div>
                      <div className="form-group" style={{ flex: "1 1 200px", minWidth: "200px" }}>
                        <label className="form-label">
                          Category: <span className="required">*</span>
                        </label>
                        <select
                          className="form-control form-select"
                          value={form.category}
                          onChange={(e) => setForm({ ...form, category: e.target.value })}
                          disabled={saving}
                          required
                        >
                          <option value="SOP">SOP</option>
                          <option value="Manual">Manual</option>
                          <option value="Training">Training</option>
                          <option value="Other">Other</option>
                        </select>
                      </div>
                    </div>
                    <div className="form-group" style={{ width: "100%" }}>
                      <label className="form-label">Description:</label>
                      <textarea
                        className="form-control"
                        placeholder="Optional description of the document"
                        value={form.description}
                        onChange={(e) => setForm({ ...form, description: e.target.value })}
                        disabled={saving}
                        rows="3"
                      />
                    </div>
                    {!editingId && (
                      <div className="form-group" style={{ width: "100%" }}>
                        <label className="form-label">
                          File (PDF or image): <span className="required">*</span>
                        </label>
                        <input
                          id="document-file"
                          type="file"
                          className="form-control"
                          accept=".pdf,.jpg,.jpeg,.png,.gif,.webp,application/pdf,image/jpeg,image/png,image/gif,image/webp"
                          onChange={(e) => {
                            const file = e.target.files?.[0];
                            if (!file) {
                              setForm((f) => ({ ...f, document: null }));
                              return;
                            }
                            if (!/\.(pdf|jpe?g|png|gif|webp)$/i.test(file.name)) {
                              setError("Only PDF and image files (JPG, PNG, GIF, WebP) are allowed.");
                              e.target.value = "";
                              return;
                            }
                            setError("");
                            setForm((f) => ({ ...f, document: file }));
                          }}
                          disabled={saving}
                          required={!editingId}
                          style={{ padding: "0.4rem" }}
                        />
                        <small className="form-text">
                          PDF and images (JPG, PNG, GIF, WebP) only. Maximum file size: 50MB
                        </small>
                      </div>
                    )}
                  </div>
                </div>
                <div className="modal-footer">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={closeModal}
                    disabled={saving}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={saving || !form.title.trim() || !form.category || (!editingId && !form.document)}
                  >
                    {saving
                      ? "Saving..."
                      : editingId
                      ? "Update Document"
                      : "Upload Document"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </Fragment>
  );
}
