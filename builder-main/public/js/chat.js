// Chat view — "Ask" natural language interface
// State lives in JS objects, DOM is just a view.

const chatState = {
    view: 'home', // 'home' or 'chat'
    messages: [],  // { role: 'user'|'assistant', type: 'text'|'validation'|'result'|'error', content: ... }
    pendingQuery: null, // ChatParsedQuery awaiting confirmation
    isLoading: false,
    originalQuestion: '', // stored for retry
    catalog: null, // { tables: [{table, description, columns: [{name, type, numeric}]}] }
    editingQuery: false // true when user is editing table/columns in validation card
};

// Track chart instances created in chat for cleanup
const chatChartInstances = {};

// Fetch the table/column catalog (once)
async function fetchChatCatalog() {
    if (chatState.catalog) return chatState.catalog;
    try {
        const resp = await fetch(`${API_BASE}/ask/catalog`, {
            headers: await refreshedAuthHeaders()
        });
        if (resp.ok) {
            chatState.catalog = await resp.json();
        }
    } catch (e) {
        // Catalog is optional — picker just won't be available
    }
    return chatState.catalog;
}

// ============== View Switching ==============

function showChatView() {
    chatState.view = 'chat';
    // Hide sidebar and main content, show chat — all inside .main-layout
    const sidebar = document.querySelector('.sidebar');
    const main = document.querySelector('.main-content');
    if (sidebar) sidebar.style.display = 'none';
    if (main) main.style.display = 'none';
    document.getElementById('chat-container').style.display = 'flex';
    // Pre-fetch catalog in background
    fetchChatCatalog();
    // Hide sidebar toggle — not relevant in chat view
    const sidebarToggle = document.getElementById('sidebar-menu-toggle');
    if (sidebarToggle) sidebarToggle.style.display = 'none';
    // Show disclaimer if not dismissed
    const disclaimer = document.getElementById('chat-disclaimer');
    if (disclaimer && !localStorage.getItem('chat_disclaimer_dismissed')) {
        disclaimer.style.display = 'block';
    }
    updateNavPills();
    // Focus input
    const input = document.getElementById('chat-input');
    if (input) input.focus();
}

function dismissChatDisclaimer() {
    const disclaimer = document.getElementById('chat-disclaimer');
    if (disclaimer) disclaimer.style.display = 'none';
    localStorage.setItem('chat_disclaimer_dismissed', '1');
}

function showHomeView() {
    chatState.view = 'home';
    destroyChatCharts();
    destroyChatMaps();
    document.getElementById('chat-container').style.display = 'none';
    // Restore sidebar and main content
    const sidebar = document.querySelector('.sidebar');
    const main = document.querySelector('.main-content');
    if (sidebar) sidebar.style.display = '';
    if (main) main.style.display = '';
    const sidebarToggle = document.getElementById('sidebar-menu-toggle');
    if (sidebarToggle) sidebarToggle.style.display = '';
    updateNavPills();
}

function updateNavPills() {
    document.querySelectorAll('.nav-pill').forEach(pill => {
        pill.classList.toggle('active', pill.dataset.view === chatState.view);
    });
}

// ============== Message Rendering ==============

function renderMessages() {
    const container = document.getElementById('chat-messages');
    if (!container) return;

    // Check if we should show empty state or messages
    const emptyState = document.getElementById('chat-empty-state');
    if (chatState.messages.length === 0) {
        if (emptyState) emptyState.style.display = '';
        container.innerHTML = '';
        return;
    }
    if (emptyState) emptyState.style.display = 'none';

    container.innerHTML = chatState.messages.map((msg, i) => {
        if (msg.role === 'user') {
            return `<div class="chat-msg chat-msg-user"><div class="chat-bubble chat-bubble-user">${escapeHtml(msg.content)}</div></div>`;
        }
        if (msg.type === 'validation') {
            return renderValidationCard(msg.content, i);
        }
        if (msg.type === 'result') {
            return renderResultMessage(msg.content, i);
        }
        if (msg.type === 'error') {
            return `<div class="chat-msg chat-msg-assistant"><div class="chat-bubble chat-bubble-error">${escapeHtml(msg.content)}</div></div>`;
        }
        return `<div class="chat-msg chat-msg-assistant"><div class="chat-bubble chat-bubble-assistant">${escapeHtml(msg.content)}</div></div>`;
    }).join('');

    // Show loading indicator
    if (chatState.isLoading) {
        container.innerHTML += '<div class="chat-msg chat-msg-assistant"><div class="chat-bubble chat-bubble-assistant chat-typing"><span></span><span></span><span></span></div></div>';
    }

    // Scroll to bottom
    container.scrollTop = container.scrollHeight;
}

function renderValidationCard(parsed, msgIndex) {
    const p = parsed;

    // Show full schema.table
    const tableDisplay = p.table || 'N/A';

    // Show actual columns with function and alias
    const aggRows = (p.aggregations || []).map(a => {
        const colDisplay = escapeHtml(a.column);
        const fnDisplay = escapeHtml(a.function.toUpperCase());
        const aliasDisplay = a.alias ? ` as "${escapeHtml(a.alias)}"` : '';
        return `<div class="chat-query-detail">${fnDisplay}(<code>${colDisplay}</code>)${aliasDisplay}</div>`;
    }).join('');

    const groupRows = (p.groupBy || []).map(g => {
        const fmt = g.format ? ` (${escapeHtml(g.format)})` : '';
        return `<code>${escapeHtml(g.field)}</code>${fmt}`;
    }).join(', ');

    const filterParts = [];
    if (p.filters) {
        if (p.filters.district) filterParts.push(`District: ${p.filters.district}`);
        if (p.filters.year) filterParts.push(`Year: ${p.filters.year}`);
        if (p.filters.month) filterParts.push(`Month: ${p.filters.month}`);
        if (p.filters.quarter) filterParts.push(`Quarter: ${p.filters.quarter}`);
        if (p.filters.week) filterParts.push(`Week: ${p.filters.week}`);
    }

    const confirmed = chatState.pendingQuery === null && msgIndex < chatState.messages.length - 1;
    const unresolved = !p.table;

    const confidenceBadges = {
        low: '<span class="chat-confidence-badge chat-confidence-low">Low confidence &mdash; please double check</span>',
        medium: '<span class="chat-confidence-badge chat-confidence-medium">Please double check the query below</span>',
        none: '<span class="chat-confidence-badge chat-confidence-none">Could not match this to a report</span>'
    };
    const confidenceBadge = confidenceBadges[p.confidence] || '';

    return `<div class="chat-msg chat-msg-assistant">
        <div class="chat-bubble chat-bubble-assistant chat-validation-card">
            ${confidenceBadge}
            <p class="chat-validation-explanation">${escapeHtml(p.explanation || '')}</p>
            ${unresolved ? '' : `<div class="chat-query-review">
                <div class="chat-query-section">
                    <span class="chat-field-label">Table:</span>
                    <code class="chat-table-name">${escapeHtml(tableDisplay)}</code>
                </div>
                <div class="chat-query-section">
                    <span class="chat-field-label">Measures:</span>
                    <div class="chat-query-details">${aggRows || '<span class="chat-muted">None</span>'}</div>
                </div>
                ${groupRows ? `<div class="chat-query-section">
                    <span class="chat-field-label">Group by:</span> ${groupRows}
                </div>` : ''}
                ${filterParts.length ? `<div class="chat-query-section">
                    <span class="chat-field-label">Filters:</span> ${escapeHtml(filterParts.join(', '))}
                </div>` : ''}
            </div>`}
            ${confirmed ? '<div class="chat-confirmed">Confirmed</div>' : `
            <div class="chat-validation-actions">
                ${unresolved ? '' : `<button class="chat-btn chat-btn-confirm" onclick="confirmQuery()">Run query</button>
                <button class="chat-btn chat-btn-edit" onclick="openQueryEditor()">Edit query</button>`}
                <button class="chat-btn chat-btn-reject" onclick="showCorrectionInput()">Rephrase</button>
            </div>
            <div class="chat-correction-input" id="chat-correction-input" style="display:${unresolved ? 'flex' : 'none'};">
                <input type="text" id="chat-correction-text" placeholder="What should I change?" maxlength="300"
                    onkeydown="if(event.key==='Enter')submitCorrection()">
                <button class="chat-btn chat-btn-confirm" onclick="submitCorrection()">Retry</button>
            </div>
            <div class="chat-query-editor" id="chat-query-editor" style="display:none;"></div>`}
        </div>
    </div>`;
}

function renderResultMessage(result, msgIndex) {
    let html = `<div class="chat-msg chat-msg-assistant">
        <div class="chat-bubble chat-bubble-assistant">
            <p class="chat-result-summary">${escapeHtml(result.summary || '')}</p>`;

    if (result.rows && result.rows.length > 0) {
        const showRows = result.rows.length > 10;
        html += `<div class="chat-result-table-wrapper${showRows ? ' chat-table-collapsed' : ''}">
            <table class="chat-result-table">
                <thead><tr>${result.headers.map(h => `<th>${escapeHtml(h)}</th>`).join('')}</tr></thead>
                <tbody>${result.rows.map(row =>
                    `<tr>${row.map(cell => `<td>${cell !== null && cell !== undefined ? escapeHtml(String(cell)) : ''}</td>`).join('')}</tr>`
                ).join('')}</tbody>
            </table>
        </div>`;
        if (showRows) {
            html += `<button class="chat-btn-link" onclick="this.previousElementSibling.classList.toggle('chat-table-collapsed');this.textContent=this.textContent==='Show all rows'?'Collapse':'Show all rows'">Show all rows</button>`;
        }
        html += `<div class="chat-result-meta">
            <div class="chat-row-count">${result.rowCount} row${result.rowCount !== 1 ? 's' : ''}</div>
            <button class="table-export-btn" data-tooltip="Download as CSV" onclick="exportChatResultCSV(${msgIndex})"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></button>
        </div>`;

        // Chart type buttons
        const suitable = getSuitableChartTypes(result);
        const hasMap = result.mapData && result.mapData.district_values && Object.keys(result.mapData.district_values).length > 0;
        if (suitable.length > 0 || hasMap) {
            html += `<div class="chat-chart-actions">`;
            if (hasMap) {
                html += `<button class="chat-btn chat-chart-btn" onclick="renderChatMap(${msgIndex})">Map</button>`;
            }
            suitable.forEach(type => {
                const label = type === 'bar' ? 'Bar chart' : type === 'line' ? 'Line chart' : 'Pie chart';
                html += `<button class="chat-btn chat-chart-btn" onclick="renderChatChart('${type}', ${msgIndex})">${label}</button>`;
            });
            html += `</div>`;
        }

        // Chart container (rendered into by renderChatChart)
        html += `<div class="chat-chart-container" id="chat-chart-${msgIndex}"></div>`;
    }

    html += '<p class="chat-result-caveat">AI-generated — verify figures against official sources before use.</p>';
    html += '</div></div>';
    return html;
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// ============== Conversation History ==============

// Build up to 3 prior exchanges from chatState.messages for follow-up context.
// Each exchange: user question → validation (parsed query) → optional result.
function buildChatHistory() {
    const history = [];
    const msgs = chatState.messages;
    let i = 0;
    while (i < msgs.length && history.length < 3) {
        if (msgs[i].role === 'user') {
            const turn = { question: msgs[i].content, table: '', explanation: '', summary: '' };
            i++;
            // Look for the following assistant validation card
            if (i < msgs.length && msgs[i].role === 'assistant' && msgs[i].type === 'validation') {
                const p = msgs[i].content;
                turn.table = p.table || '';
                turn.explanation = p.explanation || '';
                i++;
            }
            // Look for the following assistant result
            if (i < msgs.length && msgs[i].role === 'assistant' && msgs[i].type === 'result') {
                turn.summary = msgs[i].content.summary || '';
                i++;
            }
            if (turn.table) history.push(turn);
        } else {
            i++;
        }
    }
    return history;
}

// ============== User Actions ==============

async function submitQuestion() {
    const input = document.getElementById('chat-input');
    const question = (input.value || '').trim();
    if (!question || chatState.isLoading) return;

    window.telemetry?.track('ask.submit', { hasHistory: chatState.messages.length > 0 ? '1' : '0' });

    const history = buildChatHistory();
    chatState.originalQuestion = question;
    chatState.pendingQuery = null;
    input.value = '';
    destroyChatCharts();
    destroyChatMaps();

    chatState.messages.push({ role: 'user', type: 'text', content: question });
    chatState.isLoading = true;
    renderMessages();

    try {
        const body = { question };
        if (history.length > 0) body.history = history;
        const resp = await fetch(`${API_BASE}/ask/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...await refreshedAuthHeaders() },
            body: JSON.stringify(body)
        });

        if (!resp.ok) {
            const err = await resp.json().catch(() => ({ error: 'Request failed' }));
            throw new Error(err.error || `Server error (${resp.status})`);
        }

        const data = await resp.json();
        data.parsed.confidence = data.confidence;
        chatState.pendingQuery = data.parsed;
        chatState.messages.push({ role: 'assistant', type: 'validation', content: data.parsed });
    } catch (e) {
        chatState.messages.push({ role: 'assistant', type: 'error', content: e.message });
    } finally {
        chatState.isLoading = false;
        renderMessages();
    }
}

async function confirmQuery() {
    if (!chatState.pendingQuery || chatState.isLoading) return;

    const parsed = chatState.pendingQuery;
    chatState.pendingQuery = null; // Clear so validation card shows as confirmed
    chatState.isLoading = true;
    renderMessages();

    try {
        const resp = await fetch(`${API_BASE}/ask/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...await refreshedAuthHeaders() },
            body: JSON.stringify({ parsed })
        });

        if (!resp.ok) {
            const err = await resp.json().catch(() => ({ error: 'Request failed' }));
            throw new Error(err.error || `Server error (${resp.status})`);
        }

        const data = await resp.json();
        applyChatDefaultSort(data);
        chatState.messages.push({ role: 'assistant', type: 'result', content: data });
    } catch (e) {
        chatState.messages.push({ role: 'assistant', type: 'error', content: e.message });
    } finally {
        chatState.isLoading = false;
        renderMessages();
    }
}

function showCorrectionInput() {
    const el = document.getElementById('chat-correction-input');
    if (el) {
        el.style.display = 'flex';
        document.getElementById('chat-correction-text').focus();
    }
}

function cancelQueryEditor() {
    const el = document.getElementById('chat-query-editor');
    if (el) el.style.display = 'none';
    window.telemetry?.track('ask.cancel');
}

async function submitCorrection() {
    const correctionInput = document.getElementById('chat-correction-text');
    const correction = (correctionInput ? correctionInput.value : '').trim();
    if (!correction || chatState.isLoading) return;

    const history = buildChatHistory();
    chatState.pendingQuery = null;
    chatState.isLoading = true;

    chatState.messages.push({ role: 'user', type: 'text', content: correction });
    renderMessages();

    try {
        const body = { question: chatState.originalQuestion, correction };
        if (history.length > 0) body.history = history;
        const resp = await fetch(`${API_BASE}/ask/query`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...await refreshedAuthHeaders() },
            body: JSON.stringify(body)
        });

        if (!resp.ok) {
            const err = await resp.json().catch(() => ({ error: 'Request failed' }));
            throw new Error(err.error || `Server error (${resp.status})`);
        }

        const data = await resp.json();
        data.parsed.confidence = data.confidence;
        chatState.pendingQuery = data.parsed;
        chatState.messages.push({ role: 'assistant', type: 'validation', content: data.parsed });
    } catch (e) {
        chatState.messages.push({ role: 'assistant', type: 'error', content: e.message });
    } finally {
        chatState.isLoading = false;
        renderMessages();
    }
}

function handleExampleClick(question) {
    const input = document.getElementById('chat-input');
    if (input) {
        input.value = question;
        submitQuestion();
    }
}

// Get auth headers if auth is enabled
function getAuthHeaders() {
    if (window._chatAuthToken) {
        return { 'Authorization': 'Bearer ' + window._chatAuthToken };
    }
    return {};
}

// Refresh token before each chat API call (async wrapper)
async function refreshedAuthHeaders() {
    if (window.getAuthToken) {
        try {
            const token = await window.getAuthToken();
            if (token) {
                window._chatAuthToken = token;
                return { 'Authorization': 'Bearer ' + token };
            }
        } catch (e) {
            console.warn('[Chat] Token refresh failed:', e);
        }
    }
    return {};
}

// ============== Query Editor ==============

async function openQueryEditor() {
    const editor = document.getElementById('chat-query-editor');
    if (!editor || !chatState.pendingQuery) return;

    // Hide correction input if visible
    const corrInput = document.getElementById('chat-correction-input');
    if (corrInput) corrInput.style.display = 'none';

    // Toggle editor
    if (editor.style.display !== 'none') {
        editor.style.display = 'none';
        return;
    }

    editor.innerHTML = '<div class="chat-editor-loading">Loading tables...</div>';
    editor.style.display = 'block';

    const catalog = await fetchChatCatalog();
    if (!catalog || !catalog.tables) {
        editor.innerHTML = '<div class="chat-muted">Could not load table catalog.</div>';
        return;
    }

    const p = chatState.pendingQuery;
    renderQueryEditor(editor, p, catalog.tables);
}

function renderQueryEditor(container, parsed, tables) {
    // Table selector
    const tableOptions = tables.map(t => {
        const selected = t.table === parsed.table ? ' selected' : '';
        const label = t.description ? `${t.table} — ${t.description}` : t.table;
        return `<option value="${escapeHtml(t.table)}"${selected}>${escapeHtml(label)}</option>`;
    }).join('');

    // Find current table's columns
    const currentTable = tables.find(t => t.table === parsed.table);
    const columns = currentTable ? currentTable.columns : [];

    // Current aggregations
    const aggHTML = (parsed.aggregations || []).map((a, i) => renderAggRow(a, i, columns)).join('');

    // Current groupBy
    const groupHTML = (parsed.groupBy || []).map((g, i) => renderGroupByRow(g, i, columns)).join('');

    container.innerHTML = `
        <div class="chat-editor-section">
            <label class="chat-editor-label">Table</label>
            <select class="chat-editor-select" id="chat-editor-table" onchange="onEditorTableChange(this.value)">
                ${tableOptions}
            </select>
        </div>
        <div class="chat-editor-section">
            <label class="chat-editor-label">Measures</label>
            <div id="chat-editor-aggs">${aggHTML}</div>
            <button class="chat-btn-link" onclick="addEditorAgg()">+ Add measure</button>
        </div>
        <div class="chat-editor-section">
            <label class="chat-editor-label">Group by</label>
            <div id="chat-editor-groups">${groupHTML}</div>
            <button class="chat-btn-link" onclick="addEditorGroup()">+ Add group</button>
        </div>
        <div class="chat-editor-actions">
            <button class="chat-btn chat-btn-confirm" onclick="applyEditorChanges()">Run query</button>
            <button class="chat-btn chat-btn-reject" onclick="cancelQueryEditor()">Cancel</button>
        </div>`;
}

function renderAggRow(agg, index, columns) {
    const fnOptions = ['sum', 'count', 'avg', 'max', 'min'].map(fn => {
        const sel = (agg.function || 'sum') === fn ? ' selected' : '';
        return `<option value="${fn}"${sel}>${fn.toUpperCase()}</option>`;
    }).join('');

    // Column can be a free-text expression (e.g. "col_a + col_b"), so use input with datalist
    const colListId = `chat-agg-cols-${index}`;
    const colOptions = columns.filter(c => c.numeric).map(c =>
        `<option value="${escapeHtml(c.name)}">`
    ).join('');

    return `<div class="chat-editor-row" data-agg-index="${index}">
        <select class="chat-editor-fn">${fnOptions}</select>
        <input class="chat-editor-col" list="${colListId}" value="${escapeHtml(agg.column || '')}" placeholder="Column or expression">
        <datalist id="${colListId}">${colOptions}</datalist>
        <input class="chat-editor-alias" value="${escapeHtml(agg.alias || '')}" placeholder="Label">
        <button class="chat-btn-link chat-editor-remove" onclick="this.parentElement.remove()">x</button>
    </div>`;
}

function renderGroupByRow(group, index, columns) {
    const allCols = columns.map(c =>
        `<option value="${escapeHtml(c.name)}">`
    ).join('');
    const colListId = `chat-grp-cols-${index}`;

    const fmtOptions = ['', 'month', 'year', 'quarter', 'week', 'date'].map(fmt => {
        const sel = (group.format || '') === fmt ? ' selected' : '';
        const label = fmt || '(none)';
        return `<option value="${fmt}"${sel}>${label}</option>`;
    }).join('');

    return `<div class="chat-editor-row" data-group-index="${index}">
        <input class="chat-editor-col" list="${colListId}" value="${escapeHtml(group.field || '')}" placeholder="Column">
        <datalist id="${colListId}">${allCols}</datalist>
        <select class="chat-editor-fn">${fmtOptions}</select>
        <button class="chat-btn-link chat-editor-remove" onclick="this.parentElement.remove()">x</button>
    </div>`;
}

function addEditorAgg() {
    const container = document.getElementById('chat-editor-aggs');
    if (!container) return;
    const columns = getEditorColumns();
    const index = container.children.length;
    const div = document.createElement('div');
    div.innerHTML = renderAggRow({ column: '', function: 'sum', alias: '' }, index, columns);
    container.appendChild(div.firstElementChild);
}

function addEditorGroup() {
    const container = document.getElementById('chat-editor-groups');
    if (!container) return;
    const columns = getEditorColumns();
    const index = container.children.length;
    const div = document.createElement('div');
    div.innerHTML = renderGroupByRow({ field: '', format: '' }, index, columns);
    container.appendChild(div.firstElementChild);
}

function getEditorColumns() {
    if (!chatState.catalog) return [];
    const tableName = document.getElementById('chat-editor-table')?.value;
    const t = chatState.catalog.tables.find(t => t.table === tableName);
    return t ? t.columns : [];
}

function onEditorTableChange(tableName) {
    if (!chatState.catalog || !chatState.pendingQuery) return;
    // Update pendingQuery table
    chatState.pendingQuery.table = tableName;
    // Re-render aggs and groups with new columns
    const t = chatState.catalog.tables.find(t => t.table === tableName);
    const columns = t ? t.columns : [];

    const aggsContainer = document.getElementById('chat-editor-aggs');
    const groupsContainer = document.getElementById('chat-editor-groups');

    // Keep existing rows but update datalists
    if (aggsContainer) {
        const rows = aggsContainer.querySelectorAll('.chat-editor-row');
        rows.forEach((row, i) => {
            const dl = row.querySelector('datalist');
            if (dl) dl.innerHTML = columns.filter(c => c.numeric).map(c => `<option value="${escapeHtml(c.name)}">`).join('');
        });
    }
    if (groupsContainer) {
        const rows = groupsContainer.querySelectorAll('.chat-editor-row');
        rows.forEach((row, i) => {
            const dl = row.querySelector('datalist');
            if (dl) dl.innerHTML = columns.map(c => `<option value="${escapeHtml(c.name)}">`).join('');
        });
    }
}

function applyEditorChanges() {
    if (!chatState.pendingQuery) return;

    const table = document.getElementById('chat-editor-table')?.value;
    if (table) chatState.pendingQuery.table = table;

    // Read aggregations from editor
    const aggRows = document.querySelectorAll('#chat-editor-aggs .chat-editor-row');
    const aggs = [];
    aggRows.forEach(row => {
        const col = row.querySelector('.chat-editor-col')?.value?.trim();
        const fn = row.querySelector('.chat-editor-fn')?.value || 'sum';
        const alias = row.querySelector('.chat-editor-alias')?.value?.trim() || '';
        if (col) aggs.push({ column: col, function: fn, alias: alias });
    });
    if (aggs.length > 0) chatState.pendingQuery.aggregations = aggs;

    // Read groupBy from editor
    const grpRows = document.querySelectorAll('#chat-editor-groups .chat-editor-row');
    const groups = [];
    grpRows.forEach(row => {
        const field = row.querySelector('.chat-editor-col')?.value?.trim();
        const format = row.querySelector('.chat-editor-fn')?.value || '';
        if (field) groups.push({ field: field, format: format });
    });
    chatState.pendingQuery.groupBy = groups;

    // Close editor and run
    document.getElementById('chat-query-editor').style.display = 'none';
    confirmQuery();
}

// ============== Chart Visualization ==============

function chatResultToChartData(result) {
    const headers = result.headers || [];
    const rows = result.rows || [];
    if (headers.length === 0 || rows.length === 0) return null;

    // Find label column (first non-numeric column based on row data)
    let labelIdx = -1;
    for (let c = 0; c < headers.length; c++) {
        const allNumeric = rows.every(row => {
            const v = row[c];
            return v !== null && v !== undefined && v !== '' && !isNaN(Number(v));
        });
        if (!allNumeric) {
            labelIdx = c;
            break;
        }
    }
    // If all columns are numeric, use first as labels
    if (labelIdx === -1) labelIdx = 0;

    const labels = rows.map(row => String(row[labelIdx] !== null && row[labelIdx] !== undefined ? row[labelIdx] : ''));

    const palette = getChartPalette();
    const colors = palette.blues.concat(palette.accents);

    const datasets = [];
    for (let c = 0; c < headers.length; c++) {
        if (c === labelIdx) continue;
        const vals = rows.map(row => {
            const v = row[c];
            return v !== null && v !== undefined ? Number(v) : 0;
        });
        // Only include columns that are actually numeric
        if (vals.some(v => isNaN(v))) continue;
        datasets.push({
            label: headers[c],
            data: vals,
            color: colors[datasets.length % colors.length]
        });
    }

    if (datasets.length === 0) return null;
    return { labels, datasets };
}

// Time-like header names — matched case-insensitively. Mirrors getSuitableChartTypes.
const CHAT_TIME_HEADERS = ['year', 'month', 'quarter', 'period', 'week', 'date'];

// Apply a default sort to result.rows in place so table and any chart agree.
// Time-grouped results stay chronological ascending; categorical results sort
// descending by the first all-numeric column ("biggest at top").
function applyChatDefaultSort(result) {
    const headers = result && result.headers;
    const rows = result && result.rows;
    if (!Array.isArray(headers) || !Array.isArray(rows) || rows.length < 2) return;

    const timeIdx = headers.findIndex(h => CHAT_TIME_HEADERS.includes(String(h).toLowerCase()));
    if (timeIdx !== -1) {
        rows.sort((a, b) => {
            const av = a[timeIdx], bv = b[timeIdx];
            if (av === bv) return 0;
            if (av === null || av === undefined) return 1;
            if (bv === null || bv === undefined) return -1;
            return String(av).localeCompare(String(bv));
        });
        return;
    }

    let numericIdx = -1;
    for (let c = 0; c < headers.length; c++) {
        const allNumeric = rows.every(row => {
            const v = row[c];
            return v !== null && v !== undefined && v !== '' && !isNaN(Number(v));
        });
        if (allNumeric) { numericIdx = c; break; }
    }
    if (numericIdx === -1) return;
    rows.sort((a, b) => Number(b[numericIdx]) - Number(a[numericIdx]));
}

function exportChatResultCSV(msgIndex) {
    const msg = chatState.messages[msgIndex];
    if (!msg || msg.type !== 'result') return;
    const result = msg.content || {};
    const headers = result.headers || [];
    const rows = result.rows || [];
    if (rows.length === 0) return;
    const ts = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-');
    exportTableToCSV(headers, rows, `ask-result-${ts}.csv`);
    trackCSVDownload('chat:ask', 'Ask result', rows.length);
    window.telemetry?.track('ask.csv_download', { rows: String(rows.length) });
}

function getSuitableChartTypes(result) {
    const headers = result.headers || [];
    const rows = result.rows || [];
    if (rows.length === 0) return [];

    const chartData = chatResultToChartData(result);
    if (!chartData || chartData.datasets.length === 0) return [];

    const types = [];
    // Bar: always available
    types.push('bar');

    // Line: if time-like labels or more than 4 rows
    const timeHeaders = ['year', 'month', 'quarter', 'period', 'week', 'date'];
    const hasTimeColumn = headers.some(h => timeHeaders.includes(h.toLowerCase()));
    if (hasTimeColumn || rows.length > 4) {
        types.push('line');
    }

    // Pie: single numeric column and <= 15 categories
    if (chartData.datasets.length === 1 && rows.length <= 15) {
        types.push('pie');
    }

    return types;
}

function destroyChatCharts() {
    for (const id in chatChartInstances) {
        if (chatChartInstances[id]) {
            chatChartInstances[id].destroy();
        }
        delete chatChartInstances[id];
    }
}

function renderChatChart(type, msgIndex) {
    const msg = chatState.messages[msgIndex];
    if (!msg || msg.type !== 'result') return;

    const container = document.getElementById('chat-chart-' + msgIndex);
    if (!container) return;

    // Destroy previous chart in this container
    const prevChartId = container.dataset.chartId;
    if (prevChartId && chatChartInstances[prevChartId]) {
        chatChartInstances[prevChartId].destroy();
        delete chatChartInstances[prevChartId];
    }
    container.innerHTML = '';

    const chartData = chatResultToChartData(msg.content);
    if (!chartData) return;

    // Build component object that renderBarChart/renderLineChart/renderPieChart expect
    const component = {
        title: '',
        data: chartData
    };

    // Render using existing functions from components.js
    if (type === 'bar') {
        renderBarChart(container, component);
    } else if (type === 'line') {
        renderLineChart(container, component);
    } else if (type === 'pie') {
        renderPieChart(container, component);
    }

    // Track chart instance for cleanup — the renderers use rAF so we find the canvas after
    requestAnimationFrame(() => {
        const canvas = container.querySelector('canvas');
        if (canvas && canvas.id && chartInstances[canvas.id]) {
            container.dataset.chartId = canvas.id;
            chatChartInstances[canvas.id] = chartInstances[canvas.id];
        }
    });
}

// ============== Map Visualization ==============

// Track chat map instances for cleanup
const chatMapInstances = {};

function renderChatMap(msgIndex) {
    const msg = chatState.messages[msgIndex];
    if (!msg || msg.type !== 'result' || !msg.content.mapData) return;

    const container = document.getElementById('chat-chart-' + msgIndex);
    if (!container) return;

    // Destroy previous chart in this container
    const prevChartId = container.dataset.chartId;
    if (prevChartId && chatChartInstances[prevChartId]) {
        chatChartInstances[prevChartId].destroy();
        delete chatChartInstances[prevChartId];
    }
    // Destroy previous map in this container
    const prevMapId = container.dataset.mapId;
    if (prevMapId && chatMapInstances[prevMapId]) {
        chatMapInstances[prevMapId].remove();
        delete chatMapInstances[prevMapId];
    }
    container.innerHTML = '';

    const mapData = msg.content.mapData;

    // Normalize district names: if GeoJSON has "Kampala District" but data has "Kampala",
    // we need to match them. Fetch GeoJSON first, build a lookup, then remap.
    fetchGeoJSON(mapData.geojson_path).then(geojson => {
        const geoNames = geojson.features.map(f => f.properties.name);
        const normalized = normalizeChatMapValues(mapData.district_values, geoNames);

        const component = {
            title: '',
            data: {
                district_values: normalized,
                geojson_path: mapData.geojson_path,
                color_scheme: mapData.color_scheme,
                legend_title: mapData.legend_title
            }
        };

        renderChoropleth(container, component);

        // Track map instance for cleanup
        setTimeout(() => {
            const mapDiv = container.querySelector('.choropleth-map-area');
            if (mapDiv && mapDiv.id && window.choroplethMaps && window.choroplethMaps[mapDiv.id]) {
                container.dataset.mapId = mapDiv.id;
                chatMapInstances[mapDiv.id] = window.choroplethMaps[mapDiv.id];
            }
        }, 200);
    });
}

// Normalize data keys to match GeoJSON feature names.
// Handles cases like "Kampala" → "Kampala District", or exact matches.
function normalizeChatMapValues(districtValues, geoNames) {
    const result = {};
    // Build lowercase lookup: "kampala district" → "Kampala District"
    const geoLookup = {};
    for (const name of geoNames) {
        geoLookup[name.toLowerCase()] = name;
    }

    for (const [key, val] of Object.entries(districtValues)) {
        const lower = key.toLowerCase();
        // Try exact match first
        if (geoLookup[lower]) {
            result[geoLookup[lower]] = val;
            continue;
        }
        // Try appending " district"
        if (geoLookup[lower + ' district']) {
            result[geoLookup[lower + ' district']] = val;
            continue;
        }
        // Try appending " city"
        if (geoLookup[lower + ' city']) {
            result[geoLookup[lower + ' city']] = val;
            continue;
        }
        // Try stripping " district" or " city" suffix from key
        const stripped = lower.replace(/ (district|city)$/, '');
        if (stripped !== lower && geoLookup[stripped]) {
            result[geoLookup[stripped]] = val;
            continue;
        }
        // Partial match: find GeoJSON name that starts with the key
        const match = geoNames.find(n => n.toLowerCase().startsWith(lower + ' '));
        if (match) {
            result[match] = val;
            continue;
        }
        // Keep original (won't match but preserves data)
        result[key] = val;
    }
    return result;
}

function destroyChatMaps() {
    for (const id in chatMapInstances) {
        if (chatMapInstances[id]) {
            chatMapInstances[id].remove();
        }
        delete chatMapInstances[id];
    }
}

// ============== Initialization ==============

async function initChat() {
    // Check if Ask module is enabled via server config
    try {
        const basePath = window.BASE_PATH || '';
        const resp = await fetch(`${basePath}/api/frontend/config`);
        if (resp.ok) {
            const config = await resp.json();
            if (!config.askEnabled) {
                // Hide footer nav and chat container entirely
                const footer = document.querySelector('.app-footer');
                if (footer) footer.style.display = 'none';
                const chatContainer = document.getElementById('chat-container');
                if (chatContainer) chatContainer.style.display = 'none';
                return;
            }
        }
    } catch (e) {
        // If config fetch fails, hide Ask by default
        const footer = document.querySelector('.app-footer');
        if (footer) footer.style.display = 'none';
        return;
    }

    // Set up input handling
    const input = document.getElementById('chat-input');
    const sendBtn = document.getElementById('chat-send-btn');

    if (input) {
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                submitQuestion();
            }
        });
    }

    if (sendBtn) {
        sendBtn.addEventListener('click', submitQuestion);
    }

    // Set up nav pills
    document.querySelectorAll('.nav-pill').forEach(pill => {
        pill.addEventListener('click', () => {
            if (pill.dataset.view === 'chat') {
                showChatView();
            } else {
                showHomeView();
            }
        });
    });

    updateNavPills();
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initChat);
} else {
    initChat();
}
