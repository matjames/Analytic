import React, { Fragment, useEffect, useState } from "react";
import Header from "./Header";

const slugify = (text) =>
  text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "-");

function extractHeadings(md) {
  const headings = [];
  const regex = /^(#{2,3})\s+(.*)$/gm; // capture ## and ### headings
  let match;
  while ((match = regex.exec(md)) !== null) {
    const level = match[1].length; // 2 or 3
    const label = match[2].trim();
    const id = slugify(label);
    headings.push({ id, label, level });
  }
  return headings;
}

function markdownToHtml(md) {
  if (!md) return "";
  let html = md.replace(/```([\s\S]*?)```/g, (_m, code) => {
    return `<pre><code>${code.replace(/</g, "&lt;").replace(/>/g, "&gt;")}</code></pre>`;
  });
  
  // Replace HTTP method patterns (e.g., **GET** `/api/endpoint`) with styled badges
  html = html.replace(/\*\*(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\*\*\s+`([^`]+)`/g, 
    (match, method, path) => {
      const methodClass = method.toLowerCase();
      return `<div class="api-endpoint-header"><span class="api-method ${methodClass}">${method}</span><code class="api-path">${path}</code></div>`;
    }
  );
  
  html = html.replace(/^### (.*)$/gm, (_m, title) => `<h3 id="${slugify(title)}">${title}</h3>`);
  html = html.replace(/^## (.*)$/gm, (_m, title) => `<h2 id="${slugify(title)}">${title}</h2>`);
  html = html.replace(/^# (.*)$/gm, "<h1>$1</h1>");
  html = html.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");
  html = html.replace(/\*(.+?)\*/g, "<em>$1</em>");
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  html = html.replace(/(?:^|\n)(- .*(?:\n- .*)*)/g, (match) => {
    const items = match
      .trim()
      .split("\n")
      .map((line) => line.replace(/^- /, "<li>") + "</li>")
      .join("");
    return `<ul>${items}</ul>`;
  });
  
  // Replace horizontal rules
  html = html.replace(/^---$/gm, "<hr>");
  
  html = html
    .split(/\n{2,}/)
    .map((block) => {
      const trimmed = block.trim();
      if (!trimmed) return "";
      if (/^<(h\d|ul|pre|table|blockquote|code|div|hr)/i.test(trimmed)) return trimmed;
      return `<p>${trimmed.replace(/\n/g, "<br/>")}</p>`;
    })
    .join("");
  return html;
}

const iconRules = [
  { match: /base url/i, icon: "bi-info-circle" },
  { match: /health check/i, icon: "bi-heart-pulse" },
  { match: /admin levels/i, icon: "bi-diagram-3" },
  { match: /admin units/i, icon: "bi-diagram-3" },
  { match: /facility levels/i, icon: "bi-building" },
  { match: /ownership/i, icon: "bi-person-badge" },
  { match: /authority/i, icon: "bi-shield-lock" },
  { match: /status/i, icon: "bi-flag" },
  { match: /facilit(y|ies)/i, icon: "bi-hospital" },
  { match: /authentication|authorization/i, icon: "bi-key" },
  { match: /rate limit/i, icon: "bi-speedometer2" },
  { match: /error/i, icon: "bi-exclamation-triangle" },
  { match: /data model/i, icon: "bi-database" },
  { match: /materialized path/i, icon: "bi-diagram-3" },
  { match: /example/i, icon: "bi-code-slash" },
  { match: /notes/i, icon: "bi-journal-text" },
];

const iconForLabel = (label) => {
  const rule = iconRules.find((r) => r.match.test(label));
  return rule ? rule.icon : "bi-dot";
};

const ApiDocs = () => {
  const [html, setHtml] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [headings, setHeadings] = useState([]);

  useEffect(() => {
    async function load() {
      try {
        const response = await fetch("/MFL_API_Documentation.md");
        if (!response.ok) {
          throw new Error(`Failed to load API documentation: ${response.statusText}`);
        }
        const md = await response.text();
        setHeadings(extractHeadings(md).filter((h) => h.level === 2));
        setHtml(markdownToHtml(md));
      } catch (e) {
        const message =
          e?.message ||
          "Failed to load API documentation";
        setError(message);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  return (
    <Fragment>
      <Header />
      <div className="container">
        <div className="page-title-section">
          <div className="page-title-header">
            <h2>
              <i className="bi bi-code-slash me-2"></i>
              API Documentation
            </h2>
            <p>Complete reference for StatGate Field Operations Registry API endpoints and integrations.</p>
          </div>
        </div>

        <div className="api-layout">
          <aside className="api-sidebar">
            <div className="api-sidebar-section">
              <div className="api-sidebar-title">
                <i className="bi bi-list-ul me-2"></i>
                On This Page
              </div>
              <ul className="api-sidebar-menu">
                {headings.map((h) => (
                  <li key={h.id}>
                    <a href={`#${h.id}`} className="sidebar-link">
                      <i className={`bi ${iconForLabel(h.label)}`} /> 
                      <span>{h.label}</span>
                    </a>
                  </li>
                ))}
                {!headings.length && (
                  <li>
                    <span className="api-sidebar-empty">No sections found</span>
                  </li>
                )}
              </ul>
            </div>
          </aside>

          <main className="api-content">
            {loading && (
              <div className="api-docs-state">
                <div className="loading-spinner">
                  <i className="bi bi-arrow-repeat"></i>
                </div>
                <p>Loading API documentation...</p>
              </div>
            )}
            {!loading && error && (
              <div className="api-docs-error">
                <i className="bi bi-exclamation-triangle-fill"></i>
                <h3>Error Loading Documentation</h3>
                <p>{error}</p>
              </div>
            )}
            {!loading && !error && (
              <div
                id="markdown-content"
                className="markdown-content"
                dangerouslySetInnerHTML={{
                  __html: html || "<p>No documentation available.</p>",
                }}
              />
            )}
          </main>
        </div>
      </div>
    </Fragment>
  );
}

export default ApiDocs;
