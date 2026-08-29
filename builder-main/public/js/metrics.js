// Metrics Dashboard JavaScript
// Fetches and displays server metrics using Chart.js and tabbed segments

const API_BASE = (window.BASE_PATH || '') + '/api';
let autoRefreshInterval = null;

// Global chart instances registry to avoid overlapping canvas contexts
window.myCharts = window.myCharts || {};

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    loadMetrics();
    setupAutoRefresh();
    setupRefreshButton();
    setupFeedbackPanel();
    loadFeedbackPanel(true);
    setupSearchFilters();

    // Pause auto-refresh when tab is hidden to save memory and network
    document.addEventListener('visibilitychange', () => {
        const checkbox = document.getElementById('auto-refresh');
        if (document.hidden) {
            stopAutoRefresh();
        } else if (checkbox && checkbox.checked) {
            startAutoRefresh();
        }
    });
});

// Setup refresh button
function setupRefreshButton() {
    const btn = document.getElementById('refresh-btn');
    if (!btn) return;
    btn.addEventListener('click', () => {
        if (!btn.classList.contains('loading')) {
            loadMetrics();
            loadFeedbackPanel(true);
        }
    });
}

// Setup search inputs for report statistics and user tables
function setupSearchFilters() {
    const reportSearch = document.getElementById('report-search');
    if (reportSearch) {
        reportSearch.addEventListener('input', () => {
            if (window.lastReportStats) {
                renderReportStats(window.lastReportStats, reportSearch.value.trim().toLowerCase());
            }
        });
    }

    const userSearch = document.getElementById('user-search');
    if (userSearch) {
        userSearch.addEventListener('input', () => {
            if (window.lastUserStats) {
                renderUserStats(window.lastUserStats, userSearch.value.trim().toLowerCase());
            }
        });
    }
}

// Setup auto-refresh
function setupAutoRefresh() {
    const checkbox = document.getElementById('auto-refresh');
    if (!checkbox) return;

    checkbox.addEventListener('change', () => {
        if (checkbox.checked) {
            startAutoRefresh();
        } else {
            stopAutoRefresh();
        }
    });
}

function startAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
    }
    autoRefreshInterval = setInterval(() => {
        loadMetrics();
        loadFeedbackPanel(false);
    }, 30000); // 30 seconds
}

function stopAutoRefresh() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
}

// Load metrics from API
async function loadMetrics() {
    const btn = document.getElementById('refresh-btn');
    if (btn) {
        btn.classList.add('loading');
        btn.querySelector('span').textContent = 'Loading...';
        btn.querySelector('.refresh-icon').style.animation = 'spin 1s linear infinite';
    }

    try {
        // Wait for auth to initialise so the token is available before we fetch
        if (window.authReady) {
            try { await window.authReady; } catch (e) {}
        }

        const headers = {};
        if (typeof window.getAuthToken === 'function') {
            try {
                const token = await window.getAuthToken();
                if (token) headers['Authorization'] = 'Bearer ' + token;
            } catch (e) {}
        }

        const response = await fetch(`${API_BASE}/metrics`, { headers });
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        const data = await response.json();
        renderMetrics(data);
    } catch (error) {
        console.error('Failed to load metrics:', error);
        showError('Failed to load metrics. Is the server running?');
    } finally {
        if (btn) {
            btn.classList.remove('loading');
            btn.querySelector('span').textContent = 'Refresh';
            btn.querySelector('.refresh-icon').style.animation = 'none';
        }
    }
}

// Render metrics data
function renderMetrics(data) {
    // Update server info
    const serverStart = new Date(data.serverStartTime);
    document.getElementById('server-info').textContent =
        `Server started: ${formatDateTime(serverStart)}`;

    // Update Uptime
    document.getElementById('uptime').textContent = formatUptime(serverStart);

    // Update Total Requests & Concurrent Activity
    document.getElementById('total-requests').textContent = formatNumber(data.totalRequests);
    document.getElementById('concurrent-requests').textContent = `${data.activeRequests || 0} active requests (Peak: ${data.peakConcurrent || 0})`;

    // Update Error Rate Card
    const errorRate = data.errorRate || 0;
    const errorCard = document.getElementById('error-rate-card');
    const errorVal = document.getElementById('global-error-rate');
    errorVal.textContent = `${errorRate.toFixed(2)}%`;
    document.getElementById('total-errors').textContent = `${formatNumber(data.totalErrors || 0)} total errors`;

    // Apply color class to Error Rate box based on severity
    if (errorCard) {
        errorCard.className = 'kpi-card';
        if (errorRate > 5.0) {
            errorCard.classList.add('error');
        } else if (errorRate > 1.0) {
            errorCard.classList.add('warning');
        } else {
            errorCard.classList.add('success');
        }
    }

    // System Health Status indicator
    const healthIndicator = document.getElementById('system-health-indicator');
    const healthText = document.getElementById('system-health-text');
    const globalStatusDot = document.getElementById('global-status-dot');
    
    let systemStatus = 'success';
    let statusMessage = 'All Systems Operational';

    if (errorRate > 5.0) {
        systemStatus = 'error';
        statusMessage = 'System Disruption Detected';
    } else if (errorRate > 1.0) {
        systemStatus = 'warning';
        statusMessage = 'Degraded System Performance';
    }

    if (healthIndicator && healthText) {
        healthIndicator.className = `system-status-indicator ${systemStatus}`;
        healthText.textContent = `${statusMessage} (${errorRate.toFixed(2)}% err)`;
    }
    if (globalStatusDot) {
        globalStatusDot.style.backgroundColor = systemStatus === 'error' ? 'var(--color-error)' : (systemStatus === 'warning' ? 'var(--color-warning)' : 'var(--color-success)');
    }

    // Update cache stats
    if (data.cache) {
        document.getElementById('cache-entries').textContent =
            `${formatNumber(data.cache.totalEntries)} cache entries`;
        const totalLookups = (data.cache.hits || 0) + (data.cache.misses || 0);
        document.getElementById('cache-hit-rate').textContent =
            totalLookups > 0 ? `${data.cache.hitRate.toFixed(1)}%`
            : data.cache.totalEntries > 0 ? '0.0%' : '--';
    }

    // Update DBPool open connections indicator
    if (data.dbPool) {
        const dbOpenVal = document.getElementById('db-open-conns');
        if (dbOpenVal) {
            dbOpenVal.textContent = formatNumber(data.dbPool.openConnections);
        }
    }

    // Store stats for client-side search filtering
    window.lastReportStats = data.reportStats || [];
    window.lastUserStats = data.userStats || [];

    // Render components
    renderDBPoolStats(data.dbPool || {});
    renderReportStats(data.reportStats || []);
    renderDurationBuckets(data.durationBuckets || {});
    renderDailyHits(data.dailyHits || {});
    renderReportRankings(data);

    // Render modern charts
    renderAdoptionChartJS(data.dateHits || {});
    renderDBPoolChartJS(data.dbPool || {});
    renderUIEventChartJS(data.uiStats || {});

    // Update PDF stats
    if (data.pdfStats) {
        document.getElementById('pdf-downloads').textContent =
            formatNumber(data.pdfStats.totalDownloads);
        document.getElementById('pdf-errors').textContent =
            formatNumber(data.pdfStats.totalErrors);
        document.getElementById('pdf-avg-time').textContent =
            data.pdfStats.totalDownloads > 0 ? formatDuration(data.pdfStats.avgDurationMs) : '--';
        renderPDFByReport(data.pdfStats.byReport || {});
    }

    // Update CSV stats
    if (data.csvStats) {
        document.getElementById('csv-downloads').textContent =
            formatNumber(data.csvStats.totalDownloads);
        document.getElementById('csv-rows').textContent =
            formatNumber(data.csvStats.totalRows);
        renderCSVByReport(data.csvStats.byReport || {});
    }

    // Render user stats (hidden when no data / auth off)
    renderUserStats(data.userStats || []);

    // Render chat stats
    renderChatStats(data.chatStats || {});

    // Render component health stats
    renderComponentStats(data.componentStats || {});

    // Render query performance stats
    renderQueryStats(data.queryStats || {});

    // Render UI event log list
    renderUIEventLog(data.uiStats || {});
}

// Render database pool statistics table
function renderDBPoolStats(pool) {
    const tbody = document.getElementById('db-pool-body');
    if (!tbody) return;

    if (!pool || pool.openConnections === undefined) {
        tbody.innerHTML = `
            <tr>
                <td colspan="3" class="empty-state">No database pool data available.</td>
            </tr>
        `;
        return;
    }

    const maxOpen = pool.maxOpenConnections || 50;
    const inUsePercent = pool.openConnections > 0 ? (pool.inUse / maxOpen * 100) : 0;
    const hasWaits = pool.waitCount > 0;

    const metrics = [
        {
            name: 'Open Connections',
            value: `${pool.openConnections} / ${maxOpen}`,
            desc: 'Current open vs max pool size',
            status: pool.openConnections >= Math.floor(maxOpen * 0.8) ? 'warning' : 'success'
        },
        {
            name: 'In Use Connections',
            value: pool.inUse,
            desc: 'Connections actively executing queries',
            status: inUsePercent >= 80 ? 'warning' : 'info'
        },
        {
            name: 'Idle Connections',
            value: pool.idle,
            desc: 'Connections waiting in pool',
            status: ''
        },
        {
            name: 'Request Wait Count',
            value: pool.waitCount,
            desc: hasWaits ? 'Queries had to wait for a connection - consider scaling pool' : 'No queries had to wait (optimal)',
            status: hasWaits ? 'warning' : 'success'
        },
        {
            name: 'Total Wait Duration',
            value: formatDuration(pool.waitDurationMs),
            desc: 'Time spent waiting for connections',
            isFormatted: true,
            status: pool.waitDurationMs > 1000 ? 'warning' : ''
        },
        {
            name: 'Closed (Idle Limit)',
            value: pool.maxIdleClosed,
            desc: 'Connections closed due to max idle settings',
            status: ''
        },
        {
            name: 'Closed (Max Lifetime)',
            value: pool.maxLifetimeClosed,
            desc: 'Connections recycled automatically',
            status: ''
        }
    ];

    tbody.innerHTML = metrics.map(m => {
        const badgeClass = m.status ? ` status-badge ${m.status}` : '';
        const badgeHtml = m.status ? `<span class="${badgeClass}">${escapeHtml(m.name)}</span>` : `<strong>${escapeHtml(m.name)}</strong>`;
        return `
            <tr>
                <td>${badgeHtml}</td>
                <td class="number"><strong>${m.isFormatted ? m.value : formatNumber(m.value)}</strong></td>
                <td class="desc" style="color: var(--caption); font-size: 12.5px;">${escapeHtml(m.desc)}</td>
            </tr>
        `;
    }).join('');
}

// Group reports by category (first path segment)
function groupByCategory(stats) {
    const groups = {};
    for (const stat of stats) {
        const category = stat.reportId.split('/')[0] || 'uncategorized';
        if (!groups[category]) {
            groups[category] = [];
        }
        groups[category].push(stat);
    }
    return groups;
}

// Track expanded categories in UI
const expandedCategories = new Set();

// Toggle category expand/collapse
function toggleCategory(category) {
    if (expandedCategories.has(category)) {
        expandedCategories.delete(category);
    } else {
        expandedCategories.add(category);
    }
    if (window.lastReportStats) {
        const reportSearch = document.getElementById('report-search');
        renderReportStats(window.lastReportStats, reportSearch ? reportSearch.value.trim().toLowerCase() : '');
    }
}
window.toggleCategory = toggleCategory;

// Render report statistics table with category grouping and optional search filter
function renderReportStats(stats, searchQuery = '') {
    const tbody = document.getElementById('report-stats-body');
    const badge = document.getElementById('report-count-badge');
    if (!tbody) return;

    if (!stats || stats.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="empty-state">
                    No report requests yet. Visit some reports to see metrics.
                </td>
            </tr>
        `;
        return;
    }

    // Update count badge
    if (badge) {
        badge.textContent = `${stats.length} reports`;
    }

    // Filter reports if search query is provided
    let filteredStats = stats;
    if (searchQuery) {
        filteredStats = stats.filter(s => s.reportId.toLowerCase().includes(searchQuery));
    }

    if (filteredStats.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="empty-state">No reports match your search query "${escapeHtml(searchQuery)}".</td>
            </tr>
        `;
        return;
    }

    // Group by category
    const groups = groupByCategory(filteredStats);
    const categoryNames = Object.keys(groups).sort();

    let html = '';
    for (const category of categoryNames) {
        const reports = groups[category];
        reports.sort((a, b) => b.requestCount - a.requestCount);

        // Calculate aggregates
        const totalRequests = reports.reduce((sum, r) => sum + r.requestCount, 0);
        const totalErrors = reports.reduce((sum, r) => sum + r.errorCount, 0);
        const avgDuration = reports.reduce((sum, r) => sum + r.avgDurationMs * r.requestCount, 0) / (totalRequests || 1);
        const maxDuration = Math.max(...reports.map(r => r.maxDurationMs));
        const lastAccessed = new Date(Math.max(...reports.map(r => new Date(r.lastAccessed).getTime())));

        // Auto-expand category if searching, otherwise check if expanded
        const isExpanded = searchQuery ? true : expandedCategories.has(category);
        const toggleClass = isExpanded ? '' : 'collapsed';

        html += `
            <tr class="category-row" onclick="toggleCategory('${escapeHtml(category)}')">
                <td>
                    <span class="category-toggle ${toggleClass}"></span>
                    <span class="category-name">${escapeHtml(category)}</span>
                    <span style="color: var(--caption); font-weight: normal; font-size: 11.5px;"> (${reports.length} reports)</span>
                </td>
                <td class="number"><strong>${formatNumber(totalRequests)}</strong></td>
                <td class="number ${totalErrors > 0 ? 'error-count' : ''}" style="${totalErrors > 0 ? 'color: var(--color-error); font-weight: bold;' : ''}">${formatNumber(totalErrors)}</td>
                <td class="number duration">${formatDuration(avgDuration)}</td>
                <td class="number duration">${formatDuration(maxDuration)}</td>
                <td class="time-ago">${formatTimeAgo(lastAccessed)}</td>
            </tr>
        `;

        // Render report rows
        for (const stat of reports) {
            const hiddenClass = isExpanded ? '' : 'hidden';
            html += `
                <tr class="report-row ${hiddenClass}" data-category="${escapeHtml(category)}">
                    <td class="report-id">${escapeHtml(stat.reportId)}</td>
                    <td class="number">${formatNumber(stat.requestCount)}</td>
                    <td class="number" style="${stat.errorCount > 0 ? 'color: var(--color-error); font-weight: 500;' : ''}">${formatNumber(stat.errorCount)}</td>
                    <td class="number duration" style="color: var(--caption);">${formatDuration(stat.avgDurationMs)}</td>
                    <td class="number duration" style="color: var(--caption);">${formatDuration(stat.maxDurationMs)}</td>
                    <td class="time-ago">${formatTimeAgo(new Date(stat.lastAccessed))}</td>
                </tr>
            `;
        }
    }

    tbody.innerHTML = html;
}

// Render report rankings (Top 10 vs Least Requested)
function renderReportRankings(data) {
    const topBody = document.getElementById('top-reports-body');
    const leastBody = document.getElementById('least-reports-body');
    if (!topBody || !leastBody) return;

    const stats = data.reportStats || [];
    let top = data.topReports || [];
    let least = data.leastRequestedReports || [];

    if (stats.length > 0 && top.length === 0 && least.length === 0) {
        const byDesc = [...stats].sort((a, b) => b.requestCount - a.requestCount || String(a.reportId).localeCompare(String(b.reportId)));
        top = byDesc.slice(0, 10);
        
        const byAsc = [...stats].sort((a, b) => a.requestCount - b.requestCount || String(a.reportId).localeCompare(String(b.reportId)));
        least = byAsc.slice(0, 10);
    }

    const rowTop = (stat, i) => `
        <tr>
            <td class="number" style="color: var(--caption);">${i + 1}</td>
            <td class="report-id">${escapeHtml(stat.reportId)}</td>
            <td class="number"><strong>${formatNumber(stat.requestCount)}</strong></td>
            <td class="number" style="${stat.errorCount > 0 ? 'color: var(--color-error); font-weight: bold;' : 'color: var(--caption);'}">${formatNumber(stat.errorCount)}</td>
        </tr>
    `;

    const emptyRow = (msg) => `
        <tr>
            <td colspan="4" class="empty-state">${escapeHtml(msg)}</td>
        </tr>
    `;

    if (stats.length === 0) {
        topBody.innerHTML = emptyRow('No data.');
        leastBody.innerHTML = emptyRow('No data.');
        return;
    }

    topBody.innerHTML = top.length ? top.map((s, i) => rowTop(s, i)).join('') : emptyRow('No data.');
    leastBody.innerHTML = least.length ? least.map((s, i) => rowTop(s, i)).join('') : emptyRow('No data.');
}

// Render response speed buckets table
function renderDurationBuckets(buckets) {
    const tbody = document.getElementById('duration-buckets-body');
    if (!tbody) return;

    const orderedLabels = ['<100ms', '100-500ms', '500ms-1s', '1-3s', '3-10s', '>10s'];
    const total = orderedLabels.reduce((sum, label) => sum + (buckets[label] || 0), 0);

    if (total === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="3" class="empty-state">No data.</td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = orderedLabels.map(label => {
        const count = buckets[label] || 0;
        const pct = total > 0 ? (count / total * 100).toFixed(1) : '0.0';
        const barWidth = total > 0 ? Math.round(count / total * 100) : 0;
        return `
            <tr>
                <td><code>${escapeHtml(label)}</code></td>
                <td class="number"><strong>${formatNumber(count)}</strong></td>
                <td class="number">${pct}%</td>
            </tr>
        `;
    }).join('');
}

// Render daily hits (requests by weekday)
function renderDailyHits(daily) {
    const tbody = document.getElementById('daily-hits-body');
    if (!tbody) return;

    const weekdayOrder = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    const days = weekdayOrder.map(day => ({
        day,
        count: daily[day] || 0
    }));

    const maxCount = Math.max(...days.map(d => d.count), 1);
    const total = days.reduce((sum, d) => sum + d.count, 0);

    if (total === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="3" class="empty-state">No data.</td>
            </tr>
        `;
        return;
    }

    const busiestDay = days.reduce((max, d) => d.count > max.count ? d : max, days[0]).day;

    tbody.innerHTML = days.map(d => {
        const barWidth = Math.round(d.count / maxCount * 100);
        const isBusiest = d.day === busiestDay && d.count > 0;
        return `
            <tr class="${isBusiest ? 'busiest-day' : ''}">
                <td><strong>${d.day.slice(0, 3)}</strong></td>
                <td class="number"><strong>${formatNumber(d.count)}</strong></td>
                <td><span class="distribution-bar" style="width: ${barWidth}px;"></span></td>
            </tr>
        `;
    }).join('');
}

// Render cumulative and daily requests over time using Chart.js
function renderAdoptionChartJS(dateHits) {
    const ctx = document.getElementById('adoptionChartJS');
    if (!ctx) return;

    const dates = Object.keys(dateHits).sort();
    if (dates.length === 0) {
        return;
    }

    let cumulative = 0;
    const dailyData = [];
    const cumulativeData = [];

    dates.forEach(date => {
        const dailyHits = dateHits[date];
        cumulative += dailyHits;
        dailyData.push(dailyHits);
        cumulativeData.push(cumulative);
    });

    const formattedDates = dates.map(d => {
        const parts = d.split('-');
        if (parts.length === 3) {
            const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
            const mIdx = parseInt(parts[1], 10) - 1;
            return `${months[mIdx]} ${parseInt(parts[2], 10)}`;
        }
        return d;
    });

    if (window.myCharts.adoption) {
        window.myCharts.adoption.destroy();
    }

    const accentColor = getComputedStyle(document.documentElement).getPropertyValue('--accent-color').trim() || '#14b8a6';
    const gridColor = getComputedStyle(document.documentElement).getPropertyValue('--border-color').trim() || 'rgba(255, 255, 255, 0.08)';
    const captionColor = getComputedStyle(document.documentElement).getPropertyValue('--caption').trim() || '#94a3b8';

    window.myCharts.adoption = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: formattedDates,
            datasets: [
                {
                    label: 'Daily Requests',
                    data: dailyData,
                    backgroundColor: 'rgba(20, 184, 166, 0.25)',
                    borderColor: accentColor,
                    borderWidth: 1,
                    yAxisID: 'yDaily',
                    order: 2
                },
                {
                    label: 'Total Cumulative',
                    data: cumulativeData,
                    type: 'line',
                    borderColor: accentColor,
                    backgroundColor: 'rgba(20, 184, 166, 0.04)',
                    borderWidth: 2.5,
                    fill: true,
                    tension: 0.2,
                    pointBackgroundColor: accentColor,
                    pointRadius: 3,
                    pointHoverRadius: 6,
                    yAxisID: 'yCumulative',
                    order: 1
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false
            },
            plugins: {
                legend: {
                    position: 'top',
                    labels: {
                        color: captionColor,
                        font: {
                            family: 'Inter',
                            size: 11
                        }
                    }
                },
                tooltip: {
                    padding: 10,
                    bodyFont: { family: 'Inter' },
                    titleFont: { family: 'Outfit', weight: 'bold' }
                }
            },
            scales: {
                x: {
                    grid: { display: false },
                    ticks: {
                        color: captionColor,
                        font: { family: 'Inter', size: 10 },
                        maxTicksLimit: 8
                    }
                },
                yDaily: {
                    type: 'linear',
                    position: 'right',
                    title: {
                        display: true,
                        text: 'Daily Volume',
                        color: captionColor,
                        font: { family: 'Outfit', size: 11, weight: 'bold' }
                    },
                    grid: { display: false },
                    ticks: {
                        color: captionColor,
                        font: { family: 'Inter', size: 10 },
                        precision: 0
                    }
                },
                yCumulative: {
                    type: 'linear',
                    position: 'left',
                    title: {
                        display: true,
                        text: 'Total Requests',
                        color: captionColor,
                        font: { family: 'Outfit', size: 11, weight: 'bold' }
                    },
                    grid: { color: gridColor },
                    ticks: {
                        color: captionColor,
                        font: { family: 'Inter', size: 10 },
                        precision: 0
                    }
                }
            }
        }
    });
}

// Render Database connection pool doughnut chart
function renderDBPoolChartJS(pool) {
    const ctx = document.getElementById('dbPoolChartJS');
    if (!ctx || !pool || pool.openConnections === undefined) return;

    const maxConns = pool.maxOpenConnections || 50;
    const inUse = pool.inUse || 0;
    const idle = pool.idle || 0;
    const available = Math.max(0, maxConns - inUse - idle);

    if (window.myCharts.dbPool) {
        window.myCharts.dbPool.destroy();
    }

    const accentColor = getComputedStyle(document.documentElement).getPropertyValue('--accent-color').trim() || '#14b8a6';
    const cardBgColor = getComputedStyle(document.documentElement).getPropertyValue('--color-card-bg').trim() || '#1e293b';
    const bgTertiary = getComputedStyle(document.documentElement).getPropertyValue('--bg-tertiary').trim() || '#334155';

    window.myCharts.dbPool = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['In Use', 'Idle', 'Available Capacity'],
            datasets: [{
                data: [inUse, idle, available],
                backgroundColor: [
                    accentColor, // Teal for active use
                    '#f59e0b',   // Amber for idle
                    bgTertiary   // Available capacity matching theme card background/tertiary
                ],
                borderWidth: 1,
                borderColor: cardBgColor
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            return ` ${context.label}: ${context.raw} conns`;
                        }
                    }
                }
            },
            cutout: '70%'
        }
    });
}

// Render UI Event distribution doughnut chart
function renderUIEventChartJS(uiStats) {
    const ctx = document.getElementById('uiEventChartJS');
    if (!ctx || !uiStats || !uiStats.countsByName) return;

    const counts = uiStats.countsByName;
    const labels = Object.keys(counts);
    const data = Object.values(counts);

    if (labels.length === 0) {
        ctx.parentNode.innerHTML = '<div class="empty-state">No UI telemetry tracked yet.</div>';
        return;
    }

    if (window.myCharts.uiEvents) {
        window.myCharts.uiEvents.destroy();
    }

    const captionColor = getComputedStyle(document.documentElement).getPropertyValue('--caption').trim() || '#94a3b8';
    const cardBgColor = getComputedStyle(document.documentElement).getPropertyValue('--color-card-bg').trim() || '#1e293b';

    const palette = [
        '#14b8a6', '#10b981', '#3b82f6', '#8b5cf6', 
        '#f59e0b', '#ef4444', '#6366f1', '#ec4899'
    ];

    window.myCharts.uiEvents = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: labels.map(l => l.replace('.', ': ')),
            datasets: [{
                data: data,
                backgroundColor: palette.slice(0, labels.length),
                borderWidth: 1.5,
                borderColor: cardBgColor
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        boxWidth: 8,
                        padding: 10,
                        color: captionColor,
                        font: { family: 'Inter', size: 10 }
                    }
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const val = context.raw;
                            const total = context.dataset.data.reduce((a,b)=>a+b, 0);
                            const pct = total > 0 ? ((val/total)*100).toFixed(1) : 0;
                            return ` ${context.label}: ${val} (${pct}%)`;
                        }
                    }
                }
            },
            cutout: '55%'
        }
    });
}

// Render PDF downloads by report
function renderPDFByReport(byReport) {
    const tbody = document.getElementById('pdf-by-report-body');
    const badge = document.getElementById('pdf-report-count');
    if (!tbody) return;

    const entries = Object.entries(byReport)
        .map(([reportId, count]) => ({ reportId, count }))
        .sort((a, b) => b.count - a.count);

    if (badge) badge.textContent = entries.length;

    if (entries.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="2" class="empty-state">No PDF downloads yet.</td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = entries.map(entry => `
        <tr>
            <td class="report-id">${escapeHtml(entry.reportId)}</td>
            <td class="number"><strong>${formatNumber(entry.count)}</strong></td>
        </tr>
    `).join('');
}

// Render CSV downloads by report
function renderCSVByReport(byReport) {
    const tbody = document.getElementById('csv-by-report-body');
    const badge = document.getElementById('csv-report-count');
    if (!tbody) return;

    const entries = Object.entries(byReport)
        .map(([reportId, count]) => ({ reportId, count }))
        .sort((a, b) => b.count - a.count);

    if (badge) badge.textContent = entries.length;

    if (entries.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="2" class="empty-state">No CSV downloads yet.</td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = entries.map(entry => `
        <tr>
            <td class="report-id">${escapeHtml(entry.reportId)}</td>
            <td class="number"><strong>${formatNumber(entry.count)}</strong></td>
        </tr>
    `).join('');
}

// Render Active users table
function renderUserStats(userStats, searchQuery = '') {
    const section = document.getElementById('user-stats-section');
    const badge = document.getElementById('user-stats-count');
    const tbody = document.getElementById('user-stats-body');
    if (!tbody) return;

    if (!userStats || userStats.length === 0) {
        if (section) section.style.display = 'none';
        return;
    }

    if (section) section.style.display = 'block';
    if (badge) badge.textContent = `${userStats.length} active`;

    let filtered = userStats;
    if (searchQuery) {
        filtered = userStats.filter(u => 
            u.username.toLowerCase().includes(searchQuery) || 
            (u.email && u.email.toLowerCase().includes(searchQuery))
        );
    }

    if (filtered.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="5" class="empty-state">No users match "${escapeHtml(searchQuery)}"</td>
            </tr>
        `;
        return;
    }

    const TOP_N = 25;
    const sorted = [...filtered].sort((a, b) => (b.requestCount || 0) - (a.requestCount || 0));
    const shown = sorted.slice(0, TOP_N);

    tbody.innerHTML = shown.map(user => {
        const topEntries = Object.entries(user.topReports || {})
            .sort((a, b) => b[1] - a[1])
            .slice(0, 3);
        const topChips = topEntries.length > 0
            ? topEntries.map(([id, count]) =>
                `<span class="chip">${escapeHtml(id)} (${count})</span>`
            ).join('')
            : '<span class="chip">None</span>';

        return `
            <tr>
                <td><strong>${escapeHtml(user.username)}</strong></td>
                <td>${escapeHtml(user.email || 'N/A')}</td>
                <td class="number"><strong>${formatNumber(user.requestCount)}</strong></td>
                <td class="time-ago">${formatTimeAgo(new Date(user.lastSeen))}</td>
                <td><div class="chips-container">${topChips}</div></td>
            </tr>
        `;
    }).join('');

    if (filtered.length > TOP_N) {
        tbody.innerHTML += `
            <tr>
                <td colspan="5" class="empty-state" style="padding: 10px;">
                    Showing top ${TOP_N} of ${formatNumber(filtered.length)} users.
                </td>
            </tr>
        `;
    }
}

// Render Ask Module (AI Chat) stats
function renderChatStats(chat) {
    const section = document.getElementById('chat-stats-section');
    const totalActivity = (chat.totalQuestions || 0) + (chat.totalExecutions || 0);
    if (!section) return;

    if (totalActivity === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = 'block';
    const badge = document.getElementById('chat-stats-count');
    if (badge) badge.textContent = `${totalActivity} actions`;

    document.getElementById('chat-questions').textContent = formatNumber(chat.totalQuestions || 0);
    document.getElementById('chat-executions').textContent = formatNumber(chat.totalExecutions || 0);
    document.getElementById('chat-errors').textContent = formatNumber(chat.totalErrors || 0);
    document.getElementById('chat-avg-ask').textContent =
        chat.avgAskDurationMs > 0 ? formatDuration(chat.avgAskDurationMs) : '--';
    document.getElementById('chat-avg-exec').textContent =
        chat.avgExecDurationMs > 0 ? formatDuration(chat.avgExecDurationMs) : '--';

    // Tables queried
    const byTableBody = document.getElementById('chat-by-table-body');
    const tableEntries = Object.entries(chat.byTable || {})
        .map(([table, count]) => ({ table, count }))
        .sort((a, b) => b.count - a.count);

    if (byTableBody) {
        if (tableEntries.length === 0) {
            byTableBody.innerHTML = '<tr><td colspan="2" class="empty-state">No table requests.</td></tr>';
        } else {
            byTableBody.innerHTML = tableEntries.map(e => `
                <tr>
                    <td class="report-id">${escapeHtml(e.table)}</td>
                    <td class="number"><strong>${formatNumber(e.count)}</strong></td>
                </tr>
            `).join('');
        }
    }

    // Recent interactions
    const interactionsBody = document.getElementById('chat-interactions-body');
    const interactions = chat.recentInteractions || [];

    if (interactionsBody) {
        if (interactions.length === 0) {
            interactionsBody.innerHTML = '<tr><td colspan="4" class="empty-state">No interactions yet.</td></tr>';
        } else {
            interactionsBody.innerHTML = interactions.map(i => {
                const timestamp = new Date(i.timestamp);
                const statusClass = i.isError ? 'error' : 'success';
                const statusText = i.isError ? (i.errorMsg || 'Failed') : 'Success';
                const questionSummary = i.question ? (i.question.length > 50 ? i.question.slice(0, 47) + '...' : i.question) : '';
                return `
                    <tr>
                        <td class="time-ago">${formatTimeAgo(timestamp)}</td>
                        <td title="${escapeHtml(i.question || '')}">
                            <strong>${escapeHtml(questionSummary)}</strong>
                            <div style="font-size: 11px; color: var(--caption); font-family: var(--font-mono); margin-top: 2px;">Table: ${escapeHtml(i.table || 'None')}</div>
                        </td>
                        <td class="number"><code>${formatDuration(i.durationMs)}</code></td>
                        <td><span class="status-badge ${statusClass}">${escapeHtml(statusText)}</span></td>
                    </tr>
                `;
            }).join('');
        }
    }
}

// Render component execution stats
function renderComponentStats(stats) {
    const section = document.getElementById('component-stats-section');
    const totalExecuted = stats.totalExecuted || 0;
    const totalFailed = stats.totalFailed || 0;
    if (!section) return;

    if (totalExecuted === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = 'block';
    
    const badge = document.getElementById('component-stats-count-badge');
    if (badge) badge.textContent = `${totalFailed} failures`;

    document.getElementById('comp-total-executed').textContent = formatNumber(totalExecuted);
    document.getElementById('comp-total-failed').textContent = formatNumber(totalFailed);
    document.getElementById('comp-failure-rate').textContent =
        stats.failureRate > 0 ? stats.failureRate.toFixed(2) + '%' : '0.00%';

    // Failures by report
    const byReportBody = document.getElementById('comp-by-report-body');
    const reportEntries = Object.entries(stats.failuresByReport || {})
        .map(([reportId, count]) => ({ reportId, count }))
        .sort((a, b) => b.count - a.count);

    if (byReportBody) {
        if (reportEntries.length === 0) {
            byReportBody.innerHTML = '<tr><td colspan="2" class="empty-state">No component failures.</td></tr>';
        } else {
            byReportBody.innerHTML = reportEntries.map(e => `
                <tr>
                    <td class="report-id">${escapeHtml(e.reportId)}</td>
                    <td class="number" style="color: var(--color-error); font-weight: bold;">${formatNumber(e.count)}</td>
                </tr>
            `).join('');
        }
    }

    // Recent failures
    const failuresBody = document.getElementById('comp-recent-failures-body');
    const failures = stats.recentFailures || [];

    if (failuresBody) {
        if (failures.length === 0) {
            failuresBody.innerHTML = '<tr><td colspan="2" class="empty-state">No failures yet.</td></tr>';
        } else {
            failuresBody.innerHTML = failures.map(f => {
                const timestamp = new Date(f.timestamp);
                const errorShort = f.error.length > 50 ? f.error.slice(0, 47) + '...' : f.error;
                return `
                    <tr>
                        <td class="time-ago">${formatTimeAgo(timestamp)}</td>
                        <td>
                            <strong style="color: var(--color-error);">${escapeHtml(f.reportId)}</strong> - <span>${escapeHtml(f.title)}</span>
                            <div style="font-size: 11.5px; color: var(--color-error); margin-top: 3px; font-family: var(--font-mono);" title="${escapeHtml(f.error)}">${escapeHtml(errorShort)}</div>
                        </td>
                    </tr>
                `;
            }).join('');
        }
    }
}

// Render query performance stats
function renderQueryStats(stats) {
    const section = document.getElementById('query-stats-section');
    const totalExecuted = stats.totalExecuted || 0;
    if (!section) return;

    if (totalExecuted === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = 'block';
    
    const badge = document.getElementById('query-stats-count');
    if (badge) badge.textContent = `${(stats.byTable || []).length} tables`;

    document.getElementById('query-total-executed').textContent = formatNumber(totalExecuted);
    document.getElementById('query-total-errors').textContent = formatNumber(stats.totalErrors || 0);
    document.getElementById('query-error-rate').textContent =
        stats.errorRate > 0 ? stats.errorRate.toFixed(2) + '%' : '0.00%';

    // Query speed by table
    const byTableBody = document.getElementById('query-by-table-body');
    const tables = stats.byTable || [];
    const TOP_TABLES = 25;

    if (byTableBody) {
        if (tables.length === 0) {
            byTableBody.innerHTML = '<tr><td colspan="7" class="empty-state">No query data yet.</td></tr>';
        } else {
            const sortedTables = [...tables].sort((a, b) => (b.executionCount || 0) - (a.executionCount || 0));
            const shownTables = sortedTables.slice(0, TOP_TABLES);
            
            let tableHtml = shownTables.map(t => `
                <tr>
                    <td><code>${escapeHtml(t.table)}</code></td>
                    <td class="number"><strong>${formatNumber(t.executionCount)}</strong></td>
                    <td class="number" style="${t.errorCount > 0 ? 'color: var(--color-error); font-weight: bold;' : ''}">${formatNumber(t.errorCount)}</td>
                    <td class="number duration">${formatDuration(t.avgDurationMs)}</td>
                    <td class="number duration">${formatDuration(t.maxDurationMs)}</td>
                    <td class="number">${formatNumber(t.avgRows)}</td>
                    <td class="number">${formatNumber(t.maxRows)}</td>
                </tr>
            `).join('');

            if (tables.length > TOP_TABLES) {
                tableHtml += `
                    <tr>
                        <td colspan="7" class="empty-state" style="padding: 10px;">
                            Showing top ${TOP_TABLES} of ${formatNumber(tables.length)} tables.
                        </td>
                    </tr>
                `;
            }
            byTableBody.innerHTML = tableHtml;
        }
    }

    // Slow queries list
    const slowBody = document.getElementById('query-slow-body');
    const slowQueries = stats.slowQueries || [];

    if (slowBody) {
        if (slowQueries.length === 0) {
            slowBody.innerHTML = '<tr><td colspan="4" class="empty-state">No slow queries detected.</td></tr>';
        } else {
            slowBody.innerHTML = slowQueries.map(q => {
                const timestamp = new Date(q.timestamp);
                return `
                    <tr>
                        <td class="time-ago">${formatTimeAgo(timestamp)}</td>
                        <td>
                            <code>${escapeHtml(q.table)}</code>
                            ${q.isError ? `<div style="color: var(--color-error); font-size: 11px;">Error: ${escapeHtml(q.errorMsg)}</div>` : ''}
                        </td>
                        <td class="number" style="color: var(--color-error); font-weight: bold;">${formatDuration(q.durationMs)}</td>
                        <td class="number">${formatNumber(q.rowCount)}</td>
                    </tr>
                `;
            }).join('');
        }
    }

    // Large result sets list
    const largeBody = document.getElementById('query-large-body');
    const largeResults = stats.largeResults || [];

    if (largeBody) {
        if (largeResults.length === 0) {
            largeBody.innerHTML = '<tr><td colspan="4" class="empty-state">No large result sets.</td></tr>';
        } else {
            largeBody.innerHTML = largeResults.map(q => {
                const timestamp = new Date(q.timestamp);
                return `
                    <tr>
                        <td class="time-ago">${formatTimeAgo(timestamp)}</td>
                        <td><code>${escapeHtml(q.table)}</code></td>
                        <td class="number" style="color: var(--color-warning); font-weight: bold;">${formatNumber(q.rowCount)}</td>
                        <td class="number duration">${formatDuration(q.durationMs)}</td>
                    </tr>
                `;
            }).join('');
        }
    }
}

// Render UI event log list
function renderUIEventLog(uiStats) {
    const tbody = document.getElementById('ui-event-log-body');
    if (!tbody) return;

    const events = uiStats.recentEvents || [];
    if (events.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="4" class="empty-state">No user actions tracked.</td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = events.map(ev => {
        const time = new Date(ev.timestamp);
        
        // Render props as tags/chips
        let propsHtml = '';
        if (ev.props && Object.keys(ev.props).length > 0) {
            propsHtml = Object.entries(ev.props).map(([k, v]) => 
                `<span class="chip" style="font-size: 10px;">${escapeHtml(k)}: ${escapeHtml(v)}</span>`
            ).join(' ');
        } else {
            propsHtml = '<span class="chip" style="opacity: 0.5;">None</span>';
        }

        // Highlight event types
        let eventBadgeClass = 'info';
        if (ev.name.includes('error') || ev.name.includes('validation')) {
            eventBadgeClass = 'error';
        } else if (ev.name.includes('apply')) {
            eventBadgeClass = 'success';
        } else if (ev.name.includes('save') || ev.name.includes('share')) {
            eventBadgeClass = 'warning';
        }

        return `
            <tr>
                <td class="time-ago">${formatTimeAgo(time)}</td>
                <td><span class="status-badge ${eventBadgeClass}">${escapeHtml(ev.name)}</span></td>
                <td class="report-id">${ev.reportId ? escapeHtml(ev.reportId) : 'Global'}</td>
                <td><div class="chips-container">${propsHtml}</div></td>
            </tr>
        `;
    }).join('');
}

// Show error messages in all grids
function showError(message) {
    const errRow = `<tr><td colspan="10" class="empty-state" style="color: var(--color-error); font-weight: 500;">${escapeHtml(message)}</td></tr>`;
    
    const targets = [
        'report-stats-body',
        'db-pool-body',
        'duration-buckets-body',
        'daily-hits-body',
        'top-reports-body',
        'least-reports-body',
        'query-by-table-body',
        'comp-by-report-body',
        'ui-event-log-body'
    ];

    targets.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.innerHTML = errRow;
    });

    const healthText = document.getElementById('system-health-text');
    if (healthText) healthText.textContent = 'Failed to load metrics';
}

// Formatting helpers
function formatNumber(num) {
    if (num === undefined || num === null) return '--';
    return num.toLocaleString();
}

function formatDuration(ms) {
    if (ms === undefined || ms === null) return '--';
    if (ms < 1) return '< 1ms';
    if (ms < 1000) return `${Math.round(ms)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}

function formatDateTime(date) {
    if (!date || isNaN(date.getTime())) return '--';
    return date.toLocaleString();
}

function formatUptime(startDate) {
    if (!startDate || isNaN(startDate.getTime())) return '--';

    const now = new Date();
    const diffMs = now - startDate;
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    if (diffDay > 0) {
        return `${diffDay}d ${diffHour % 24}h`;
    }
    if (diffHour > 0) {
        return `${diffHour}h ${diffMin % 60}m`;
    }
    if (diffMin > 0) {
        return `${diffMin}m`;
    }
    return `${diffSec}s`;
}

function formatTimeAgo(date) {
    if (!date || isNaN(date.getTime())) return '--';

    const now = new Date();
    const diffMs = now - date;
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);

    if (diffDay > 0) return `${diffDay}d ago`;
    if (diffHour > 0) return `${diffHour}h ago`;
    if (diffMin > 0) return `${diffMin}m ago`;
    if (diffSec > 0) return `${diffSec}s ago`;
    return 'just now';
}

function escapeHtml(str) {
    if (str === undefined || str === null) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

// ============ Feedback panel ============
const FEEDBACK_PANEL = {
    nextCursor: 0,
    hasMore: false,
    loading: false,
    totalRendered: 0,
};

function setupFeedbackPanel() {
    const more = document.getElementById('feedback-panel-more');
    if (more) more.addEventListener('click', () => loadFeedbackPanel(false));
}

async function loadFeedbackPanel(reset) {
    if (FEEDBACK_PANEL.loading) return;
    const stateEl = document.getElementById('feedback-panel-state');
    const listEl = document.getElementById('feedback-panel-list');
    const moreBtn = document.getElementById('feedback-panel-more');
    if (!stateEl || !listEl) return;

    FEEDBACK_PANEL.loading = true;
    if (reset) {
        FEEDBACK_PANEL.nextCursor = 0;
        FEEDBACK_PANEL.totalRendered = 0;
        listEl.innerHTML = '';
        listEl.style.display = 'none';
        stateEl.style.display = 'block';
        stateEl.className = 'empty-state';
        stateEl.textContent = 'Loading feedback...';
        if (moreBtn) moreBtn.style.display = 'none';
    }

    if (window.authReady && typeof window.authReady.then === 'function') {
        try { await window.authReady; } catch (e) {}
    }

    try {
        const params = new URLSearchParams({ limit: '10' });
        if (!reset && FEEDBACK_PANEL.nextCursor > 0) {
            params.set('before', String(FEEDBACK_PANEL.nextCursor));
        }

        const headers = {};
        if (typeof window.getAuthToken === 'function') {
            try {
                const token = await window.getAuthToken();
                if (token) headers['Authorization'] = 'Bearer ' + token;
            } catch (e) {}
        }

        const resp = await fetch(`${API_BASE}/feedback?${params.toString()}`, { headers });
        if (resp.status === 401 || resp.status === 403) {
            renderFeedbackSignInPrompt();
            return;
        }
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);

        const data = await resp.json();
        const entries = data.entries || [];
        FEEDBACK_PANEL.hasMore = !!data.hasMore;
        FEEDBACK_PANEL.nextCursor = data.nextCursor || 0;

        if (reset && entries.length === 0) {
            stateEl.textContent = 'No feedback logs found.';
            return;
        }

        stateEl.style.display = 'none';
        listEl.style.display = 'block';
        for (const e of entries) {
            listEl.appendChild(buildFeedbackRow(e));
        }
        FEEDBACK_PANEL.totalRendered += entries.length;

        if (moreBtn) {
            moreBtn.style.display = FEEDBACK_PANEL.hasMore ? 'block' : 'none';
        }
    } catch (err) {
        console.error('Failed to load feedback:', err);
        stateEl.style.display = 'block';
        listEl.style.display = 'none';
        stateEl.className = 'empty-state';
        stateEl.textContent = 'Failed to load feedback.';
        if (moreBtn) moreBtn.style.display = 'none';
    } finally {
        FEEDBACK_PANEL.loading = false;
    }
}

function renderFeedbackSignInPrompt() {
    const stateEl = document.getElementById('feedback-panel-state');
    const listEl = document.getElementById('feedback-panel-list');
    const moreBtn = document.getElementById('feedback-panel-more');
    if (!stateEl) return;
    listEl.style.display = 'none';
    stateEl.style.display = 'block';
    stateEl.className = 'empty-state';
    stateEl.innerHTML = '<p>Sign in required to view client feedback logs.</p>';

    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'btn-refresh';
    btn.style.margin = '10px auto 0';
    btn.textContent = 'Sign In';
    btn.addEventListener('click', () => {
        if (typeof window.signIn === 'function') {
            window.signIn();
        } else {
            console.warn('Sign-in function not available.');
        }
    });
    stateEl.appendChild(btn);

    if (moreBtn) moreBtn.style.display = 'none';
}

function buildFeedbackRow(e) {
    const row = document.createElement('div');
    row.className = 'feedback-row';

    const head = document.createElement('div');
    head.className = 'feedback-row-head';

    const user = document.createElement('span');
    user.className = 'feedback-row-user';
    user.textContent = e.username || 'anonymous';

    const reportLink = document.createElement('a');
    reportLink.className = 'feedback-row-report';
    reportLink.href = (window.BASE_PATH || '') + '/?report=' + encodeURIComponent(e.reportId || '');
    reportLink.textContent = e.reportId || '(unknown)';
    reportLink.title = 'View Report';

    const when = document.createElement('span');
    when.textContent = formatFeedbackWhen(e.createdAt);

    head.appendChild(user);
    head.appendChild(reportLink);
    if (e.sectionId) {
        const section = document.createElement('span');
        section.textContent = ` › ${e.sectionId}`;
        section.style.color = 'var(--caption)';
        head.appendChild(section);
    }
    head.appendChild(when);

    const body = document.createElement('div');
    body.className = 'feedback-row-comment';
    body.textContent = e.comment;

    row.appendChild(head);
    row.appendChild(body);
    return row;
}

function formatFeedbackWhen(unixSeconds) {
    if (!unixSeconds) return '';
    const d = new Date(unixSeconds * 1000);
    const now = new Date();
    const diff = (now - d) / 1000;
    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    if (diff < 86400 * 7) return Math.floor(diff / 86400) + 'd ago';
    return d.toLocaleDateString();
}
