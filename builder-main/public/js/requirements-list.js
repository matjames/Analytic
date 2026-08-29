import { initAuth } from './auth.js';
import { authFetch, authFetchJSON } from './authFetch.js';

let allEntries = [];
let _validateData = null;
let _validateReportName = '';
let _validateId = '';

// -- Validate modal -----------------------------------------------------------

function injectValidateModal() {
    if (document.getElementById('validate-modal')) return;
    const modal = document.createElement('div');
    modal.id = 'validate-modal';
    modal.className = 'validate-modal-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-label', 'Requirements validation results');
    modal.innerHTML = `
        <div class="validate-modal-box">
            <div class="validate-modal-toolbar">
                <span id="validate-modal-title" class="validate-modal-title"></span>
                <div class="validate-toolbar-actions">
                    <button id="validate-rerun" class="btn btn-secondary btn-sm" aria-label="Re-run validation">&#8635; Re-run</button>
                    <button id="validate-dl-csv" class="btn btn-secondary btn-sm" aria-label="Download CSV">&#8595; CSV</button>
                    <button id="validate-dl-pdf" class="btn btn-secondary btn-sm" aria-label="Download PDF">&#8595; PDF</button>
                    <button id="validate-modal-close" class="btn btn-secondary btn-sm" aria-label="Close">Close</button>
                </div>
            </div>
            <div id="validate-modal-body" class="validate-modal-body"></div>
        </div>
    `;
    document.body.appendChild(modal);
    document.getElementById('validate-modal-close').addEventListener('click', closeValidateModal);
    document.getElementById('validate-rerun').addEventListener('click', () => {
        if (_validateId) runValidation(_validateId, _validateReportName);
    });
    document.getElementById('validate-dl-csv').addEventListener('click', () => {
        if (_validateData) downloadValidateCSV(_validateData, _validateReportName);
    });
    document.getElementById('validate-dl-pdf').addEventListener('click', () => {
        if (_validateData) printValidatePDF(_validateData, _validateReportName);
    });
    modal.addEventListener('click', (e) => { if (e.target === modal) closeValidateModal(); });
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closeValidateModal(); });
}

function closeValidateModal() {
    const modal = document.getElementById('validate-modal');
    if (!modal) return;
    modal.classList.remove('open');
    document.body.style.overflow = '';
}

async function runValidation(id, name) {
    const bodyEl = document.getElementById('validate-modal-body');
    const rerunBtn = document.getElementById('validate-rerun');
    bodyEl.innerHTML = '<p class="validate-loading">Running validation\u2026</p>';
    if (rerunBtn) { rerunBtn.disabled = true; rerunBtn.textContent = '\u231B\u2026'; }
    try {
        const basePath = window.BASE_PATH || '';
        const data = await authFetchJSON(`${basePath}/api/requirements-specs/${id}/validate`);
        _validateData = data;
        _validateReportName = name;
        bodyEl.innerHTML = buildValidateHTML(data);
    } catch (err) {
        bodyEl.innerHTML = `<p class="validate-error">Validation failed: ${escapeHtml(err.message || String(err))}</p>`;
    } finally {
        if (rerunBtn) { rerunBtn.disabled = false; rerunBtn.textContent = '\u21BB Re-run'; }
    }
}

async function openValidateModal(id, name) {
    injectValidateModal();
    const titleEl = document.getElementById('validate-modal-title');
    titleEl.textContent = name;
    _validateId = id;
    const modal = document.getElementById('validate-modal');
    modal.classList.add('open');
    document.body.style.overflow = 'hidden';
    await runValidation(id, name);
}

function buildValidateHTML(data) {
    const passRate = data.totalChecks > 0
        ? Math.round((data.passed / data.totalChecks) * 100)
        : 100;
    const scoreClass = data.failed === 0
        ? 'validate-score-pass'
        : data.failed > 2 ? 'validate-score-fail' : 'validate-score-warn';

    const sourceChips = (data.dataSources || []).map(
        (s) => `<span class="validate-source-chip">${escapeHtml(s)}</span>`
    ).join('');

    // Group indicators by section
    const sections = {};
    for (const ind of (data.indicators || [])) {
        const sec = ind.section || ind.visualTitle || 'General';
        (sections[sec] = sections[sec] || []).push(ind);
    }

    let rows = '';
    for (const [sec, inds] of Object.entries(sections)) {
        rows += `<tr class="validate-section-row"><td colspan="7">${escapeHtml(sec)}</td></tr>`;
        for (const ind of inds) {
            const indBadge = ind.passed
                ? '<span class="validate-badge validate-badge-pass">PASS</span>'
                : '<span class="validate-badge validate-badge-fail">FAIL</span>';
            const modeBadge = `<span class="validate-mode-badge validate-mode-${ind.mode}">${ind.mode}</span>`;

            if (!ind.checks || ind.checks.length === 0) {
                rows += `<tr>
                    <td>${escapeHtml(ind.name)}</td>
                    <td>${modeBadge}</td>
                    <td colspan="4"><em class="validate-note">${escapeHtml(ind.note || ind.formula || 'Derived')}</em></td>
                    <td>${indBadge}</td>
                </tr>`;
                continue;
            }

            const firstCheck = ind.checks[0];
            rows += `<tr>
                <td rowspan="${ind.checks.length}">${escapeHtml(ind.name)}</td>
                <td rowspan="${ind.checks.length}">${modeBadge}</td>
                <td>${fmtCheckCol(firstCheck)}</td>
                <td>${escapeHtml(firstCheck.dataElementName || '')}</td>
                <td class="validate-de-id"><code>${escapeHtml(firstCheck.dataElementId || '')}</code></td>
                <td class="${firstCheck.found ? 'validate-cell-pass' : 'validate-cell-fail'}">${escapeHtml(firstCheck.found ? firstCheck.source : firstCheck.message)}</td>
                <td rowspan="${ind.checks.length}">${indBadge}</td>
            </tr>`;

            for (let i = 1; i < ind.checks.length; i++) {
                const ch = ind.checks[i];
                rows += `<tr>
                    <td>${fmtCheckCol(ch)}</td>
                    <td>${escapeHtml(ch.dataElementName || '')}</td>
                    <td class="validate-de-id"><code>${escapeHtml(ch.dataElementId || '')}</code></td>
                    <td class="${ch.found ? 'validate-cell-pass' : 'validate-cell-fail'}">${escapeHtml(ch.found ? ch.source : ch.message)}</td>
                </tr>`;
            }
        }
    }

    return `
        <div class="validate-summary">
            <div class="${scoreClass} validate-score">${passRate}%</div>
            <div class="validate-counts">
                <span class="validate-count-pass">&#10003; ${data.passed} passed</span>
                <span class="validate-count-fail">&#10007; ${data.failed} failed</span>
                <span class="validate-count-total">${data.totalChecks} total checks</span>
            </div>
            <div class="validate-sources"><strong>Data sources:</strong> ${sourceChips || '<em>none declared</em>'}</div>
        </div>
        <div class="validate-table-wrap">
            <table class="validate-table">
                <thead><tr>
                    <th>Indicator</th>
                    <th>Mode</th>
                    <th>MV Column</th>
                    <th>Data Element</th>
                    <th>Data Element ID</th>
                    <th>Result</th>
                    <th>Status</th>
                </tr></thead>
                <tbody>${rows}</tbody>
            </table>
        </div>
        ${buildCoverageHTML(data)}
        ${buildFilterHTML(data)}`;
}

// -- Component coverage section ----------------------------------------------

function buildCoverageHTML(data) {
    if (!data.reportYamlFound && (!data.componentCoverage || data.componentCoverage.length === 0)) {
        const reportId = data.reportId || '';
        const note = reportId
            ? `Report YAML <code>${escapeHtml(reportId)}</code> not found \u2014 link a Report ID to enable component coverage checks.`
            : 'No Report ID linked \u2014 set a Report ID on this spec to enable component coverage checks.';
        return `<div class="validate-coverage-section">
            <h4 class="validate-coverage-title">Component Coverage <span class="validate-badge validate-badge-na">N/A</span></h4>
            <p class="validate-coverage-note">${note}</p>
        </div>`;
    }

    const checks = data.componentCoverage || [];
    const coveragePct = data.coverageTotal > 0
        ? Math.round((data.coveragePassed / data.coverageTotal) * 100)
        : 100;
    const allPass = data.coverageFailed === 0 && !data.typeMismatchCount;
    const pctClass = data.coverageFailed > 0
        ? (data.coverageFailed > 2 ? 'validate-score-fail' : 'validate-score-warn')
        : (data.typeMismatchCount ? 'validate-score-warn' : 'validate-score-pass');

    const rows = checks.map((cc) => {
        const presenceBadge = cc.found
            ? '<span class="validate-badge validate-badge-pass">PRESENT</span>'
            : '<span class="validate-badge validate-badge-fail">MISSING</span>';
        const resultCell = cc.found
            ? `<span class="validate-cell-pass">${escapeHtml(cc.yamlSection || '')}</span>`
            : `<span class="validate-cell-fail">${escapeHtml(cc.message)}</span>`;

        let typeBadge = '';
        if (cc.found && cc.typeChecked) {
            typeBadge = cc.typeMatch
                ? `<span class="validate-badge validate-badge-pass" title="${escapeHtml(cc.typeMessage || '')}">MATCH</span>`
                : `<span class="validate-badge validate-badge-warn" title="${escapeHtml(cc.typeMessage || '')}">MISMATCH</span>`;
        } else if (cc.found && cc.specType && cc.specType.toLowerCase() !== 'other') {
            typeBadge = `<span class="validate-badge validate-badge-na" title="Type not checked">—</span>`;
        }

        const specTypeCell = cc.specType ? escapeHtml(cc.specType) : '<span class="validate-muted">—</span>';
        const yamlTypeCell = cc.yamlType ? `<code>${escapeHtml(cc.yamlType)}</code>` : '';

        return `<tr>
            <td>${escapeHtml(cc.visualTitle)}</td>
            <td>${escapeHtml(cc.specSection || '')}</td>
            <td>${escapeHtml(cc.yamlSection || '')}</td>
            <td>${resultCell}</td>
            <td>${presenceBadge}</td>
            <td>${specTypeCell}</td>
            <td>${yamlTypeCell}</td>
            <td>${typeBadge}</td>
        </tr>`;
    }).join('');

    const typeMismatchSummary = data.typeMismatchCount
        ? `<span class="validate-count-warn">&#9651; ${data.typeMismatchCount} type mismatch${data.typeMismatchCount > 1 ? 'es' : ''}</span>`
        : '';

    return `<div class="validate-coverage-section">
        <h4 class="validate-coverage-title">Component Coverage</h4>
        <div class="validate-coverage-summary">
            <span class="${pctClass} validate-coverage-pct">${coveragePct}%</span>
            <span class="validate-count-pass">&#10003; ${data.coveragePassed} present</span>
            <span class="validate-count-fail">&#10007; ${data.coverageFailed} missing</span>
            ${typeMismatchSummary}
            <span class="validate-count-total">${data.coverageTotal} visuals checked</span>
        </div>
        <div class="validate-table-wrap">
            <table class="validate-table">
                <thead><tr>
                    <th>Visual Title (spec)</th>
                    <th>Spec Section</th>
                    <th>YAML Section</th>
                    <th>Result</th>
                    <th>Status</th>
                    <th>Spec Type</th>
                    <th>YAML Type</th>
                    <th>Type</th>
                </tr></thead>
                <tbody>${rows || '<tr><td colspan="8" class="validate-note">No visual titles declared in spec.</td></tr>'}</tbody>
            </table>
        </div>
    </div>`;
}

// -- Filter checks section ----------------------------------------------------

function buildFilterHTML(data) {
    if (!data.reportYamlFound && (!data.filterChecks || data.filterChecks.length === 0)) {
        return '';
    }

    if (!data.filterChecks || data.filterChecks.length === 0) {
        return `<div class="validate-coverage-section">
            <h4 class="validate-coverage-title">Filter Checks <span class="validate-badge validate-badge-na">N/A</span></h4>
            <p class="validate-coverage-note">No filters declared in the spec \u2014 add filter requirements in section 4 to enable this check.</p>
        </div>`;
    }

    const allPass = data.filterFailed === 0;
    const pctClass = allPass ? 'validate-score-pass' : (data.filterFailed > 2 ? 'validate-score-fail' : 'validate-score-warn');
    const pct = data.filterTotal > 0 ? Math.round((data.filterPassed / data.filterTotal) * 100) : 100;

    const rows = (data.filterChecks || []).map((fc) => {
        const badge = fc.found
            ? '<span class="validate-badge validate-badge-pass">PASS</span>'
            : '<span class="validate-badge validate-badge-fail">FAIL</span>';
        const resultCell = fc.found
            ? `<span class="validate-cell-pass">set in YAML</span>`
            : `<span class="validate-cell-fail">${escapeHtml(fc.message)}</span>`;
        return `<tr>
            <td><code>${escapeHtml(fc.filter)}</code></td>
            <td>${resultCell}</td>
            <td>${badge}</td>
        </tr>`;
    }).join('');

    return `<div class="validate-coverage-section">
        <h4 class="validate-coverage-title">Filter Checks</h4>
        <div class="validate-coverage-summary">
            <span class="${pctClass} validate-coverage-pct">${pct}%</span>
            <span class="validate-count-pass">&#10003; ${data.filterPassed} set</span>
            <span class="validate-count-fail">&#10007; ${data.filterFailed} missing</span>
            <span class="validate-count-total">${data.filterTotal} filters checked</span>
        </div>
        <div class="validate-table-wrap">
            <table class="validate-table">
                <thead><tr>
                    <th>Required Filter</th>
                    <th>Result</th>
                    <th>Status</th>
                </tr></thead>
                <tbody>${rows}</tbody>
            </table>
        </div>
    </div>`;
}

// -- Validate downloads -------------------------------------------------------

function downloadValidateCSV(data, reportName) {
    const headers = ['Indicator', 'Mode', 'MV Column', 'Data Element', 'Data Element ID', 'Result', 'Status'];
    const csvRows = [headers];
    for (const ind of (data.indicators || [])) {
        if (!ind.checks || ind.checks.length === 0) {
            csvRows.push([ind.name, ind.mode, ind.note || 'Derived', '', '', '', ind.passed ? 'PASS' : 'FAIL']);
            continue;
        }
        for (const ch of ind.checks) {
            const mvCol = ch.resolvedColumn || ch.column;
            csvRows.push([
                ind.name,
                ind.mode,
                mvCol,
                ch.dataElementName || '',
                ch.dataElementId || '',
                ch.found ? ch.source : ch.message,
                ind.passed ? 'PASS' : 'FAIL',
            ]);
        }
    }

    // Component coverage section
    if (data.componentCoverage && data.componentCoverage.length > 0) {
        csvRows.push([]);  // blank separator row
        csvRows.push(['--- Component Coverage ---', '', '', '', '', '', '', '', '']);
        csvRows.push(['Visual Title (spec)', 'Spec Section', 'YAML Section', 'YAML Component', 'Result', 'Status', 'Spec Type', 'YAML Type', 'Type Check']);
        for (const cc of data.componentCoverage) {
            let typeCheck = '';
            if (cc.found && cc.typeChecked) {
                typeCheck = cc.typeMatch ? 'MATCH' : `MISMATCH: ${cc.typeMessage || ''}`;
            }
            csvRows.push([
                cc.visualTitle,
                cc.specSection || '',
                cc.yamlSection || '',
                cc.yamlComponent || '',
                cc.found ? cc.message : cc.message,
                cc.found ? 'PRESENT' : 'MISSING',
                cc.specType || '',
                cc.yamlType || '',
                typeCheck,
            ]);
        }
    }

    // Filter checks section
    if (data.filterChecks && data.filterChecks.length > 0) {
        csvRows.push([]);  // blank separator row
        csvRows.push(['--- Filter Checks ---', '', '']);
        csvRows.push(['Required Filter', 'Result', 'Status']);
        for (const fc of data.filterChecks) {
            csvRows.push([fc.filter, fc.message, fc.found ? 'PASS' : 'FAIL']);
        }
    }
    const csv = csvRows.map((r) =>
        r.map((v) => `"${String(v || '').replace(/"/g, '""')}"`).join(',')
    ).join('\r\n');
    const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${(reportName || 'validation').replace(/[^\w\- ]/g, '_')}-validation.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function printValidatePDF(data, reportName) {
    const html = buildValidateHTML(data);
    const fullDoc = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8">
<title>${escapeHtml(reportName)} \u2014 Validation Results</title>
<style>
* { box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; font-size: 12px; color: #1a1a1a; padding: 24px; margin: 0; }
h2 { font-size: 15px; margin: 0 0 12px; }
.validate-summary { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 10px 12px; border: 1px solid #ccc; border-radius: 4px; margin-bottom: 12px; background: #f7f7f7; }
.validate-score { font-size: 24px; font-weight: 800; }
.validate-score-pass { color: #12522c; } .validate-score-warn { color: #7c5700; } .validate-score-fail { color: #8b2d1f; }
.validate-counts { display: flex; gap: 10px; font-size: 12px; font-weight: 600; }
.validate-count-pass { color: #12522c; } .validate-count-fail { color: #8b2d1f; }
.validate-sources { font-size: 11px; color: #555; }
.validate-source-chip { display: inline-block; background: #e0e0e0; border-radius: 3px; padding: 1px 6px; margin: 1px; font-size: 10px; font-family: monospace; }
.validate-table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 11px; }
th { padding: 7px 10px; text-align: left; font-weight: 700; background: #e8eaf6; border: 1px solid #bbb; white-space: nowrap; }
td { padding: 5px 9px; border: 1px solid #ddd; vertical-align: top; }
.validate-section-row td { background: #183a3f; color: #fff; font-weight: 700; padding: 5px 9px; }
.validate-cell-pass { color: #12522c; font-family: monospace; font-size: 10px; }
.validate-cell-fail { color: #8b2d1f; }
.validate-badge { display: inline-block; padding: 1px 6px; border-radius: 3px; font-size: 10px; font-weight: 700; white-space: nowrap; }
.validate-badge-pass { background: #e4f4ea; color: #12522c; border: 1px solid #a7d1b5; }
.validate-badge-fail { background: #fbe9e7; color: #8b2d1f; border: 1px solid #edb7af; }
.validate-mode-badge { display: inline-block; padding: 1px 5px; border-radius: 3px; font-size: 10px; font-weight: 600; }
.validate-mode-direct { background: #dce8f4; color: #1a4f7a; } .validate-mode-calculated { background: #f4eddc; color: #6b3a0a; }
.validate-role { font-size: 9px; color: #777; }
.validate-de-id, code { font-family: monospace; font-size: 10px; color: #555; background: #f0f0f0; padding: 0 3px; border-radius: 2px; }
.validate-note { color: #777; font-style: italic; }
.validate-coverage-section { margin-top: 20px; }
.validate-coverage-title { font-size: 13px; font-weight: 700; margin: 0 0 8px; color: #1a1a1a; }
.validate-coverage-summary { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; font-size: 12px; font-weight: 600; margin-bottom: 8px; }
.validate-coverage-pct { font-size: 20px; font-weight: 800; }
.validate-coverage-note { font-size: 11px; color: #777; margin: 4px 0 0; }
.validate-badge-na { background: #f0f0f0; color: #777; border: 1px solid #ccc; }
@media print { @page { margin: 12mm; size: A4 landscape; } body { padding: 0; font-size: 10px; } table { page-break-inside: auto; } tr { page-break-inside: avoid; } thead { display: table-header-group; } }
</style></head><body>
<h2>${escapeHtml(reportName)} \u2014 Validation Results</h2>
${html}
</body></html>`;

    // Hidden iframe avoids popup blockers
    let frame = document.getElementById('_print-frame');
    if (!frame) {
        frame = document.createElement('iframe');
        frame.id = '_print-frame';
        frame.style.cssText = 'position:fixed;width:0;height:0;border:none;visibility:hidden;';
        document.body.appendChild(frame);
    }
    frame.srcdoc = fullDoc;
    frame.onload = () => {
        frame.contentWindow.focus();
        frame.contentWindow.print();
        frame.onload = null;
    };
}

// -- Requirements viewer modal ------------------------------------------------

function injectPdfModal() {
    if (document.getElementById('pdf-modal')) return;
    const modal = document.createElement('div');
    modal.id = 'pdf-modal';
    modal.className = 'pdf-modal-overlay';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-label', 'Requirements document viewer');
    modal.innerHTML = `
        <div class="pdf-modal-box">
            <div class="pdf-modal-toolbar">
                <span id="pdf-modal-title" class="pdf-modal-title"></span>
                <div class="pdf-modal-actions">
                    <button id="pdf-modal-print" class="btn btn-secondary btn-sm" type="button">&#8595; PDF</button>
                    <button id="pdf-modal-close" class="btn btn-secondary btn-sm" aria-label="Close viewer">Close</button>
                </div>
            </div>
            <iframe id="pdf-modal-frame" class="pdf-modal-frame" title="Requirements document"
                    srcdoc="<p style='font:14px sans-serif;padding:24px;color:#555'>Loading\u2026</p>"></iframe>
        </div>
    `;
    document.body.appendChild(modal);

    document.getElementById('pdf-modal-close').addEventListener('click', closePdfModal);
    modal.addEventListener('click', (e) => { if (e.target === modal) closePdfModal(); });
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') closePdfModal(); });
}

async function openPdfModal(viewUrl, _pdfUrl, title) {
    injectPdfModal();
    document.getElementById('pdf-modal-title').textContent = title;

    const frame = document.getElementById('pdf-modal-frame');
    frame.srcdoc = "<p style='font:14px sans-serif;padding:24px;color:#555'>Loading\u2026</p>";

    document.getElementById('pdf-modal-print').onclick = () => {
        if (frame.contentWindow) frame.contentWindow.print();
    };

    const modal = document.getElementById('pdf-modal');
    modal.classList.add('open');
    document.body.style.overflow = 'hidden';

    try {
        const response = await authFetch(viewUrl);
        if (!response.ok) throw new Error(`HTTP ${response.status} ${response.statusText}`);
        frame.srcdoc = await response.text();
    } catch (err) {
        frame.srcdoc = `<p style="font:14px sans-serif;padding:24px;color:#c00">Failed to load: ${escapeHtml(err.message)}</p>`;
    }
}

function closePdfModal() {
    const modal = document.getElementById('pdf-modal');
    if (!modal) return;
    modal.classList.remove('open');
    document.getElementById('pdf-modal-frame').srcdoc = '';
    document.body.style.overflow = '';
}

// -- Utilities ----------------------------------------------------------------

function byId(id) {
    return document.getElementById(id);
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

/** Renders the MV Column cell. When resolvedColumn is set the spec used a DHIS2 UID. */
function fmtCheckCol(ch) {
    const role = ch.role && ch.role !== 'direct'
        ? ` <span class="validate-role">(${escapeHtml(ch.role)})</span>` : '';
    if (ch.resolvedColumn) {
        return `<code>${escapeHtml(ch.column)}</code>`
            + ` <span class="validate-resolved">\u2192 <code>${escapeHtml(ch.resolvedColumn)}</code></span>${role}`;
    }
    return `<code>${escapeHtml(ch.column)}</code>${role}`;
}

function renderStatus(kind, text) {
    const status = byId('list-status');
    status.classList.remove('hidden', 'success', 'error');
    status.classList.add(kind);
    status.textContent = text;
}

function formatDate(unixSeconds) {
    if (!unixSeconds) return '';
    const d = new Date(unixSeconds * 1000);
    return Number.isNaN(d.getTime()) ? '' : d.toLocaleString();
}

// -- List rendering -----------------------------------------------------------

function renderList(entries) {
    const container = byId('requirements-list');
    if (!entries.length) {
        container.innerHTML = '<p class="selector-empty">No requirements documents found.</p>';
        return;
    }

    const basePath = window.BASE_PATH || '';
    const tableRows = entries.map((e) => {
        const htmlUrl = `${basePath}/api/requirements-specs/${e.id}/html`;
        const pdfUrl  = `${basePath}/api/requirements-specs/${e.id}/pdf`;
        return `<tr>
            <td><span class="req-report-name">${escapeHtml(e.reportName || 'Untitled Report')}</span></td>
            <td class="req-id-cell">${escapeHtml(e.reportId || '-')}</td>
            <td>${escapeHtml(e.programArea || '-')}</td>
            <td>${escapeHtml(e.requestingDepartment || '-')}</td>
            <td>${escapeHtml(e.businessOwner || '-')}</td>
            <td>${escapeHtml(e.status || '-')}</td>
            <td>${escapeHtml(e.submittedByUsername || '-')}</td>
            <td class="req-date-cell">${escapeHtml(formatDate(e.createdAt))}</td>
            <td class="req-actions-cell">
                <button class="btn btn-secondary btn-sm requirements-pdf-btn"
                        data-view-url="${htmlUrl}"
                        data-pdf-url="${pdfUrl}"
                        data-pdf-title="${escapeHtml(e.reportName || 'Requirements')}"
                        type="button">View</button>
                <button class="btn btn-secondary btn-sm requirements-validate-btn"
                        data-id="${e.id}"
                        data-name="${escapeHtml(e.reportName || 'Requirements')}"
                        type="button">Validate</button>
                <a href="${basePath}/requirements/form?edit=${e.id}"
                   class="btn btn-secondary btn-sm">Edit</a>
            </td>
        </tr>`;
    }).join('');

    container.innerHTML = `
        <div class="component-table table-cell-boundaries-enabled table-vertical-lines-enabled table-header-style-enhanced-readable">
            <table>
                <thead><tr>
                    <th>Report Name</th>
                    <th>Report ID</th>
                    <th>Program</th>
                    <th>Department</th>
                    <th>Owner</th>
                    <th>Status</th>
                    <th>Submitted By</th>
                    <th>Date</th>
                    <th>Actions</th>
                </tr></thead>
                <tbody>${tableRows}</tbody>
            </table>
        </div>`;
}

function applySearch() {
    const q = byId('requirements-search').value.trim().toLowerCase();
    if (!q) { renderList(allEntries); return; }

    const filtered = allEntries.filter((e) => {
        const hay = [
            e.reportName, e.reportId, e.programArea,
            e.requestingDepartment, e.businessOwner,
            e.status, e.submittedByUsername, e.submittedByEmail,
        ].map((v) => String(v || '').toLowerCase()).join(' ');
        return hay.includes(q);
    });
    renderList(filtered);
}

async function loadRequirements() {
    const listEl = byId('requirements-list');
    listEl.innerHTML = '<p class="selector-loading">Loading requirements documents...</p>';
    try {
        const basePath = window.BASE_PATH || '';
        const data = await authFetchJSON(`${basePath}/api/requirements-specs?limit=500`);
        allEntries = Array.isArray(data.entries) ? data.entries : [];
        renderList(allEntries);
        renderStatus('success', `${allEntries.length} requirements document(s) loaded.`);
    } catch (error) {
        listEl.innerHTML = '<p class="selector-empty">Failed to load requirements documents.</p>';
        renderStatus('error', error.message || 'Failed to load requirements documents');
    }
}

// -- Init ---------------------------------------------------------------------

async function initPage() {
    await initAuth();

    byId('requirements-refresh').addEventListener('click', loadRequirements);
    byId('requirements-search').addEventListener('input', applySearch);

    byId('requirements-list').addEventListener('click', (e) => {
        const pdfBtn = e.target.closest('.requirements-pdf-btn');
        if (pdfBtn) {
            openPdfModal(pdfBtn.dataset.viewUrl, pdfBtn.dataset.pdfUrl, pdfBtn.dataset.pdfTitle);
            return;
        }
        const valBtn = e.target.closest('.requirements-validate-btn');
        if (valBtn) {
            openValidateModal(valBtn.dataset.id, valBtn.dataset.name);
        }
    });

    await loadRequirements();
}

initPage().catch((error) => {
    renderStatus('error', `Unable to initialize requirements list: ${error.message || error}`);
});
