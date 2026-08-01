import React, { useEffect, useState } from "react";
import Footer from "./Footer";
import Header from "./Header";
import { DocumentsApi } from "../../helpers/api";

const SopsManuals = () => {
  const [documents, setDocuments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [guide, setGuide] = useState(null);
  const [preview, setPreview] = useState(null);

  useEffect(() => {
    async function loadDocuments() {
      try {
        setLoading(true);
        setError("");
        const docs = await DocumentsApi.list();
        setDocuments(docs || []);
      } catch (e) {
        setError(e?.response?.data?.error || e.message || "Failed to load documents");
      } finally {
        setLoading(false);
      }
    }
    loadDocuments();
  }, []);

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

  function formatDate(dateString) {
    if (!dateString) return "-";
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", { year: "numeric", month: "short" });
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
      alert("Failed to download document: " + (e?.response?.data?.error || e.message));
    }
  }

  async function openGuide(title, path) {
    try {
      const response = await fetch(path);
      if (!response.ok) throw new Error("Guide unavailable");
      setGuide({ title, content: await response.text() });
    } catch (e) {
      setError(e.message || "Failed to open guide");
    }
  }

  async function handlePreview(id, title) {
    try {
      const blob = await DocumentsApi.download(id);
      const url = window.URL.createObjectURL(blob);
      setPreview({ title, url });
    } catch (e) {
      alert("Failed to preview document: " + (e?.response?.data?.error || e.message));
    }
  }

  return (
    <>
      <Header />

      <div className="container">
        <div className="page-title-section">
          <h2>Documentation & Resources</h2>
          <p>
            Access SOPs, user manuals, and training materials for the StatGate
            Field Operations & Agent Workforce Registry.
          </p>
        </div>

        <div className="documents-section" style={{ marginBottom: "1.5rem" }}>
          <div className="table-header">
            <h3>StatGate starter resources</h3>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "1rem", padding: "1rem" }}>
            <button type="button" onClick={() => openGuide("Field Operations SOP", "/docs/field-operations-sop.md")} className="btn btn-outline-primary">Field Operations SOP</button>
            <button type="button" onClick={() => openGuide("Agent Workforce Guide", "/docs/agent-workforce-guide.md")} className="btn btn-outline-primary">Agent Workforce Guide</button>
            <button type="button" onClick={() => openGuide("Data Governance Guide", "/docs/data-governance-guide.md")} className="btn btn-outline-primary">Data Governance Guide</button>
          </div>
        </div>

        {guide && (
          <section className="documents-section" style={{ marginBottom: "1.5rem", padding: "1.25rem" }}>
            <div className="table-header"><h3>{guide.title}</h3><button type="button" className="btn btn-outline" onClick={() => setGuide(null)}>Close</button></div>
            <pre style={{ whiteSpace: "pre-wrap", fontFamily: "inherit", lineHeight: 1.6, margin: 0 }}>{guide.content}</pre>
          </section>
        )}

        {preview && (
          <section className="documents-section" style={{ marginBottom: "1.5rem", padding: "1.25rem" }}>
            <div className="table-header"><h3>{preview.title}</h3><button type="button" className="btn btn-outline" onClick={() => { window.URL.revokeObjectURL(preview.url); setPreview(null); }}>Close preview</button></div>
            <iframe title={preview.title} src={preview.url} style={{ width: "100%", height: "70vh", border: "1px solid #d1d5db" }} />
          </section>
        )}

        {error && (
          <div className="alert alert-danger" style={{ marginBottom: "1.5rem" }}>
            {error}
          </div>
        )}

        <div className="documents-section">
          {loading ? (
            <div style={{ padding: "3rem", textAlign: "center", color: "var(--text-muted)" }}>
              Loading documents...
            </div>
          ) : documents.length === 0 ? (
            <div style={{ padding: "3rem", textAlign: "center", color: "var(--text-muted)" }}>
              No documents available at this time.
            </div>
          ) : (
            <table className="documents-table">
              <thead>
                <tr>
                  <th style={{ width: "30%", maxWidth: "350px" }}>Document</th>
                  <th style={{ width: "12%" }}>Category</th>
                  <th style={{ width: "8%" }}>Type</th>
                  <th style={{ width: "12%" }}>Last Updated</th>
                  <th style={{ width: "38%", minWidth: "250px", textAlign: "right" }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {documents.map((doc) => (
                  <tr key={doc.id}>
                    <td style={{ maxWidth: "350px", wordWrap: "break-word" }}>
                      <div className="doc-title" style={{ display: "flex", alignItems: "flex-start", gap: "0.5rem" }}>
                        <i className="bi bi-file-pdf-fill doc-icon" style={{ flexShrink: 0 }} />
                        <div style={{ minWidth: 0, flex: 1 }}>
                          <div style={{ wordBreak: "break-word" }}>{doc.title}</div>
                          {doc.description && (
                            <div className="doc-description" style={{ wordBreak: "break-word" }}>{doc.description}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span
                        className="doc-category badge-category"
                        style={getCategoryStyle(doc.category)}
                      >
                        {doc.category}
                      </span>
                    </td>
                    <td className="doc-meta">PDF</td>
                    <td className="doc-meta">{formatDate(doc.createdAt)}</td>
                    <td style={{ width: "38%", minWidth: "250px", textAlign: "right" }}>
                      <div className="doc-actions" style={{ display: "flex", gap: "0.5rem", flexWrap: "nowrap", justifyContent: "flex-end" }}>
                        <button
                          type="button"
                          className="btn btn-primary"
                          title="Download"
                          onClick={() => handleDownload(doc.id, doc.original_filename)}
                          style={{ whiteSpace: "nowrap", padding: "0.4rem 0.75rem" }}
                        >
                          <i className="bi bi-download" /> Download
                        </button>
                        <button
                          type="button"
                          className="btn btn-outline"
                          title="Preview"
                          onClick={() => handlePreview(doc.id, doc.title)}
                          style={{ whiteSpace: "nowrap", padding: "0.4rem 0.75rem" }}
                        >
                          <i className="bi bi-eye" /> Preview
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
      <Footer />
    </>
  );
};

export default SopsManuals;
