// API base URL - uses BASE_PATH from server for subpath deployments
// window.BASE_PATH is injected by the server (empty string if not set)
const API_BASE = (window.BASE_PATH || '') + '/api';

// Month names for display (used by renderReportHeader, getFilterSummaryText)
const monthNames = ['January', 'February', 'March', 'April', 'May', 'June',
                    'July', 'August', 'September', 'October', 'November', 'December'];

// Search synonyms for abbreviations and alternative terms
const SEARCH_SYNONYMS = {};

// loadReport is exposed globally via its top-level function declaration

// Track current state
let currentReportMetadata = null;
let currentReportId = null;
let allReports = [];
let reportHistory = [];
let isNavigatingFromURL = false; // Flag to prevent double URL updates
let aimaraTree = null; // AimaraJS tree instance
let popularReports = []; // Cached popular reports from metrics
let reportFetchController = null; // AbortController for in-flight report fetch

// Search home page state
let searchIndex = []; // Pre-built for fast search
let autocompleteState = { results: [], selectedIndex: -1, isOpen: false };

const defaultClients = [
    {
        id: "statgate-report-builder",
        name: "StatGate Report Builder",
        baseUrl: window.BASE_PATH ? `${window.BASE_PATH}/builder.html` : "/builder.html"
    }
];

// Smart time defaults, filter bar, multiselect, and filter persistence
// are now in filterManager.js

// ============ URL State Management ============

/**
 * Parse URL parameters into state object
 */
function getStateFromURL() {
    const params = new URLSearchParams(window.location.search);
    const state = {
        reportId: params.get('report'),
        filters: {}
    };

    // Extract filter params (all except 'report')
    params.forEach((value, key) => {
        if (key !== 'report') {
            state.filters[key] = value;
        }
    });

    return state;
}

/**
 * Update URL with current state (without page reload)
 */
function updateURL(reportId, filters) {
    // Skip if navigating from URL (to prevent double updates)
    if (isNavigatingFromURL) return;

    const params = new URLSearchParams();

    if (reportId) {
        params.set('report', reportId);
    }

    // Add filter params
    for (const [key, value] of Object.entries(filters)) {
        if (value) {
            params.set(key, value);
        }
    }

    const newURL = params.toString()
        ? `${window.location.pathname}?${params.toString()}`
        : window.location.pathname;

    // Only push if URL actually changed
    if (window.location.href !== window.location.origin + newURL) {
        history.pushState({ reportId, filters }, '', newURL);
    }
}

/**
 * Apply URL state on page load
 */
async function applyURLState() {
    const state = getStateFromURL();

    if (state.reportId && allReports.some(r => r.id === state.reportId)) {
        isNavigatingFromURL = true;
        await selectReport(state.reportId, state.filters);
        isNavigatingFromURL = false;
        return;
    }

    // No report in URL — resume last viewed report if available
    const recently = JSON.parse(localStorage.getItem('recentlyViewed') || '[]');
    if (recently.length > 0) {
        const lastId = typeof recently[0] === 'object' ? recently[0].id : recently[0];
        if (lastId && allReports.some(r => r.id === lastId)) {
            isNavigatingFromURL = true;
            await selectReport(lastId);
            isNavigatingFromURL = false;
        }
    }
}

/**
 * Initialize URL state handling for back/forward navigation
 */
function initURLStateHandling() {
    window.addEventListener('popstate', async (event) => {
        if (event.state && event.state.reportId) {
            isNavigatingFromURL = true;
            await selectReport(event.state.reportId, event.state.filters);
            isNavigatingFromURL = false;
        } else {
            // No state - show empty view
            goBackToAllReports();
        }
    });
}

// ============ End URL State Management ============

// ============ Progress Bar (WhatsApp-style) ============

/**
 * Show the progress bar
 */
function showProgressBar() {
    const container = document.getElementById('progress-bar-container');
    if (container) {
        container.classList.add('active');
    }
}

/**
 * Hide the progress bar
 */
function hideProgressBar() {
    const container = document.getElementById('progress-bar-container');
    if (container) {
        container.classList.remove('active');
    }
}

// ============ Skeleton Screens ============

/**
 * Create skeleton loading placeholders for components
 */
function createSkeletonSection(numComponents = 3, layout = 'single') {
    const section = document.createElement('div');
    section.className = 'section';

    const titleSkeleton = document.createElement('div');
    titleSkeleton.className = 'skeleton skeleton-text';
    titleSkeleton.style.width = '200px';
    titleSkeleton.style.height = '24px';
    titleSkeleton.style.marginBottom = '20px';
    section.appendChild(titleSkeleton);

    const componentsContainer = document.createElement('div');
    componentsContainer.className = `section-components layout-${layout}`;

    for (let i = 0; i < numComponents; i++) {
        const component = document.createElement('div');
        component.className = 'component';

        const chartSkeleton = document.createElement('div');
        chartSkeleton.className = 'skeleton skeleton-chart';
        component.appendChild(chartSkeleton);

        componentsContainer.appendChild(component);
    }

    section.appendChild(componentsContainer);
    return section;
}

/**
 * Show skeleton screens while loading
 */
function showSkeletonScreens() {
    const sectionsContainer = document.getElementById('sections-container');
    if (!sectionsContainer) return;
    sectionsContainer.innerHTML = '';

    // Show 2-3 skeleton sections
    for (let i = 0; i < 2; i++) {
        const layout = i % 2 === 0 ? 'two-column' : 'single';
        const numComponents = layout === 'two-column' ? 2 : 3;
        sectionsContainer.appendChild(createSkeletonSection(numComponents, layout));
    }
}

// ============ Phase 2: Recently Viewed Reports & Swipe Gestures ============

/**
 * Migrate old recently viewed format (array of strings) to new format (array of objects with timestamps)
 */
function migrateRecentlyViewed() {
    const recently = JSON.parse(localStorage.getItem('recentlyViewed') || '[]');
    if (recently.length === 0) return;

    // Check if already migrated (first item is an object)
    if (typeof recently[0] === 'object' && recently[0].id) return;

    // Migrate: convert string IDs to objects with timestamps
    const migrated = recently.map((id, index) => ({
        id: id,
        viewedAt: Date.now() - (index * 3600000) // Stagger by 1 hour for display
    }));

    localStorage.setItem('recentlyViewed', JSON.stringify(migrated));
}

/**
 * Format relative time (e.g., "2 hours ago", "yesterday")
 */
function formatTimeAgo(timestamp) {
    const now = Date.now();
    const diff = now - timestamp;

    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days === 1) return 'yesterday';
    if (days < 7) return `${days}d ago`;
    return new Date(timestamp).toLocaleDateString();
}

/**
 * Track recently viewed reports in localStorage
 */
function trackRecentlyViewed(reportId) {
    migrateRecentlyViewed();
    const recently = JSON.parse(localStorage.getItem('recentlyViewed') || '[]');
    const filtered = recently.filter(item =>
        (typeof item === 'object' ? item.id : item) !== reportId
    );
    filtered.unshift({ id: reportId, viewedAt: Date.now() });
    const limited = filtered.slice(0, 2);
    localStorage.setItem('recentlyViewed', JSON.stringify(limited));
    renderRecentlyViewed();
}

/**
 * Remove a single item from recently viewed
 */
function removeFromRecentlyViewed(reportId) {
    const recently = JSON.parse(localStorage.getItem('recentlyViewed') || '[]');
    const filtered = recently.filter(item =>
        (typeof item === 'object' ? item.id : item) !== reportId
    );
    localStorage.setItem('recentlyViewed', JSON.stringify(filtered));
    renderRecentlyViewed();
}

/**
 * Render recently viewed reports in sidebar
 */
function renderRecentlyViewed() {
    migrateRecentlyViewed();
    const recently = JSON.parse(localStorage.getItem('recentlyViewed') || '[]');
    const list = document.getElementById('recently-viewed-list');
    const section = document.getElementById('recently-viewed-section');

    if (recently.length === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = 'block';
    list.innerHTML = '';

    recently.forEach(item => {
        const reportId = typeof item === 'object' ? item.id : item;
        const viewedAt = typeof item === 'object' ? item.viewedAt : null;
        const report = allReports.find(r => r.id === reportId);
        if (!report) return;

        const reportItem = document.createElement('div');
        reportItem.className = 'recently-viewed-item';
        reportItem.setAttribute('data-report-id', reportId);

        const reportBtn = document.createElement('button');
        reportBtn.className = 'recently-viewed-link';
        reportBtn.type = 'button';

        const titleSpan = document.createElement('span');
        titleSpan.className = 'recently-viewed-title';
        titleSpan.textContent = report.title;
        reportBtn.appendChild(titleSpan);

        if (viewedAt) {
            const timeSpan = document.createElement('span');
            timeSpan.className = 'recently-viewed-time';
            timeSpan.textContent = formatTimeAgo(viewedAt);
            reportBtn.appendChild(timeSpan);
        }

        reportBtn.addEventListener('click', async (e) => {
            e.preventDefault();
            await selectReport(reportId);
            closeMobileSidebar();
        });

        const removeBtn = document.createElement('button');
        removeBtn.className = 'recently-viewed-remove';
        removeBtn.type = 'button';
        removeBtn.innerHTML = '&times;';
        removeBtn.title = 'Remove from recently viewed';
        removeBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            removeFromRecentlyViewed(reportId);
        });

        reportItem.appendChild(reportBtn);
        reportItem.appendChild(removeBtn);
        list.appendChild(reportItem);
    });
}


/**
 * Initialize swipe gestures for mobile / narrow layouts (same breakpoint as overlay sidebar)
 */
function initSwipeGestures() {
    const sidebar = document.getElementById('sidebar');
    let startX = 0;

    function isOverlaySidebarMode() {
        return window.innerWidth <= 1200;
    }

    document.addEventListener('touchstart', (e) => {
        startX = e.touches[0].clientX;
    }, { passive: true });

    document.addEventListener('touchend', (e) => {
        if (!sidebar || !isOverlaySidebarMode()) return;

        const endX = e.changedTouches[0].clientX;
        const diff = endX - startX;

        if (startX < 50 && diff > 100) {
            sidebar.classList.add('active');
            document.getElementById('sidebar-overlay')?.classList.add('active');
            document.getElementById('sidebar-menu-toggle')?.classList.add('active');
        } else if (diff < -100 && sidebar.classList.contains('active')) {
            closeMobileSidebar();
        }
    }, { passive: true });
}

// ============ End Phase 2 ============


// Initialize on page load
document.addEventListener('DOMContentLoaded', async () => {
    console.log('DOMContentLoaded fired');
    try {
        // Wait for auth to initialize before loading reports
        // This ensures auth tokens are available for API calls when AUTH_MODE=on
        if (window.authReady) {
            await window.authReady;
        }

        // Register service worker for PWA
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register((window.BASE_PATH || '') + '/sw.js', {
                scope: (window.BASE_PATH || '') + '/'
            }).then(reg => console.log('SW registered, scope:', reg.scope))
              .catch(err => console.warn('SW registration failed:', err));
        }

        // Load and organize reports by category
        await loadAndOrganizeReports();

        // Initialize sidebar toggle for mobile
        initSidebarToggle();

        // Initialize desktop sidebar collapse toggle (subtle chevron)
        initSidebarCollapseToggle();

        // Initialize swipe gestures (Phase 2)
        initSwipeGestures();

        // Initialize mobile filter toggle
        filterManager.initMobileToggle();

        // Initialize sidebar search
        initSidebarSearch();

        // Render external dashboards
        renderExternalDashboards();

        // Initialize PWA features (offline detection, install prompt, favorites)
        pwa.init();

        // Initialize keyboard shortcuts
        initKeyboardShortcuts();

        // Initialize URL state handling for back/forward navigation
        initURLStateHandling();

        // Check for report in URL and auto-load
        await applyURLState();

        initMenuToggle();

        await listClients();

        // Initialize empty state with popular reports (if no report loaded from URL)
        if (!currentReportId) {
            await initEmptyState();
        }

        console.log('Initialization complete');
    } catch (error) {
        console.error('Error in DOMContentLoaded:', error);
    }
});

/**
 * Initialize keyboard shortcuts
 * / = Focus search
 * Escape = Clear search / close modals
 */
function initKeyboardShortcuts() {
    document.addEventListener('keydown', (e) => {
        // Ignore if user is typing in an input/textarea
        const activeEl = document.activeElement;
        const isTyping = activeEl && (
            activeEl.tagName === 'INPUT' ||
            activeEl.tagName === 'TEXTAREA' ||
            activeEl.isContentEditable
        );

        // Escape key - always works
        if (e.key === 'Escape') {
            // Clear search if focused
            const searchInput = document.getElementById('sidebar-search');
            if (searchInput && document.activeElement === searchInput) {
                searchInput.value = '';
                searchInput.dispatchEvent(new Event('input'));
                searchInput.blur();
                return;
            }

            // Close sidebar on mobile
            const sidebar = document.getElementById('sidebar');
            if (sidebar && sidebar.classList.contains('active')) {
                sidebar.classList.remove('active');
                document.getElementById('sidebar-overlay')?.classList.remove('active');
                document.getElementById('sidebar-menu-toggle')?.classList.remove('active');
                return;
            }

            // Close header menu if open
            const headerMenu = document.getElementById('header-menu');
            if (headerMenu && headerMenu.classList.contains('active')) {
                headerMenu.classList.remove('active');
                return;
            }
        }

        // Skip other shortcuts if typing
        if (isTyping) return;

        // / = Focus search
        if (e.key === '/') {
            e.preventDefault();
            const searchInput = document.getElementById('sidebar-search');
            if (searchInput) {
                // Open sidebar on mobile if needed
                const sidebar = document.getElementById('sidebar');
                if (sidebar && window.innerWidth <= 1200 && !sidebar.classList.contains('active')) {
                    sidebar.classList.add('active');
                    document.getElementById('sidebar-overlay')?.classList.add('active');
                    document.getElementById('sidebar-menu-toggle')?.classList.add('active');
                }
                searchInput.focus();
            }
        }

    });
}

/**
 * Load reports and organize them by nested folder hierarchy.
 * Supports: section[/subsection...]/report.yaml
 * Includes empty category folders from the folder structure.
 */
async function loadAndOrganizeReports() {
    try {
        const data = await apiCache.fetch(`${API_BASE}/reports`);
        const reports = data.reports || [];
        const categoryPaths = data.sections || []; // Includes empty folders

        // Store all reports for Phase 2 recently viewed tracking
        allReports = reports;

        const categories = {};

        // First, initialize categories from folder structure (includes empty ones).
        categoryPaths.forEach(path => {
            ensureCategoryPath(categories, path);
        });

        // Then, add reports to their respective category path.
        reports.forEach(report => {
            const category = report.category || 'uncategorized';
            ensureCategoryPath(categories, category).reports.push(report);
        });

        // Initialize AimaraJS tree in sidebar
        initAimaraTree(categories);

        // Render recently viewed (Phase 2)
        renderRecentlyViewed();

        // Build search index for search home page
        buildSearchIndex();

    } catch (error) {
        console.error('Error loading reports:', error);
        const treeContainer = document.getElementById('aimara-tree-container');
        if (treeContainer) {
            treeContainer.innerHTML = `<div role="alert" aria-live="polite" style="padding:16px;font-size:0.85em;color:var(--color-error,#dc2626);">Could not load reports. Reload the page to try again.</div>`;
        }
    }
}

/**
 * Format category name to title case (e.g., "disease-surveillance" → "Disease Surveillance")
 */
function createCategoryModelNode(key, path) {
    return {
        key,
        path,
        title: formatCategoryTitle(key),
        children: {},
        reports: []
    };
}

function ensureCategoryPath(root, path) {
    const parts = (path || 'uncategorized').split('/').filter(Boolean);
    let cursor = root;
    let node = null;
    let currentPath = '';

    parts.forEach(part => {
        currentPath = currentPath ? `${currentPath}/${part}` : part;
        if (!cursor[part]) {
            cursor[part] = createCategoryModelNode(part, currentPath);
        }
        node = cursor[part];
        cursor = node.children;
    });

    return node || ensureCategoryPath(root, 'uncategorized');
}

function formatCategoryTitle(category) {
    return category
        .split('-')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ');
}

// Predefined section order (matches business requirements)
const SECTION_ORDER = [
    'Health-Profiles',
    'Deaths',
    'Disease-Areas',
    'Programs',
    'Surveillance',
    'Community'
];

// Predefined category order at specific folder paths
const CATEGORY_ORDER = {
    '': SECTION_ORDER,
    'Health-Profiles': ['District-Profile', 'Regional-Profile', 'National-Profile', 'Subcounty', 'Facility'],
    'Disease-Areas': ['NCD'],
    'Programs': ['Malaria', 'NTLP', 'Immunization', 'ACP', 'RMCH', 'Nutrition', 'Palliative-Care', 'Commodities-Supply-Chain', 'Pharmaceutical-Services'],
    'Programs/Pharmaceutical-Services': ['Supply-Chain-and-Logistics', 'Quality-Assurance', 'Access'],
    'Programs/Pharmaceutical-Services/Supply-Chain-and-Logistics': ['Warehouse-Reports', 'Health-Facility-Reports', 'Community-Reports'],
    'Programs/Pharmaceutical-Services/Access': ['Warehouse-Reports', 'Health-Facility-and-Community-Reports']
};

/**
 * Sort categories by predefined order, with unknown items at the end
 */
function sortByOrder(items, orderList) {
    return items.sort((a, b) => {
        const indexA = orderList.indexOf(a);
        const indexB = orderList.indexOf(b);
        // Items not in the order list go to the end (alphabetically)
        if (indexA === -1 && indexB === -1) return a.localeCompare(b);
        if (indexA === -1) return 1;
        if (indexB === -1) return -1;
        return indexA - indexB;
    });
}

/**
 * Initialize AimaraJS tree with a nested category hierarchy.
 */
function initAimaraTree(categories) {
    // Create tree instance (no context menu)
    aimaraTree = createTree('aimara-tree-container', 'transparent', null);

    // Restore expanded state from localStorage
    const expandedState = JSON.parse(localStorage.getItem('expandedCategories') || '{}');

    renderCategoryNodes(categories, null, '', expandedState);

    // Draw the tree
    aimaraTree.drawTree();

    // Set up click handler for report selection
    aimaraTree.clickNodeEvent = async function(node) {
        if (node.tag && node.tag.type === 'report') {
            collapseSiblingsForNode(node);
            await selectReport(node.tag.reportId);
            closeMobileSidebar();
        } else if (node.tag && node.tag.type === 'category') {
            // Toggle category expansion. When opening, apply accordion
            // focus so peer branches collapse; when closing, leave peers alone.
            const wasExpanded = node.expanded;
            node.toggleNode();
            if (!wasExpanded) collapseSiblingsForNode(node);
            saveTreeExpandedState();
        }
    };

    // Set up expand/collapse events to save state
    aimaraTree.nodeAfterOpenEvent = function(node) {
        saveTreeExpandedState();
    };

    aimaraTree.nodeBeforeCloseEvent = function(node) {
        // Delay save to after close completes
        setTimeout(saveTreeExpandedState, 10);
    };
}

function renderCategoryNodes(categoryMap, parentNode, parentPath, expandedState) {
    const order = CATEGORY_ORDER[parentPath] || [];
    const sortedKeys = sortByOrder(Object.keys(categoryMap), order);

    sortedKeys.forEach(key => {
        const category = categoryMap[key];
        const isExpanded = expandedState[category.path] || false;
        const tag = {
            type: 'category',
            path: category.path,
            sectionKey: category.path.split('/')[0],
            categoryKey: key
        };

        const node = parentNode
            ? parentNode.createChildNode(category.title, isExpanded, null, tag, null)
            : aimaraTree.createNode(category.title, isExpanded, null, null, tag, null);

        renderCategoryNodes(category.children, node, category.path, expandedState);

        category.reports.forEach(report => {
            node.createChildNode(
                report.title,
                false,
                null,
                { type: 'report', reportId: report.id },
                null
            );
        });
    });
}

/**
 * Accordion behavior: collapse every category that is not on the clicked
 * node's ancestor chain. Skipped while a sidebar search is active, since
 * search auto-expands matching branches.
 */
function collapseSiblingsForNode(targetNode) {
    if (!aimaraTree || !targetNode) return;

    const searchInput = document.getElementById('sidebar-search');
    if (searchInput && searchInput.value.trim() !== '') return;

    const ancestors = new Set();
    let cursor = targetNode;
    while (cursor) {
        ancestors.add(cursor);
        cursor = cursor.parent;
    }

    function collapseOutsideBranch(nodes) {
        nodes.forEach(node => {
            if (node.childNodes) {
                collapseOutsideBranch(node.childNodes);
            }
            if (node.tag && node.tag.type === 'category'
                && node.expanded && !ancestors.has(node)) {
                node.collapseNode();
            }
        });
    }

    collapseOutsideBranch(aimaraTree.childNodes);

    saveTreeExpandedState();
}

function walkTreeNodes(nodes, callback) {
    nodes.forEach(node => {
        callback(node);
        if (node.childNodes) {
            walkTreeNodes(node.childNodes, callback);
        }
    });
}

/**
 * Save tree expanded state to localStorage
 */
function saveTreeExpandedState() {
    if (!aimaraTree) return;

    const expanded = {};
    walkTreeNodes(aimaraTree.childNodes, node => {
        if (node.tag && node.tag.type === 'category') {
            expanded[node.tag.path] = node.expanded;
        }
    });

    localStorage.setItem('expandedCategories', JSON.stringify(expanded));
}

/**
 * Highlight active report in tree
 */
function highlightActiveReportInTree(reportId) {
    if (!aimaraTree) return;

    // Find and select the report node
    const reportNode = aimaraTree.findNodeByTag('reportId', reportId);
    if (reportNode) {
        // Expand every category ancestor so deeply nested reports are visible.
        let expandCursor = reportNode.parent;
        while (expandCursor) {
            if (expandCursor.expanded === false) {
                expandCursor.expandNode();
            }
            expandCursor = expandCursor.parent;
        }

        // Add active class to the node content
        const allContents = document.querySelectorAll('.tree-content');
        allContents.forEach(el => el.classList.remove('active'));
        // Clear previous ancestor highlights
        document.querySelectorAll('.tree-node.active-ancestor').forEach(el => el.classList.remove('active-ancestor'));

        const nodeContent = reportNode.elementLi.querySelector('.tree-content');
        if (nodeContent) {
            nodeContent.classList.add('active');
        }

        // Highlight the root section ancestor
        let ancestor = reportNode.parent;
        while (ancestor && ancestor.elementLi) {
            // Root section nodes are direct children of ul.aimara-tree
            if (ancestor.elementLi.parentElement && ancestor.elementLi.parentElement.classList.contains('aimara-tree')) {
                ancestor.elementLi.classList.add('active-ancestor');
                break;
            }
            ancestor = ancestor.parent;
        }

        // Select the node in the tree
        aimaraTree.selectNode(reportNode);
    }
}

/**
 * Handle report selection
 * @param {string} reportId - The report ID to load
 * @param {object|null} urlFilters - Optional filters from URL state
 */
async function selectReport(reportId, urlFilters = null) {
    // Close any open Superset embed before switching reports
    closeExternalDashboardEmbed();

    currentReportId = reportId;
    currentReportMetadata = null;

    // Track report open telemetry
    window.telemetry?.track('report.open', { reportId: reportId });

    // Immediate visual feedback - show loading state before fetching metadata
    document.getElementById('empty-state').style.display = 'none';
    document.getElementById('sections-container').style.display = 'flex';
    showProgressBar();
    showSkeletonScreens();

    // Immediately show the new report's title so users know their click registered.
    // Title is available from allReports (loaded at startup) before the API call.
    const knownReport = allReports.find(r => r.id === reportId);
    const headerEl = document.getElementById('report-header');
    const titleEl = document.getElementById('report-header-title');
    const filtersEl = document.getElementById('report-header-filters');
    if (headerEl && titleEl) {
        titleEl.textContent = knownReport?.title || formatCategoryTitle(reportId.split('/').pop());
        if (filtersEl) filtersEl.textContent = 'Loading\u2026';
        headerEl.style.display = 'block';
        renderBreadcrumb();
    }
    const fbBtn = document.getElementById('feedback-btn');
    if (fbBtn) fbBtn.style.display = 'inline-flex';

    // Track recently viewed (Phase 2)
    trackRecentlyViewed(reportId);

    // Track navigation history (Phase 3)
    if (reportHistory[reportHistory.length - 1] !== reportId) {
        reportHistory.push(reportId);
    }

    // Mark report as active in sidebar tree
    highlightActiveReportInTree(reportId);

    try {
        // Fetch report metadata to get filter definitions
        currentReportMetadata = await apiCache.fetch(`${API_BASE}/report/${reportId}`);
        console.log('Report metadata loaded, filters:', currentReportMetadata.filters);

        // Initialize filter manager and build filter bar
        filterManager.init(currentReportMetadata, reportId);
        await filterManager.buildFilterBar();

        // Apply filters: URL filters take priority, then localStorage, then smart defaults
        if (urlFilters && Object.keys(urlFilters).length > 0) {
            filterManager.setValues(urlFilters);
        } else {
            const restored = filterManager.restoreState();

            // Apply smart time defaults if no saved filters for this report
            if (!restored && currentReportMetadata.filters) {
                filterManager.applySmartDefaults();
            }
        }

        // Auto-load report with current filters
        await loadReport();

    } catch (error) {
        console.error('Error selecting report:', error);
    }
}

/**
 * Capitalize first letter
 */
function capitalizeFirstLetter(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Render breadcrumb navigation
 */
function renderBreadcrumb() {
    const breadcrumb = document.getElementById('breadcrumb');
    const breadcrumbContent = document.getElementById('breadcrumb-content');

    if (!breadcrumb || !breadcrumbContent || !currentReportId) {
        if (breadcrumb) breadcrumb.style.display = 'none';
        return;
    }

    const reportId = currentReportId;
    const parts = reportId.split('/');

    // Build breadcrumb links - always start with "All" (home)
    const crumbs = [];
    crumbs.push(`<a href="#" onclick="goHome(event); return false;">All</a>`);

    if (parts.length > 1) {
        parts.slice(0, -1).forEach((part, index) => {
            const path = parts.slice(0, index + 1).join('/');
            crumbs.push(`<a href="#" onclick="filterSidebarBySection('${path}'); return false;">${formatCategoryTitle(part)}</a>`);
        });
    }

    // Current report name (not a link)
    const reportName = currentReportMetadata?.title || formatCategoryTitle(parts[parts.length - 1]);
    crumbs.push(`<span>${reportName}</span>`);

    breadcrumbContent.innerHTML = crumbs.join(' <span class="breadcrumb-sep">&gt;</span> ');
    breadcrumb.style.display = 'block';
}

/**
 * Navigate to a category from breadcrumb. Collapses all, then expands the
 * target category and its ancestors.
 */
function filterSidebarBySection(sectionPath) {
    if (!aimaraTree) return;

    // Clear search first
    const searchInput = document.getElementById('sidebar-search');
    if (searchInput && searchInput.value) {
        searchInput.value = '';
        searchInput.dispatchEvent(new Event('input'));
    }

    // Collapse all categories first
    aimaraTree.collapseTree();

    let targetNode = null;
    walkTreeNodes(aimaraTree.childNodes, node => {
        if (node.tag && node.tag.type === 'category' && node.tag.path === sectionPath) {
            targetNode = node;
        }
    });

    let cursor = targetNode;
    while (cursor) {
        if (cursor.tag && cursor.tag.type === 'category') {
            cursor.expandNode();
        }
        cursor = cursor.parent;
    }

    // Save the expanded state
    saveTreeExpandedState();

    // Make sure sidebar is visible when navigating via breadcrumb
    const sidebar = document.getElementById('sidebar');
    const sidebarMenuToggle = document.getElementById('sidebar-menu-toggle');

    if (sidebar) {
        if (window.innerWidth > 1200) {
            // Desktop: expand sidebar if collapsed (user navigated to browse)
            if (sidebar.classList.contains('collapsed')) {
                sidebar.classList.remove('collapsed');
                localStorage.setItem('sidebarCollapsed', 'false');
                syncSidebarCollapseToggleLabel(false);
            }
        } else {
            // Small laptop / mobile: show sidebar overlay
            sidebar.classList.add('active');
            document.getElementById('sidebar-overlay')?.classList.add('active');
            sidebarMenuToggle?.classList.add('active');
        }
    }
}

/**
 * Render report header with logo, brand name, title, and filters
 */
function renderReportHeader(filterValues) {
    const headerEl = document.getElementById('report-header');
    const titleEl = document.getElementById('report-header-title');
    const filtersEl = document.getElementById('report-header-filters');

    if (!headerEl || !currentReportMetadata) {
        return;
    }

    // Render breadcrumb navigation
    renderBreadcrumb();

    // Set report title
    titleEl.textContent = currentReportMetadata.title || '';

    // Build filter display string (period first: month/week/quarter, then year, then geography, facility last)
    const filterParts = [];
    const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

    if (filterValues.month) {
        const months = filterValues.month.split(',').map(m => monthNames[parseInt(m) - 1] || m);
        if (months.length <= 3) {
            filterParts.push(`Month: ${months.join(', ')}`);
        }
    }

    if (filterValues.week) {
        const weeks = filterValues.week.split(',');
        if (weeks.length <= 3) {
            filterParts.push(`Week: ${filterValues.week}`);
        }
    }

    if (filterValues.quarter) {
        const quarters = filterValues.quarter.split(',').map(q => `Q${q}`);
        if (quarters.length <= 3) {
            filterParts.push(`Quarter: ${quarters.join(', ')}`);
        }
    }

    if (filterValues.year) {
        filterParts.push(`Year: ${filterValues.year}`);
    }

    if (filterValues.region) {
        const regions = filterValues.region.split(',');
        if (regions.length <= 2) {
            filterParts.push(`Region: ${regions.join(', ')}`);
        } else {
            filterParts.push(`Regions: ${regions.length} selected`);
        }
    }

    if (filterValues.district) {
        const districts = filterValues.district.split(',');
        if (districts.length <= 2) {
            filterParts.push(`District: ${districts.join(', ')}`);
        } else {
            filterParts.push(`Districts: ${districts.length} selected`);
        }
    }

    if (filterValues.facility) {
        const facilities = filterValues.facility.split(',');
        if (facilities.length <= 2) {
            filterParts.push(`Facility: ${facilities.join(', ')}`);
        } else {
            filterParts.push(`Facilities: ${facilities.length} selected`);
        }
    }

    filtersEl.textContent = filterParts.join(' | ');

    // Add PWA action buttons (favorite + save offline) to filter bar
    const filterBarButtons = document.querySelector('.filter-bar-buttons');
    if (filterBarButtons) {
        // Remove any previous PWA buttons
        filterBarButtons.querySelectorAll('.report-action-btn').forEach(el => el.remove());

        // Re-trigger the CSS pulse animation on the icon span.
        const pulseIcon = (el) => {
            const icon = el.querySelector('.action-icon');
            if (!icon) return;
            icon.classList.remove('pulse');
            // Force reflow so removing + re-adding the class restarts the animation.
            void icon.offsetWidth;
            icon.classList.add('pulse');
        };

        // Favorite (star) button
        const starBtn = document.createElement('button');
        starBtn.id = 'btn-favorite';
        starBtn.type = 'button';
        starBtn.className = 'report-action-btn btn-favorite' + (pwa.isFavorite(currentReportId) ? ' active' : '');
        starBtn.dataset.reportId = currentReportId;
        const favLabel = pwa.isFavorite(currentReportId) ? 'Remove from favorites' : 'Add to favorites';
        starBtn.setAttribute('data-tooltip', favLabel);
        starBtn.setAttribute('aria-label', favLabel);
        starBtn.innerHTML = '<span class="action-icon">' + (pwa.isFavorite(currentReportId) ? '\u2605' : '\u2606') + '</span>';
        starBtn.addEventListener('click', () => {
            const added = pwa.toggleFavorite(currentReportId);
            starBtn.classList.toggle('active', added);
            const t = added ? 'Remove from favorites' : 'Add to favorites';
            starBtn.setAttribute('data-tooltip', t);
            starBtn.setAttribute('aria-label', t);
            starBtn.querySelector('.action-icon').textContent = added ? '\u2605' : '\u2606';
            pulseIcon(starBtn);
        });
        filterBarButtons.appendChild(starBtn);

        // Save offline button
        const offlineBtn = document.createElement('button');
        offlineBtn.type = 'button';
        offlineBtn.className = 'report-action-btn btn-offline' + (pwa.isReportSavedOffline(currentReportId) ? ' saved' : '');
        const offSavedInit = pwa.isReportSavedOffline(currentReportId);
        const offLabelInit = offSavedInit ? 'Saved for offline (click to remove)' : 'Save for offline viewing';
        offlineBtn.setAttribute('data-tooltip', offLabelInit);
        offlineBtn.setAttribute('aria-label', offLabelInit);
        offlineBtn.innerHTML = '<span class="action-icon">' + (offSavedInit ? '\u2713' : '\u2193') + '</span>';
        offlineBtn.addEventListener('click', async () => {
            if (pwa.isReportSavedOffline(currentReportId)) {
                await pwa.removeOfflineReport(currentReportId);
                offlineBtn.classList.remove('saved');
                const t = 'Save for offline viewing';
                offlineBtn.setAttribute('data-tooltip', t);
                offlineBtn.setAttribute('aria-label', t);
                offlineBtn.querySelector('.action-icon').textContent = '\u2193';
                pulseIcon(offlineBtn);
                pwa.showToast('Removed from offline reports');
            } else {
                // Get current report data from the last fetch
                const fv = filterManager.getValues();
                const p = new URLSearchParams();
                for (const [k, v] of Object.entries(fv)) p.append(k, v);
                const qs = p.toString() ? '?' + p.toString() : '';
                try {
                    const data = await apiCache.fetch(API_BASE + '/report/' + currentReportId + qs);
                    await pwa.saveForOffline(currentReportId, fv, data);
                    offlineBtn.classList.add('saved');
                    const st = 'Saved for offline (click to remove)';
                    offlineBtn.setAttribute('data-tooltip', st);
                    offlineBtn.setAttribute('aria-label', st);
                    offlineBtn.querySelector('.action-icon').textContent = '\u2713';
                    pulseIcon(offlineBtn);
                } catch (err) {
                    pwa.showToast('Could not save: ' + err.message);
                }
            }
        });
        filterBarButtons.appendChild(offlineBtn);
    }

    // Show the header
    headerEl.style.display = 'block';
}

/**
 * Load and display report with current filters
 */
async function loadReport(options = {}) {
    if (!currentReportId) {
        console.warn('No report selected');
        return;
    }

    let timedOut = false;
    try {
        // Ensure sections container is visible and show loading state
        document.getElementById('sections-container').style.display = 'flex';
        showProgressBar();
        showSkeletonScreens();

        // Update filter pills (clear all visibility, summary, badge)
        filterManager.updatePills();

        // Get filter values from filter bar (includes both checkboxes and regular selects)
        const filterValues = filterManager.getValues();

        // Render report header with title and filters
        renderReportHeader(filterValues);

        // Build query string
        const params = new URLSearchParams();
        for (const [key, value] of Object.entries(filterValues)) {
            params.append(key, value);
        }
        if (options && options.noTimeDefaults) {
            params.append('no_time_defaults', '1');
        }
        const queryString = params.toString() ? `?${params.toString()}` : '';

        // Cancel any in-flight report fetch before starting a new one
        if (reportFetchController) {
            reportFetchController.abort();
        }
        reportFetchController = new AbortController();
        const thisController = reportFetchController;

        // 45-second hard timeout — sets a flag so the catch block can distinguish
        // timeout aborts from user-navigation aborts
        const timeoutId_hard = setTimeout(() => {
            timedOut = true;
            thisController.abort();
        }, 45000);

        // Show "taking longer than expected" banner after 8 seconds
        const slowBannerId = setTimeout(() => {
            if (thisController.signal.aborted) return;
            const sc = document.getElementById('sections-container');
            if (sc && !sc.querySelector('.slow-loading-banner')) {
                const banner = document.createElement('div');
                banner.className = 'slow-loading-banner';
                banner.textContent = 'Taking longer than usual\u2026 the query is still running.';
                sc.prepend(banner);
            }
        }, 8000);

        const fetchSignal = thisController.signal;

        let report;
        try {
            report = await apiCache.fetch(`${API_BASE}/report/${currentReportId}${queryString}`, { bypassCache: true, signal: fetchSignal });
        } finally {
            clearTimeout(timeoutId_hard);
            clearTimeout(slowBannerId);
            const slowBanner = document.querySelector('.slow-loading-banner');
            if (slowBanner) slowBanner.remove();
        }
        const sectionsContainer = document.getElementById('sections-container');

        // Cleanup existing charts and maps to prevent memory leaks
        cleanupAllInstances();

        // Render sections (replaces skeleton screens)
        if (sectionsContainer) {
            if (report.allEmpty) {
                // All components returned empty data - show single alert instead of many empty messages
                sectionsContainer.innerHTML = `
                    <div style="padding:48px 24px;text-align:center;max-width:480px;margin:40px auto;">
                        <div style="font-size:1.1em;font-weight:600;color:var(--text-primary);margin-bottom:8px;">No data available</div>
                        <div style="color:var(--text-secondary);line-height:1.5;">No data was returned for this report with the current filters. Try adjusting the year, month, or district.</div>
                    </div>`;
            } else if (report.sections && report.sections.length > 0) {
                // Build all sections into a DocumentFragment to batch DOM updates (single reflow)
                const fragment = document.createDocumentFragment();
                // Show partial failure banner at the top
                if (report.partialFailure && report.failedComponents && report.failedComponents.length > 0) {
                    const n = report.failedComponents.length;
                    const banner = document.createElement('div');
                    banner.id = 'partial-failure-banner';
                    banner.style.cssText = 'margin:0 0 12px 0;padding:10px 16px;background:var(--color-warning-bg,#fef3c7);color:var(--color-warning-text,#78350f);border-radius:6px;border-left:3px solid var(--color-warning,#f59e0b);font-size:0.875em;';
                    banner.textContent = `${n} component${n > 1 ? 's' : ''} could not load data. Other components are shown below.`;
                    fragment.appendChild(banner);
                }
                report.sections.forEach(section => {
                    renderSection(fragment, section);
                });
                sectionsContainer.innerHTML = '';
                sectionsContainer.appendChild(fragment);
            } else {
                sectionsContainer.innerHTML = '<p style="padding:40px;text-align:center;color:var(--text-secondary);">No data found for these filters</p>';
            }
        }

        // Render report footer (category siblings + timestamp)
        renderReportFooter(report, allReports, currentReportId);

        // Update filter summary for mobile
        filterManager.updatePills();

        // Update URL with current state
        updateURL(currentReportId, filterValues);

        // Hide progress bar on success
        hideProgressBar();

    } catch (error) {
        // Silently ignore aborted fetches from user navigating to a different report
        if (error.name === 'AbortError' && !timedOut) return;

        console.error('Error loading report:', error);
        hideProgressBar();
        cleanupAllInstances();
        const sectionsContainer = document.getElementById('sections-container');
        if (sectionsContainer) {
            sectionsContainer.innerHTML = '';

            if (pwa.isOffline() || (error.message && error.message.includes('offline'))) {
                const isSaved = pwa.isReportSavedOffline(currentReportId);
                const offlineBox = document.createElement('div');
                offlineBox.className = 'report-offline-state';
                offlineBox.setAttribute('role', 'alert');
                offlineBox.setAttribute('aria-live', 'assertive');
                const offlineTitle = document.createElement('strong');
                offlineTitle.textContent = 'You are offline';
                const offlineMsg = document.createElement('p');
                offlineMsg.textContent = isSaved
                    ? 'Loading saved version of this report...'
                    : 'This report is not available offline. Save reports while connected to view them later.';
                offlineBox.appendChild(offlineTitle);
                offlineBox.appendChild(offlineMsg);
                sectionsContainer.appendChild(offlineBox);
            } else {
                const errBox = document.createElement('div');
                errBox.className = 'report-error-state';
                errBox.setAttribute('role', 'alert');
                errBox.setAttribute('aria-live', 'assertive');
                const strong = document.createElement('strong');
                strong.textContent = timedOut ? 'Request timed out: ' : 'Error: ';
                const span = document.createElement('span');
                span.textContent = timedOut
                    ? 'The report took too long to load. This may be due to a complex query or slow connection.'
                    : error.message;
                errBox.appendChild(strong);
                errBox.appendChild(span);
                const retryBtn = document.createElement('button');
                retryBtn.className = 'report-retry-btn';
                retryBtn.textContent = 'Try again';
                retryBtn.addEventListener('click', () => loadReport());
                errBox.appendChild(retryBtn);
                sectionsContainer.appendChild(errBox);
                retryBtn.focus();
            }
        }
    }
}

/**
 * Initialize sidebar toggle for mobile and header menu
 */
function initSidebarToggle() {
    const sidebarToggle = document.getElementById('sidebar-toggle');
    const headerMenu = document.getElementById('header-menu');
    const sidebar = document.getElementById('sidebar');
    const sidebarOverlay = document.getElementById('sidebar-overlay');
    const sidebarMenuToggle = document.getElementById('sidebar-menu-toggle');

    // Hamburger menu toggle (for sidebar - hidden on large desktop, visible <=1200px)
    if (sidebarMenuToggle && sidebar) {
        // Large desktop: sidebar is always expanded (no collapse option)
        if (window.innerWidth > 1200) {
            sidebar.classList.remove('collapsed');
        }

        sidebarMenuToggle.addEventListener('click', (e) => {
            e.stopPropagation();

            // Only toggle on small laptop / mobile - large desktop sidebar is always visible
            if (window.innerWidth <= 1200) {
                sidebar.classList.toggle('active');
                sidebarOverlay?.classList.toggle('active');
                sidebarMenuToggle.classList.toggle('active');
            }
        });
    }

    // Menu toggle (vertical dots menu)
    if (sidebarToggle && headerMenu) {
        sidebarToggle.addEventListener('click', (e) => {
            e.stopPropagation();
            headerMenu.classList.toggle('active');
        });
    }

    // Close menu when clicking outside
    document.addEventListener('click', (e) => {
        if (headerMenu && !headerMenu.contains(e.target) && e.target !== sidebarToggle) {
            headerMenu.classList.remove('active');
        }
        // Close share menu when clicking outside
        const shareMenu = document.getElementById('share-menu');
        const shareBtn = document.getElementById('share-btn');
        if (shareMenu && !shareMenu.contains(e.target) && e.target !== shareBtn) {
            shareMenu.classList.remove('active');
        }
        // Multiselect panel close-on-outside-click is handled by filterManager
    });

    // Close menu when selecting an item
    if (headerMenu) {
        const menuItems = headerMenu.querySelectorAll('.menu-item:not(:disabled)');
        menuItems.forEach(item => {
            item.addEventListener('click', () => {
                headerMenu.classList.remove('active');
            });
        });
    }

    // Sidebar overlay click to close
    if (sidebarOverlay) {
        sidebarOverlay.addEventListener('click', () => {
            closeMobileSidebar();
        });
    }

    // Handle window resize - ensure sidebar is always visible on large desktop
    window.addEventListener('resize', () => {
        if (window.innerWidth > 1200) {
            // Switching to large desktop - close mobile overlay state, then
            // restore the user's persisted collapsed preference.
            sidebarOverlay?.classList.remove('active');
            sidebar?.classList.remove('active');
            const collapsed = localStorage.getItem('sidebarCollapsed') === 'true';
            sidebar?.classList.toggle('collapsed', collapsed);
            syncSidebarCollapseToggleLabel(collapsed);
        } else {
            // Switching to small laptop / mobile - .collapsed is desktop-only,
            // overlay pattern takes over.
            sidebar?.classList.remove('collapsed');
            sidebarMenuToggle?.classList.remove('active');
        }
    });
}

function syncSidebarCollapseToggleLabel(collapsed) {
    const toggle = document.getElementById('sidebar-collapse-toggle');
    if (!toggle) return;
    const label = collapsed ? 'Expand sidebar' : 'Collapse sidebar';
    toggle.title = label;
    toggle.setAttribute('aria-label', label);
    toggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
}

/**
 * Initialize the desktop sidebar collapse toggle (subtle chevron at top-right
 * of the sidebar). Persists state in localStorage. No-op on small laptop /
 * mobile where the hamburger overlay pattern is used instead.
 */
function initSidebarCollapseToggle() {
    const sidebar = document.getElementById('sidebar');
    const toggle = document.getElementById('sidebar-collapse-toggle');
    if (!sidebar || !toggle) return;

    // Apply persisted state on desktop without an animation flash on first load.
    if (window.innerWidth > 1200 && localStorage.getItem('sidebarCollapsed') === 'true') {
        sidebar.classList.add('no-transitions', 'collapsed');
        // Force reflow, then re-enable transitions for subsequent toggling.
        void sidebar.offsetWidth;
        requestAnimationFrame(() => sidebar.classList.remove('no-transitions'));
        syncSidebarCollapseToggleLabel(true);
    } else {
        syncSidebarCollapseToggleLabel(false);
    }

    toggle.addEventListener('click', (e) => {
        e.stopPropagation();
        // Only operates on desktop. On smaller screens the hamburger handles it.
        if (window.innerWidth <= 1200) return;
        const collapsed = sidebar.classList.toggle('collapsed');
        localStorage.setItem('sidebarCollapsed', String(collapsed));
        syncSidebarCollapseToggleLabel(collapsed);
    });
}

/**
 * Close mobile sidebar
 */
function closeMobileSidebar() {
    const sidebar = document.getElementById('sidebar');
    const sidebarOverlay = document.getElementById('sidebar-overlay');
    const sidebarMenuToggle = document.getElementById('sidebar-menu-toggle');
    if (sidebar) sidebar.classList.remove('active');
    if (sidebarOverlay) sidebarOverlay.classList.remove('active');
    if (sidebarMenuToggle) sidebarMenuToggle.classList.remove('active');
}

/**
 * Initialize sidebar search functionality
 */
function initSidebarSearch() {
    const searchInput = document.getElementById('sidebar-search');
    const clearBtn = document.getElementById('sidebar-search-clear');

    if (!searchInput) return;

    searchInput.addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase().trim();
        filterSidebarReports(query);
        clearBtn.style.display = query ? 'block' : 'none';
    });

    clearBtn.addEventListener('click', () => {
        searchInput.value = '';
        filterSidebarReports('');
        clearBtn.style.display = 'none';
        searchInput.focus();
    });

    // Clear search on Escape key
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            searchInput.value = '';
            filterSidebarReports('');
            clearBtn.style.display = 'none';
            searchInput.blur();
        }
    });
}

/**
 * Filter sidebar reports by search query (AimaraJS tree version)
 * Handles nested category → report hierarchy
 * Uses the same scored search engine as the home page (keywords, synonyms, category, etc.)
 */
function filterSidebarReports(query) {
    if (!aimaraTree) return;

    const noResultsEl = document.getElementById('sidebar-no-results');
    let totalVisible = 0;

    // Use the same scored engine as the home page: match by report ID, not just title.
    // Require 2+ chars to match home page behaviour; null means no filter (show all).
    const matchedIds = query.length >= 2
        ? new Set(searchReports(query).map(r => r.id))
        : null;

    function applyFilter(node) {
        if (!node.tag) return 0;

        if (node.tag.type === 'report') {
            const matches = !matchedIds || matchedIds.has(node.tag.reportId);
            if (node.elementLi) {
                node.elementLi.classList.toggle('search-hidden', !matches);
            }
            return matches ? 1 : 0;
        }

        if (node.tag.type !== 'category') return 0;

        const visibleInCategory = (node.childNodes || []).reduce((sum, child) => {
            return sum + applyFilter(child);
        }, 0);

        if (node.elementLi) {
            node.elementLi.classList.toggle('search-hidden', visibleInCategory === 0);
        }

        if (visibleInCategory > 0 && matchedIds && !node.expanded) {
            node.expandNode();
        }

        return visibleInCategory;
    }

    totalVisible = aimaraTree.childNodes.reduce((sum, node) => sum + applyFilter(node), 0);

    // Show/hide no results message
    if (noResultsEl) {
        noResultsEl.style.display = totalVisible === 0 && query ? 'block' : 'none';
    }
}


// ============ Phase 3: Related Reports Navigation ============

/**
 * Go back to all reports view
 */
function goBackToAllReports() {
    reportHistory = [];
    currentReportId = null;
    currentReportMetadata = null;
    filterManager.destroy();
    cleanupAllInstances();
    document.getElementById('filter-bar').style.display = 'none';
    document.getElementById('breadcrumb').style.display = 'none';
    document.getElementById('report-header').style.display = 'none';
    const fbBtn = document.getElementById('feedback-btn');
    if (fbBtn) fbBtn.style.display = 'none';
    document.getElementById('sections-container').innerHTML = '';
    document.getElementById('sections-container').style.display = 'none';
    hideReportFooter();
    renderEmptyState(); // Show search home page
    document.querySelectorAll('.report-item').forEach(item => item.classList.remove('active'));

    // Clear tree selection
    document.querySelectorAll('.tree-content').forEach(el => el.classList.remove('active'));
    document.querySelectorAll('.tree-node.active-ancestor').forEach(el => el.classList.remove('active-ancestor'));

    // Clear sidebar search
    const searchInput = document.getElementById('sidebar-search');
    if (searchInput && searchInput.value) {
        searchInput.value = '';
        searchInput.dispatchEvent(new Event('input'));
    }

    // Collapse all tree categories
    if (aimaraTree) {
        aimaraTree.collapseTree();
        saveTreeExpandedState();
    }

    // Expand sidebar so user can browse reports
    const sidebar = document.getElementById('sidebar');
    const sidebarMenuToggle = document.getElementById('sidebar-menu-toggle');

    if (sidebar) {
        if (window.innerWidth > 1200) {
            // Large desktop: expand sidebar if collapsed (user came home to browse)
            if (sidebar.classList.contains('collapsed')) {
                sidebar.classList.remove('collapsed');
                localStorage.setItem('sidebarCollapsed', 'false');
                syncSidebarCollapseToggleLabel(false);
            }
        } else {
            // Small laptop / mobile: show sidebar overlay
            sidebar.classList.add('active');
            document.getElementById('sidebar-overlay')?.classList.add('active');
            sidebarMenuToggle?.classList.add('active');
        }
    }

    // Clear URL state
    history.pushState(null, '', window.location.pathname);
}

/**
 * Navigate to home (reports landing page)
 * Called when clicking the logo/Reports header
 */
function goHome(event) {
    if (event) {
        event.preventDefault();
    }
    // If chat view is active, switch back to home first
    if (typeof chatState !== 'undefined' && chatState.view === 'chat') {
        showHomeView();
    }
    goBackToAllReports();
}

// ============ End Phase 3 ============

// ============ Popular Reports in Empty State ============

/**
 * Fetch popular reports from metrics API
 */
async function fetchPopularReports() {
    try {
        const data = await apiCache.fetch(`${API_BASE}/metrics`);
        const reportStats = data.reportStats || [];

        // Sort by request count and take top 5
        const sorted = reportStats
            .filter(stat => stat.requestCount > 0)
            .sort((a, b) => b.requestCount - a.requestCount)
            .slice(0, 5);

        // Map to report objects with titles
        popularReports = sorted.map(stat => {
            const report = allReports.find(r => r.id === stat.reportId);
            return {
                id: stat.reportId,
                title: report ? report.title : formatCategoryTitle(stat.reportId.split('/').pop()),
                requestCount: stat.requestCount
            };
        }).filter(r => r.id); // Remove any undefined

        return popularReports;
    } catch (error) {
        console.warn('Could not fetch popular reports:', error);
        return [];
    }
}

/**
 * Build search index from all reports for fast searching
 * Called once after reports are loaded
 */
function buildSearchIndex() {
    searchIndex = allReports.map(report => {
        // Build category path (e.g., "Programs/Malaria")
        const categoryPath = report.category || '';
        const categoryParts = categoryPath.split('/').filter(p => p);
        const categorySearchable = categoryParts.map(p => formatCategoryTitle(p)).join(' ');

        // Parse keywords into array for better matching
        const keywords = report.keywords || '';
        const keywordsArray = keywords.split(',').map(k => k.trim().toLowerCase()).filter(k => k);

        const searchText = report.searchText || '';

        return {
            id: report.id,
            title: report.title || '',
            titleLower: (report.title || '').toLowerCase(),
            keywords: keywords,
            keywordsLower: keywords.toLowerCase(),
            keywordsArray: keywordsArray,
            description: report.description || '',
            descriptionLower: (report.description || '').toLowerCase(),
            category: categoryPath,
            categoryLower: categoryPath.toLowerCase(),
            categorySearchable: categorySearchable,
            categorySearchableLower: categorySearchable.toLowerCase(),
            searchText: searchText,
            searchTextLower: searchText.toLowerCase()
        };
    });
}

/**
 * Expand query terms using synonyms
 * Returns array of all terms to search (original + synonyms)
 */
function expandQueryWithSynonyms(queryTerms) {
    const expanded = new Set(queryTerms);

    for (const term of queryTerms) {
        const synonyms = SEARCH_SYNONYMS[term];
        if (synonyms) {
            for (const syn of synonyms) {
                expanded.add(syn);
            }
        }
    }

    return Array.from(expanded);
}

/**
 * Search reports by query string
 * Returns scored and ranked results
 */
function searchReports(query) {
    if (!query || query.length < 2) return [];

    const queryLower = query.toLowerCase().trim();
    const queryTerms = queryLower.split(/\s+/).filter(t => t.length > 0);

    // Expand query terms with synonyms for broader matching
    const expandedTerms = expandQueryWithSynonyms(queryTerms);

    const results = [];

    for (const item of searchIndex) {
        let score = 0;

        // Title contains query (exact substring): 100 points
        if (item.titleLower.includes(queryLower)) {
            score += 100;
        }

        // All query terms in title: 80 points
        const allTermsInTitle = queryTerms.every(term => item.titleLower.includes(term));
        if (allTermsInTitle && !item.titleLower.includes(queryLower)) {
            score += 80;
        }

        // Keywords match (curated for search): 70 points for exact keyword match
        if (item.keywordsArray && item.keywordsArray.length > 0) {
            // Exact keyword match (e.g., "malaria" matches keyword "malaria")
            if (item.keywordsArray.some(kw => kw === queryLower)) {
                score += 70;
            }
            // Partial keyword match (query is part of keyword or vice versa)
            else if (item.keywordsLower.includes(queryLower)) {
                score += 50;
            }
        }

        // Category match (search the full path): 50 points
        if (item.categoryLower.includes(queryLower) || item.categorySearchableLower.includes(queryLower)) {
            score += 50;
        }

        // Section/component title match: 35 points
        if (item.searchTextLower.includes(queryLower)) {
            score += 35;
        }

        // Description match (legacy, lower priority): 20 points
        if (item.descriptionLower.includes(queryLower)) {
            score += 20;
        }

        // Check individual terms for partial matches
        if (score === 0) {
            for (const term of queryTerms) {
                // Individual keyword term matches: 25 points each
                if (item.keywordsArray && item.keywordsArray.some(kw => kw.includes(term))) score += 25;
                if (item.titleLower.includes(term)) score += 20;
                if (item.searchTextLower.includes(term)) score += 15;
                if (item.categorySearchableLower.includes(term)) score += 15;
                if (item.descriptionLower.includes(term)) score += 10;
            }
        }

        // Synonym matching: check expanded terms against title and keywords
        // Lower score than direct matches to prioritize exact matches
        if (score === 0) {
            for (const term of expandedTerms) {
                // Skip original query terms (already checked above)
                if (queryTerms.includes(term)) continue;

                if (item.titleLower.includes(term)) score += 45;
                if (item.keywordsArray && item.keywordsArray.some(kw => kw.includes(term))) score += 40;
                if (item.categorySearchableLower.includes(term)) score += 25;
                if (item.searchTextLower.includes(term)) score += 20;
                if (item.descriptionLower.includes(term)) score += 15;
            }
        }

        if (score > 0) {
            const report = allReports.find(r => r.id === item.id);
            results.push({
                id: item.id,
                title: report?.title || item.title,
                // Show keywords in search results if available, otherwise description
                description: report?.keywords || report?.description || item.keywords || item.description,
                category: item.categorySearchable || item.category,
                score: score
            });
        }
    }

    // Sort by score (descending) and limit to 8
    return results
        .sort((a, b) => b.score - a.score)
        .slice(0, 8);
}

/**
 * Highlight matching text in a string
 */
function highlightMatch(text, query) {
    if (!text || !query) return text;

    const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escaped})`, 'gi');
    return text.replace(regex, '<mark>$1</mark>');
}

/**
 * Get a random suggested report (not in popular list)
 */
function getRandomSuggestedReport() {
    const popularIds = new Set(popularReports.map(r => r.id));
    const nonPopular = allReports.filter(r => !popularIds.has(r.id));

    if (nonPopular.length === 0) {
        // Fall back to any report if all are popular
        if (allReports.length === 0) return null;
        return allReports[Math.floor(Math.random() * allReports.length)];
    }

    return nonPopular[Math.floor(Math.random() * nonPopular.length)];
}

/**
 * Render the Google-style search home page
 */
function renderSearchHomePage() {
    const emptyState = document.getElementById('empty-state');
    if (!emptyState) return;

    // Get random suggested report
    const suggestedReport = getRandomSuggestedReport();

    // Build popular reports pills with view counts
    let popularHtml = '';
    if (popularReports.length > 0) {
        const pills = popularReports.map(r => {
            const viewCount = r.requestCount ? ` (${r.requestCount})` : '';
            return `<button type="button" class="search-popular-pill" data-report-id="${r.id}">${r.title}<span class="popular-view-count">${viewCount}</span></button>`;
        }).join('');

        popularHtml = `
            <div class="search-popular">
                <p class="search-popular-label">Popular Reports</p>
                <div class="search-popular-list">${pills}</div>
            </div>
        `;
    }

    // Build suggested report link
    let suggestionHtml = '';
    if (suggestedReport) {
        suggestionHtml = `
            <div class="search-suggestion">
                <span class="search-suggestion-label">Try this:</span>
                <button type="button" class="search-suggestion-link" data-report-id="${suggestedReport.id}">${suggestedReport.title}</button>
            </div>
        `;
    }

    emptyState.innerHTML = `
        <div class="search-home">
            <h2 class="search-home-title">StatGate Report Builder</h2>
            <p class="search-home-tagline">Turning your data into insight</p>
            <div class="search-home-input-wrapper">
                <span class="search-home-icon">🔍</span>
                <input type="text"
                       class="search-home-input"
                       id="search-home-input"
                       placeholder="Search..."
                       autocomplete="off"
                       spellcheck="false">
                <button type="button" class="search-home-clear" id="search-home-clear">&times;</button>
                <div class="search-autocomplete" id="search-autocomplete"></div>
            </div>
            ${suggestionHtml}
            ${popularHtml}
        </div>
    `;

    emptyState.style.display = 'flex';

    // Initialize search functionality
    initSearchHome();
}

/**
 * Initialize search home page event listeners
 */
// AbortController for search home document-level listeners
let _searchHomeAbortController = null;

function initSearchHome() {
    const input = document.getElementById('search-home-input');
    const clearBtn = document.getElementById('search-home-clear');
    const autocompleteEl = document.getElementById('search-autocomplete');

    if (!input) return;

    // Focus input on load
    setTimeout(() => input.focus(), 100);

    // Input handler - search as user types
    input.addEventListener('input', (e) => {
        const query = e.target.value;

        if (query.length < 2) {
            closeAutocomplete();
            return;
        }

        const results = searchReports(query);
        autocompleteState.results = results;
        autocompleteState.selectedIndex = -1;

        renderAutocomplete(results, query);
    });

    // Keyboard navigation
    input.addEventListener('keydown', handleAutocompleteKeydown);

    // Clear button
    if (clearBtn) {
        clearBtn.addEventListener('click', () => {
            input.value = '';
            closeAutocomplete();
            input.focus();
        });
    }

    // Click outside to close — use AbortController to prevent stacking
    if (_searchHomeAbortController) {
        _searchHomeAbortController.abort();
    }
    _searchHomeAbortController = new AbortController();
    document.addEventListener('click', (e) => {
        if (!e.target.closest('.search-home-input-wrapper')) {
            closeAutocomplete();
        }
    }, { signal: _searchHomeAbortController.signal });

    // Delegated click handler on autocomplete container (prevents per-item listeners)
    if (autocompleteEl && !autocompleteEl.dataset.delegated) {
        autocompleteEl.dataset.delegated = 'true';
        autocompleteEl.addEventListener('click', (e) => {
            const item = e.target.closest('.search-autocomplete-item');
            if (item) {
                const reportId = item.getAttribute('data-report-id');
                if (reportId) {
                    closeAutocomplete();
                    selectReport(reportId);
                }
            }
        });
        autocompleteEl.addEventListener('mouseover', (e) => {
            const item = e.target.closest('.search-autocomplete-item');
            if (item) {
                const index = parseInt(item.getAttribute('data-index'));
                if (!isNaN(index)) {
                    updateAutocompleteSelection(index);
                }
            }
        });
    }

    // Popular report pills click handlers
    document.querySelectorAll('.search-popular-pill').forEach(pill => {
        pill.addEventListener('click', () => {
            const reportId = pill.getAttribute('data-report-id');
            if (reportId) selectReport(reportId);
        });
    });

    // Suggested report click handler
    const suggestionLink = document.querySelector('.search-suggestion-link');
    if (suggestionLink) {
        suggestionLink.addEventListener('click', () => {
            const reportId = suggestionLink.getAttribute('data-report-id');
            if (reportId) selectReport(reportId);
        });
    }
}

/**
 * Render autocomplete dropdown
 */
function renderAutocomplete(results, query) {
    const autocompleteEl = document.getElementById('search-autocomplete');
    if (!autocompleteEl) return;

    if (results.length === 0) {
        autocompleteEl.innerHTML = `
            <div class="search-autocomplete-no-results">
                No reports found for "${query}"
            </div>
        `;
        autocompleteEl.classList.add('open');
        autocompleteState.isOpen = true;
        return;
    }

    const itemsHtml = results.map((result, index) => {
        const highlightedTitle = highlightMatch(result.title, query);
        const description = result.description
            ? (result.description.length > 80 ? result.description.slice(0, 80) + '...' : result.description)
            : '';

        return `
            <div class="search-autocomplete-item${index === autocompleteState.selectedIndex ? ' selected' : ''}"
                 data-index="${index}"
                 data-report-id="${result.id}">
                <span class="search-autocomplete-item-title">${highlightedTitle}</span>
                <div class="search-autocomplete-item-meta">
                    <span class="search-autocomplete-item-category">${result.category}</span>
                    ${description ? `<span class="search-autocomplete-item-description">${description}</span>` : ''}
                </div>
            </div>
        `;
    }).join('');

    autocompleteEl.innerHTML = itemsHtml;
    autocompleteEl.classList.add('open');
    autocompleteState.isOpen = true;
    // Click and hover handlers are delegated from initSearchHome()
}

/**
 * Handle keyboard navigation in autocomplete
 */
function handleAutocompleteKeydown(e) {
    if (!autocompleteState.isOpen) return;

    const { results, selectedIndex } = autocompleteState;

    switch (e.key) {
        case 'ArrowDown':
            e.preventDefault();
            const nextIndex = selectedIndex < results.length - 1 ? selectedIndex + 1 : 0;
            updateAutocompleteSelection(nextIndex);
            break;

        case 'ArrowUp':
            e.preventDefault();
            const prevIndex = selectedIndex > 0 ? selectedIndex - 1 : results.length - 1;
            updateAutocompleteSelection(prevIndex);
            break;

        case 'Enter':
            e.preventDefault();
            if (selectedIndex >= 0 && selectedIndex < results.length) {
                closeAutocomplete();
                selectReport(results[selectedIndex].id);
            }
            break;

        case 'Escape':
            closeAutocomplete();
            break;
    }
}

/**
 * Update autocomplete selection highlight
 */
function updateAutocompleteSelection(index) {
    autocompleteState.selectedIndex = index;

    const autocompleteEl = document.getElementById('search-autocomplete');
    if (!autocompleteEl) return;

    autocompleteEl.querySelectorAll('.search-autocomplete-item').forEach((item, i) => {
        item.classList.toggle('selected', i === index);
    });

    // Scroll into view if needed
    const selectedItem = autocompleteEl.querySelector('.search-autocomplete-item.selected');
    if (selectedItem) {
        selectedItem.scrollIntoView({ block: 'nearest' });
    }
}

/**
 * Close autocomplete dropdown
 */
function closeAutocomplete() {
    const autocompleteEl = document.getElementById('search-autocomplete');
    if (autocompleteEl) {
        autocompleteEl.classList.remove('open');
    }
    autocompleteState.isOpen = false;
    autocompleteState.selectedIndex = -1;
}

/**
 * Render empty state - now renders search home page
 */
function renderEmptyState() {
    renderSearchHomePage();
}

/**
 * Initialize empty state with popular reports
 * Called after reports are loaded
 */
async function initEmptyState() {
    await fetchPopularReports();
    buildSearchIndex();
    renderSearchHomePage();
}

// ============ End Popular Reports ============

/**
 * Wire up filter bar buttons
 */
document.addEventListener('DOMContentLoaded', () => {
    const applyBtn = document.getElementById('apply-filters-btn');
    const clearAllBtn = document.getElementById('clear-all-filters');

    if (applyBtn) {
        applyBtn.addEventListener('click', async () => {
            window.telemetry?.track('filter.apply', { reportId: currentReportId });
            filterManager.setApplyLoading(true);
            filterManager.showFeedback();
            await loadReport();
            filterManager.setApplyLoading(false);
        });
    }

    if (clearAllBtn) {
        clearAllBtn.addEventListener('click', async () => {
            window.telemetry?.track('filter.reset', { reportId: currentReportId });
            await filterManager.clearAll();
            filterManager.showFeedback();
            await loadReport({ noTimeDefaults: true });
        });
    }
});

/**
 * Convert all Chart.js canvases and maps to base64 images in the report
 * This allows charts and maps to be properly rendered in the PDF
 */
async function convertChartsToImages() {
    const sectionsContainer = document.getElementById('sections-container');
    const reportHeader = document.getElementById('report-header');

    // Force-render all lazy-loaded chart/map placeholders before PDF capture
    const placeholders = sectionsContainer.querySelectorAll('.chart-placeholder');
    for (const placeholder of placeholders) {
        if (placeholder._lazyRenderFn) {
            placeholder._lazyRenderFn();
        }
    }

    // Second pass: force inner lazy renders for off-screen charts that never entered
    // the viewport (IntersectionObserver never fired, canvas is blank)
    const innerLazyCharts = [...sectionsContainer.querySelectorAll('.component-chart')]
        .filter(el => typeof el._lazyRenderFn === 'function');
    for (const chartEl of innerLazyCharts) {
        const fn = chartEl._lazyRenderFn;
        delete chartEl._lazyRenderFn; // prevent double-init when IntersectionObserver fires later
        fn();
    }

    if (placeholders.length > 0 || innerLazyCharts.length > 0) {
        // Wait for Chart.js (rAF) and Leaflet map init (setTimeout)
        await new Promise(r => requestAnimationFrame(() => setTimeout(r, 150)));
        // Wait for any pending GeoJSON fetches (choropleth data)
        const geojsonPromises = Object.values(window._geojsonCache || {});
        if (geojsonPromises.length > 0) {
            await Promise.allSettled(geojsonPromises);
        }
        // Buffer for canvas painting to finish
        await new Promise(r => setTimeout(r, 300));
    }

    // Expand paginated tables to show all rows in the PDF (not just current page)
    const expandableTables = [...sectionsContainer.querySelectorAll('.component-table')]
        .filter(el => typeof el._pdfRenderAll === 'function');
    expandableTables.forEach(el => el._pdfRenderAll());

    // Capture all maps from the ORIGINAL DOM (where they're rendered)
    // We need to do this before cloning because maps render asynchronously
    const mapCaptures = new Map();

    // Capture regular maps
    const maps = sectionsContainer.querySelectorAll('.component-map');
    for (const mapEl of maps) {
        try {
            const canvas = await html2canvas(mapEl, {
                useCORS: true,
                allowTaint: true,
                backgroundColor: '#ffffff',
                scale: 2, // Higher quality
                logging: false
            });
            mapCaptures.set(mapEl, canvas.toDataURL('image/png'));
        } catch (e) {
            console.warn('Failed to capture map:', e);
        }
    }

    // Capture choropleth maps
    const choropleths = sectionsContainer.querySelectorAll('.component-choropleth');
    for (const choroplethEl of choropleths) {
        try {
            const canvas = await html2canvas(choroplethEl, {
                useCORS: true,
                allowTaint: true,
                backgroundColor: '#ffffff',
                scale: 2,
                logging: false
            });
            mapCaptures.set(choroplethEl, canvas.toDataURL('image/png'));
        } catch (e) {
            console.warn('Failed to capture choropleth:', e);
        }
    }

    // Capture faceted choropleth containers (the whole grid)
    const facetedChoropleths = sectionsContainer.querySelectorAll('.component-faceted-choropleth');
    for (const facetedEl of facetedChoropleths) {
        try {
            const canvas = await html2canvas(facetedEl, {
                useCORS: true,
                allowTaint: true,
                backgroundColor: '#ffffff',
                scale: 2,
                logging: false
            });
            mapCaptures.set(facetedEl, canvas.toDataURL('image/png'));
        } catch (e) {
            console.warn('Failed to capture faceted choropleth:', e);
        }
    }

    // Clone the container (tables are now showing all rows)
    const clone = sectionsContainer.cloneNode(true);

    // Restore normal pagination on the live tables immediately after cloning
    expandableTables.forEach(el => el._pdfRestorePage());

    // Replace maps in clone with captured images
    // We need to match by index since cloned elements are different objects
    const clonedMaps = clone.querySelectorAll('.component-map');
    maps.forEach((originalMap, index) => {
        const imageData = mapCaptures.get(originalMap);
        if (imageData && clonedMaps[index]) {
            const img = document.createElement('img');
            img.src = imageData;
            img.style.width = '100%';
            img.style.height = 'auto';
            img.className = 'component-map-image';
            clonedMaps[index].innerHTML = '';
            clonedMaps[index].appendChild(img);
        }
    });

    const clonedChoropleths = clone.querySelectorAll('.component-choropleth');
    choropleths.forEach((originalChoropleth, index) => {
        const imageData = mapCaptures.get(originalChoropleth);
        if (imageData && clonedChoropleths[index]) {
            const img = document.createElement('img');
            img.src = imageData;
            img.style.width = '100%';
            img.style.height = 'auto';
            img.className = 'component-choropleth-image';
            clonedChoropleths[index].innerHTML = '';
            clonedChoropleths[index].appendChild(img);
        }
    });

    const clonedFaceted = clone.querySelectorAll('.component-faceted-choropleth');
    facetedChoropleths.forEach((originalFaceted, index) => {
        const imageData = mapCaptures.get(originalFaceted);
        if (imageData && clonedFaceted[index]) {
            const img = document.createElement('img');
            img.src = imageData;
            img.style.width = '100%';
            img.style.height = 'auto';
            img.className = 'component-faceted-image';
            clonedFaceted[index].innerHTML = '';
            clonedFaceted[index].appendChild(img);
        }
    });

    // Convert Chart.js canvases to images
    const canvases = clone.querySelectorAll('canvas');
    for (const canvas of canvases) {
        const chartId = canvas.id;
        const chartInstance = chartInstances[chartId];

        if (chartInstance) {
            const imageData = chartInstance.toBase64Image();
            const img = document.createElement('img');
            img.src = imageData;
            img.style.width = canvas.style.width || '100%';
            img.style.height = canvas.style.height || 'auto';
            img.style.maxWidth = '100%';
            canvas.parentNode.replaceChild(img, canvas);
        }
    }

    // Include report header in the PDF output
    let headerHTML = '';
    if (reportHeader && reportHeader.style.display !== 'none') {
        const headerClone = reportHeader.cloneNode(true);
        headerClone.style.display = 'block';

        // Remove category from PDF (keep only: logo, brand, title, filters)
        const categoryEl = headerClone.querySelector('.report-header-category');
        if (categoryEl) {
            categoryEl.remove();
        }

        // Convert logo to base64 so it works in headless Chrome
        const logoImg = headerClone.querySelector('.report-header-logo');
        if (logoImg) {
            try {
                const canvas = document.createElement('canvas');
                const originalLogo = reportHeader.querySelector('.report-header-logo');
                canvas.width = originalLogo.naturalWidth || 100;
                canvas.height = originalLogo.naturalHeight || 100;
                const ctx = canvas.getContext('2d');
                ctx.drawImage(originalLogo, 0, 0);
                logoImg.src = canvas.toDataURL('image/png');
            } catch (e) {
                console.warn('Failed to convert logo to base64:', e);
            }
        }

        headerHTML = headerClone.outerHTML;
    }

    return headerHTML + clone.innerHTML;
}

/**
 * Download the current report as PDF
 */
async function downloadReportAsPDF() {
    const downloadPdfBtn = document.getElementById('download-pdf-btn');
    const shareBtn = document.getElementById('share-btn');
    const originalText = downloadPdfBtn ? downloadPdfBtn.innerHTML : '';
    const originalShareInner = shareBtn ? shareBtn.innerHTML : '';

    try {
        // Show loading state - use share button since menu closes
        showProgressBar();
        if (shareBtn) {
            shareBtn.innerHTML = '<span class="btn-share-loading" aria-live="polite">PDF…</span>';
            shareBtn.disabled = true;
        }
        if (downloadPdfBtn) {
            downloadPdfBtn.disabled = true;
        }

        // Convert charts to images and get HTML from #sections-container
        // The backend will read the CSS file and wrap the HTML
        const htmlContent = await convertChartsToImages();

        // Build headers with auth token if available
        const headers = { 'Content-Type': 'application/json' };
        if (typeof window.getAuthToken === 'function') {
            const token = await window.getAuthToken();
            if (token) {
                headers['Authorization'] = `Bearer ${token}`;
            }
        }

        // Send to API (backend will add CSS and wrap in HTML document)
        const response = await fetch(`${API_BASE}/report/pdf`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
                html: htmlContent,
                reportId: currentReportId || ''
            })
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Failed to generate PDF');
        }

        // Get PDF blob
        const blob = await response.blob();

        // Create download link
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        const reportTitle = currentReportMetadata?.title || 'health-report';
        const sanitizedTitle = reportTitle.replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-|-$/g, '');
        a.download = `${sanitizedTitle}-${new Date().toISOString().split('T')[0]}.pdf`;
        document.body.appendChild(a);
        a.click();

        // Cleanup
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);

        // Restore button state
        hideProgressBar();
        if (shareBtn) {
            shareBtn.innerHTML = originalShareInner;
            shareBtn.disabled = false;
        }
        if (downloadPdfBtn) {
            downloadPdfBtn.disabled = false;
            downloadPdfBtn.innerHTML = originalText;
        }

        console.log('PDF downloaded successfully');
    } catch (error) {
        console.error('Error generating PDF:', error);
        alert('Failed to generate PDF: ' + error.message);

        // Restore button state
        hideProgressBar();
        if (shareBtn) {
            shareBtn.innerHTML = originalShareInner;
            shareBtn.disabled = false;
        }
        if (downloadPdfBtn) {
            downloadPdfBtn.disabled = false;
            downloadPdfBtn.innerHTML = originalText || 'Download PDF';
        }
    }
}

/**
 * Toggle share menu visibility
 */
function toggleShareMenu(e) {
    e.stopPropagation();
    const shareMenu = document.getElementById('share-menu');
    if (shareMenu) {
        const willOpen = !shareMenu.classList.contains('active');
        shareMenu.classList.toggle('active');
        if (willOpen) {
            window.telemetry?.track('share.menu_open', { reportId: currentReportId });
        }
    }
}

/**
 * Copy current report link to clipboard
 */
function copyReportLink() {
    const shareMenu = document.getElementById('share-menu');
    if (shareMenu) shareMenu.classList.remove('active');

    window.telemetry?.track('share.copy_link', { reportId: currentReportId });

    navigator.clipboard.writeText(window.location.href).then(() => {
        // Show brief confirmation
        const btn = document.getElementById('share-btn');
        const originalText = btn.textContent;
        btn.textContent = 'Copied!';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 1500);
    }).catch(err => {
        console.error('Failed to copy:', err);
        alert('Failed to copy link. Please copy from the address bar.');
    });
}

/**
 * Share report via email (tries default mail client, falls back to Gmail compose)
 */
function shareViaEmail() {
    const shareMenu = document.getElementById('share-menu');
    if (shareMenu) shareMenu.classList.remove('active');

    const title = currentReportMetadata?.title || 'StatGate Report';
    const filterSummary = getFilterSummaryText();

    const subject = encodeURIComponent(`${title}${filterSummary ? ' - ' + filterSummary : ''}`);
    const body = encodeURIComponent(
        `View the report here:\n${window.location.href}\n\n--\nShared from Health Dashboard`
    );

    // Try mailto: first; if no mail client handles it, fall back to Gmail
    const mailtoUrl = `mailto:?subject=${subject}&body=${body}`;
    const gmailUrl = `https://mail.google.com/mail/?view=cm&su=${subject}&body=${body}`;

    const w = window.open(mailtoUrl, '_blank');
    setTimeout(() => {
        // If the window opened but stayed on about:blank, mailto: wasn't handled
        if (!w || w.closed || !w.location || w.location.href === 'about:blank') {
            if (w) w.close();
            window.open(gmailUrl, '_blank');
        }
    }, 500);
}

/**
 * Share report via WhatsApp
 */
function shareViaWhatsApp() {
    const shareMenu = document.getElementById('share-menu');
    if (shareMenu) shareMenu.classList.remove('active');

    const title = currentReportMetadata?.title || 'StatGate Report';
    const filterSummary = getFilterSummaryText();

    const message = encodeURIComponent(
        `${title}${filterSummary ? ' (' + filterSummary + ')' : ''}\n${window.location.href}`
    );

    window.open(`https://wa.me/?text=${message}`, '_blank');
}

/**
 * Get human-readable filter summary for sharing
 */
function getFilterSummaryText() {
    const parts = [];
    const filters = getStateFromURL().filters;

    // District/Region
    if (filters.district) parts.push(filters.district);
    else if (filters.region) parts.push(filters.region);

    // Time period
    if (filters.year) {
        if (filters.quarter) {
            parts.push(`Q${filters.quarter} ${filters.year}`);
        } else if (filters.month) {
            const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
            const monthNums = filters.month.split(',');
            if (monthNums.length === 1) {
                parts.push(`${months[parseInt(monthNums[0])-1]} ${filters.year}`);
            } else {
                parts.push(filters.year);
            }
        } else {
            parts.push(filters.year);
        }
    }

    return parts.join(', ');
}


function initMenuToggle() {
    const toggleBtn = document.getElementById('menuIconBtn');
    const container = document.getElementById('app_menu_container');

    if (!toggleBtn || !container) return;

    const closeMenu = () => {
        container.classList.add('hidden');
        toggleBtn.setAttribute('aria-expanded', 'false');
    };

    const openMenu = () => {
        container.classList.remove('hidden');
        toggleBtn.setAttribute('aria-expanded', 'true');
        container.focus();
    };

    toggleBtn.addEventListener('click', (event) => {
        event.stopPropagation();
        if (container.classList.contains('hidden')) {
            openMenu();
        } else {
            closeMenu();
        }
    });

    document.addEventListener('click', (event) => {
        const isClickInside = container.contains(event.target);
        const isClickOnButton = toggleBtn.contains(event.target);

        if (!isClickInside && !isClickOnButton) {
            closeMenu();
        }
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && !container.classList.contains('hidden')) {
            closeMenu();
            toggleBtn.focus();
        }
    });

    container.addEventListener('click', (event) => {
        const appLink = event.target.closest('.app-menu-tile');
        if (appLink) {
            closeMenu();
            return;
        }
        event.stopPropagation();
    });
}

function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

async function resolveMenuAuthToken() {
    if (typeof window.getAuthToken === 'function') {
        const token = await window.getAuthToken();
        if (token) return token;
    }
    return getCookie('access_token') || null;
}

async function resolveCurrentUserForMenu() {
    if (window.currentUser) {
        return window.currentUser;
    }
    if (typeof window.fetchCurrentUser === 'function') {
        return window.fetchCurrentUser();
    }
    return null;
}

function mapAccessibleSystemsToClients(systems) {
    if (!Array.isArray(systems)) return [];

    return systems
        .filter(system => system && system.displayInLauncher !== false && system.launchUrl)
        .map(system => ({
            id: system.clientId,
            clientId: system.clientId,
            name: system.displayName || system.clientId,
            displayName: system.displayName || system.clientId,
            baseUrl: system.launchUrl,
            launchUrl: system.launchUrl,
            launchMode: system.launchMode || 'internal',
            order: system.sortOrder,
            icon: system.icon,
            category: system.category,
            roles: Array.isArray(system.roles) ? system.roles : [],
            attributes: {
                'ui.order': system.sortOrder,
                'ui.launchMode': system.launchMode || 'internal',
            },
        }));
}

async function listClients() {
    renderClientsLoading();

    try {
        const user = await resolveCurrentUserForMenu();
        const accessibleClients = mapAccessibleSystemsToClients(user?.accessibleSystems || []);
        if (accessibleClients.length > 0) {
            renderClients(accessibleClients);
            return;
        }
    } catch (error) {
        console.warn('Current user applications unavailable, falling back to client API:', error);
    }

    const configs = await getFrontedConfig();
    const API_URL = configs?.clientAPIUrl;

    if (!API_URL) {
        renderClients(defaultClients);
        return;
    }

    try {
        const token = await resolveMenuAuthToken();
        const headers = { 'Content-Type': 'application/json' };
        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }

        const response = await fetch(API_URL, {
            method: 'GET',
            credentials: 'include',
            headers,
        });

        if (!response.ok) throw new Error('Network response was not ok');

        const res = await response?.json();
        renderClients(res?.data || []);
    } catch (error) {
        renderClients(defaultClients);
        console.error('Failed to fetch clients:', error);
    }
}

function escapeHtml(value) {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function getClientHref(client) {
    return escapeHtml(client?.baseUrl || client?.launchUrl || '#');
}

function getClientTarget(client) {
    const launchMode = client?.launchMode || client?.attributes?.['ui.launchMode'];
    return launchMode === 'new_tab' ? '_blank' : '_self';
}

function getClientRel(client) {
    return getClientTarget(client) === '_blank' ? 'noopener noreferrer' : '';
}

function getClientName(client) {
    return escapeHtml(client?.name || client?.displayName || client?.clientId || 'Application');
}

function renderClientsLoading() {
    const listContainer = document.getElementById('app_menu_content');
    if (!listContainer) return;

    listContainer.className = 'app-grid-state';
    listContainer.innerHTML = '<span>Loading applications...</span>';
}

function renderClients(clients) {
    const listContainer = document.getElementById('app_menu_content');

    if (!listContainer) return;

    if (!clients || clients.length === 0) {
        listContainer.className = 'app-grid-state';
        listContainer.innerHTML = `
            <div class="app-grid-state__tile">
                <h5 class="app-grid-state__title">No applications available</h5>
                <p class="app-grid-state__description">Applications assigned to your account will appear here.</p>
            </div>
        `;
        return;
    }

    const sortedClients = [...clients].sort((a, b) => {
        const orderA = Number(a?.attributes?.['ui.order'] ?? a?.order ?? Number.MAX_SAFE_INTEGER);
        const orderB = Number(b?.attributes?.['ui.order'] ?? b?.order ?? Number.MAX_SAFE_INTEGER);

        if (Number.isFinite(orderA) && Number.isFinite(orderB) && orderA !== orderB) {
            return orderA - orderB;
        }

        return getClientName(a).localeCompare(getClientName(b));
    });

    listContainer.className = 'app-grid';

    const html = sortedClients.map(client => `
       <div class="app-grid__item" role="listitem">
         <a class="app-menu-tile"
            tabindex="0"
            id="clickable-tile-${escapeHtml(client?.id || client?.clientId || '')}"
            href="${getClientHref(client)}"
            target="${getClientTarget(client)}"
            rel="${getClientRel(client)}">
           <span class="app-menu-tile__icon-wrap" aria-hidden="true">
             <svg focusable="false" preserveAspectRatio="xMidYMid meet" fill="currentColor" width="20" height="20" viewBox="0 0 32 32" aria-hidden="true" xmlns="http://www.w3.org/2000/svg"><path d="M14 4H18V8H14zM4 4H8V8H4zM24 4H28V8H24zM14 14H18V18H14zM4 14H8V18H4zM24 14H28V18H24zM14 24H18V28H14zM4 24H8V28H4zM24 24H28V28H24z"></path></svg>
           </span>
           <span class="app-menu-tile__label">${getClientName(client)}</span>
         </a>
        </div>
    `).join('');

    listContainer.innerHTML = html;
}

async function getFrontedConfig() {

    try {
        const basePath = window.BASE_PATH || '';
        const response = await fetch(`${basePath}/api/frontend/config`);
        if (!response.ok) {
            throw new Error(`Auth config fetch failed: ${response.status}`);
        }
        const configs = await response.json();
        return configs;
    } catch (error) {
        console.error('Fetching frontend configs:', error);
        return {};
    }
}

/**
 * External dashboards — grouped by thematic area, rendered at sidebar bottom.
 */
const EXTERNAL_DASHBOARDS = [
];

function setExternalDashboardMode(open) {
    const reportContainer = document.getElementById('report-container');
    const breadcrumb = document.getElementById('breadcrumb');
    const filterBar = document.getElementById('filter-bar');

    if (reportContainer) {
        if (open) {
            reportContainer.dataset.prevDisplay = reportContainer.style.display || '';
            reportContainer.style.display = 'none';
        } else {
            reportContainer.style.display = reportContainer.dataset.prevDisplay || '';
        }
    }
    if (breadcrumb) {
        if (open) {
            breadcrumb.dataset.prevDisplay = breadcrumb.style.display || '';
            breadcrumb.style.display = 'none';
        } else {
            breadcrumb.style.display = breadcrumb.dataset.prevDisplay || '';
        }
    }
    if (filterBar) {
        if (open) {
            filterBar.dataset.prevDisplay = filterBar.style.display || '';
            filterBar.style.display = 'none';
        } else {
            filterBar.style.display = filterBar.dataset.prevDisplay || '';
        }
    }
}

function openExternalDashboardEmbed(link) {
    const embedPanel = document.getElementById('external-dashboard-embed');
    const frame = document.getElementById('external-dashboard-embed-frame');
    const title = document.getElementById('external-dashboard-embed-title');
    const error = document.getElementById('external-dashboard-embed-error');
    const openLink = document.getElementById('external-dashboard-embed-open-link');

    if (!embedPanel || !frame || !title || !error || !openLink) return;

    title.textContent = link.name;
    openLink.href = link.url;
    error.style.display = 'none';
    frame.src = link.url;
    embedPanel.style.display = '';
    setExternalDashboardMode(true);

    frame.onload = () => {
        error.style.display = 'none';
    };

    frame.onerror = () => {
        error.style.display = '';
    };
}

function closeExternalDashboardEmbed() {
    const embedPanel = document.getElementById('external-dashboard-embed');
    const frame = document.getElementById('external-dashboard-embed-frame');

    if (!embedPanel || !frame) return;

    frame.src = 'about:blank';
    embedPanel.style.display = 'none';
    setExternalDashboardMode(false);
}

function renderExternalDashboards() {
    const list = document.getElementById('dashboards-list');
    if (!list) return;

    const html = EXTERNAL_DASHBOARDS.map(group => `
        <div class="dashboards-group">
            <div class="dashboards-area">${group.area}</div>
            ${group.links.map(link => `
                <a href="${link.url}" ${link.embed ? '' : 'target="_blank" rel="noopener noreferrer"'} class="dashboard-link" data-embed="${link.embed ? 'true' : 'false'}">
                    <span class="dashboard-link-name">${link.name}</span>
                    <svg class="dashboard-external-icon" width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <path d="M10.5 1.5L5.5 6.5M10.5 1.5H7.5M10.5 1.5V4.5M5 1.5H2.5C1.95 1.5 1.5 1.95 1.5 2.5V9.5C1.5 10.05 1.95 10.5 2.5 10.5H9.5C10.05 10.5 10.5 10.05 10.5 9.5V7" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                </a>
            `).join('')}
        </div>
    `).join('');

    list.innerHTML = html;

    list.querySelectorAll('.dashboard-link').forEach(linkElement => {
        if (linkElement.dataset.embed === 'true') {
            linkElement.addEventListener('click', event => {
                event.preventDefault();
                const link = EXTERNAL_DASHBOARDS.flatMap(group => group.links).find(item => item.url === linkElement.href);
                if (link) {
                    openExternalDashboardEmbed(link);
                }
            });
        }
    });

    const closeButton = document.getElementById('external-dashboard-embed-close');
    closeButton?.addEventListener('click', closeExternalDashboardEmbed);

    const toggle = document.getElementById('dashboards-toggle');
    const icon = toggle.querySelector('.dashboards-toggle-icon');
    const expanded = localStorage.getItem('dashboardsExpanded') === 'true';

    if (expanded) {
        list.style.display = '';
        icon.textContent = '−';
    }

    toggle.addEventListener('click', () => {
        const isHidden = list.style.display === 'none';
        list.style.display = isHidden ? '' : 'none';
        icon.textContent = isHidden ? '−' : '+';
        localStorage.setItem('dashboardsExpanded', isHidden);
    });
}
// ============ Feedback ============
// Free-text feedback per report. Auth required. Stored in SQLite on the server.
// Writes are append-only; no edit/delete in this iteration.

const FEEDBACK_STATE = {
    reportId: null,
    sectionId: null,   // empty string = report-level; non-empty = section-scoped
    nextCursor: 0,
    hasMore: false,
    loading: false,
};

async function feedbackAuthHeaders(extra) {
    const headers = Object.assign({}, extra || {});
    if (typeof window.getAuthToken === 'function') {
        try {
            const token = await window.getAuthToken();
            if (token) headers['Authorization'] = `Bearer ${token}`;
        } catch (e) {
            console.warn('[Feedback] token fetch failed:', e);
        }
    }
    return headers;
}

function openFeedbackModal() {
    if (!currentReportId) return;
    const modal = document.getElementById('feedback-modal');
    if (!modal) return;

    FEEDBACK_STATE.reportId = currentReportId;
    FEEDBACK_STATE.sectionId = '';
    FEEDBACK_STATE.nextCursor = 0;
    FEEDBACK_STATE.hasMore = false;

    const reportLabel = document.getElementById('feedback-modal-report');
    if (reportLabel) {
        const meta = allReports.find(r => r.id === currentReportId);
        reportLabel.textContent = 'Report: ' + (meta?.title || currentReportId);
    }

    const textarea = document.getElementById('feedback-comment');
    if (textarea) {
        textarea.value = '';
        updateFeedbackCounter();
        textarea.oninput = updateFeedbackCounter;
    }

    const errEl = document.getElementById('feedback-form-error');
    if (errEl) errEl.textContent = '';

    const listEl = document.getElementById('feedback-list');
    if (listEl) listEl.innerHTML = '';
    const countEl = document.getElementById('feedback-list-count');
    if (countEl) countEl.textContent = '';
    const moreBtn = document.getElementById('feedback-load-more');
    if (moreBtn) moreBtn.style.display = 'none';

    modal.style.display = 'flex';
    document.body.style.overflow = 'hidden';

    if (textarea) setTimeout(() => textarea.focus(), 50);

    fetchFeedback(true);
}

function closeFeedbackModal() {
    const modal = document.getElementById('feedback-modal');
    if (modal) modal.style.display = 'none';
    document.body.style.overflow = '';
}

function openSectionFeedbackModal(sectionTitle) {
    if (!currentReportId) return;
    const modal = document.getElementById('feedback-modal');
    if (!modal) return;

    FEEDBACK_STATE.reportId = currentReportId;
    FEEDBACK_STATE.sectionId = sectionTitle || '';
    FEEDBACK_STATE.nextCursor = 0;
    FEEDBACK_STATE.hasMore = false;

    const reportLabel = document.getElementById('feedback-modal-report');
    if (reportLabel) {
        const meta = allReports.find(r => r.id === currentReportId);
        const reportTitle = meta?.title || currentReportId;
        reportLabel.textContent = sectionTitle
            ? `Report: ${reportTitle} › ${sectionTitle}`
            : `Report: ${reportTitle}`;
    }

    const textarea = document.getElementById('feedback-comment');
    if (textarea) {
        textarea.value = '';
        updateFeedbackCounter();
        textarea.oninput = updateFeedbackCounter;
    }

    const errEl = document.getElementById('feedback-form-error');
    if (errEl) errEl.textContent = '';

    const listEl = document.getElementById('feedback-list');
    if (listEl) listEl.innerHTML = '';
    const countEl = document.getElementById('feedback-list-count');
    if (countEl) countEl.textContent = '';
    const moreBtn = document.getElementById('feedback-load-more');
    if (moreBtn) moreBtn.style.display = 'none';

    modal.style.display = 'flex';
    document.body.style.overflow = 'hidden';

    if (textarea) setTimeout(() => textarea.focus(), 50);

    fetchFeedback(true);
}

function updateFeedbackCounter() {
    const textarea = document.getElementById('feedback-comment');
    const counter = document.getElementById('feedback-counter');
    if (!textarea || !counter) return;
    const len = textarea.value.length;
    counter.textContent = `${len} / 4000`;
    counter.classList.toggle('is-over', len > 4000);
}

async function submitFeedback(event) {
    event.preventDefault();
    const errEl = document.getElementById('feedback-form-error');
    const submitBtn = document.getElementById('feedback-submit');
    const textarea = document.getElementById('feedback-comment');
    if (!textarea || !currentReportId) return;

    const comment = textarea.value.trim();
    if (errEl) errEl.textContent = '';
    if (!comment) {
        if (errEl) errEl.textContent = 'Please enter a comment.';
        return;
    }
    if (comment.length > 4000) {
        if (errEl) errEl.textContent = 'Comment is too long (max 4000 characters).';
        return;
    }

    if (submitBtn) submitBtn.disabled = true;
    try {
        const headers = await feedbackAuthHeaders({ 'Content-Type': 'application/json' });
        const body = { comment };
        if (FEEDBACK_STATE.sectionId) body.sectionId = FEEDBACK_STATE.sectionId;
        const resp = await fetch(`${API_BASE}/feedback/${encodeURI(currentReportId)}`, {
            method: 'POST',
            headers,
            body: JSON.stringify(body),
        });
        if (!resp.ok) {
            let msg = `HTTP ${resp.status}`;
            try {
                const body = await resp.json();
                if (body.error) msg = body.error;
            } catch (e) {}
            throw new Error(msg);
        }
        textarea.value = '';
        updateFeedbackCounter();
        // Refresh list from the top so the new entry is visible.
        FEEDBACK_STATE.nextCursor = 0;
        await fetchFeedback(true);
    } catch (err) {
        console.error('[Feedback] submit failed:', err);
        if (errEl) errEl.textContent = 'Could not submit: ' + (err.message || 'unknown error');
    } finally {
        if (submitBtn) submitBtn.disabled = false;
    }
}

async function fetchFeedback(reset) {
    if (FEEDBACK_STATE.loading || !FEEDBACK_STATE.reportId) return;
    FEEDBACK_STATE.loading = true;
    try {
        const params = new URLSearchParams({ limit: '50' });
        if (!reset && FEEDBACK_STATE.nextCursor > 0) {
            params.set('before', String(FEEDBACK_STATE.nextCursor));
        }
        if (FEEDBACK_STATE.sectionId) {
            params.set('sectionId', FEEDBACK_STATE.sectionId);
        }
        const url = `${API_BASE}/feedback/${encodeURI(FEEDBACK_STATE.reportId)}?${params.toString()}`;
        const headers = await feedbackAuthHeaders();
        const resp = await fetch(url, { headers });
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        const data = await resp.json();
        const entries = data.entries || [];
        renderFeedbackEntries(entries, reset);
        FEEDBACK_STATE.hasMore = !!data.hasMore;
        FEEDBACK_STATE.nextCursor = data.nextCursor || 0;

        const moreBtn = document.getElementById('feedback-load-more');
        if (moreBtn) moreBtn.style.display = FEEDBACK_STATE.hasMore ? '' : 'none';
    } catch (err) {
        console.error('[Feedback] list failed:', err);
        const listEl = document.getElementById('feedback-list');
        if (listEl && reset) {
            listEl.innerHTML = '';
            const p = document.createElement('div');
            p.className = 'feedback-list-empty';
            p.textContent = 'Could not load feedback.';
            listEl.appendChild(p);
        }
    } finally {
        FEEDBACK_STATE.loading = false;
    }
}

function loadMoreFeedback() {
    fetchFeedback(false);
}

function renderFeedbackEntries(entries, reset) {
    const listEl = document.getElementById('feedback-list');
    const countEl = document.getElementById('feedback-list-count');
    if (!listEl) return;

    if (reset) listEl.innerHTML = '';

    if (reset && entries.length === 0) {
        const p = document.createElement('div');
        p.className = 'feedback-list-empty';
        p.textContent = 'No feedback yet. Be the first.';
        listEl.appendChild(p);
        if (countEl) countEl.textContent = '';
        return;
    }

    for (const e of entries) {
        const row = document.createElement('div');
        row.className = 'feedback-entry';

        const meta = document.createElement('div');
        meta.className = 'feedback-entry-meta';
        const user = document.createElement('span');
        user.className = 'feedback-entry-user';
        user.textContent = e.username || 'anonymous';
        const when = document.createElement('span');
        when.className = 'feedback-entry-when';
        when.textContent = formatFeedbackTimestamp(e.createdAt);
        meta.appendChild(user);
        meta.appendChild(when);

        // Show section badge only in report-level view where entries may span sections
        if (!FEEDBACK_STATE.sectionId && e.sectionId) {
            const sectionBadge = document.createElement('span');
            sectionBadge.className = 'feedback-entry-section';
            sectionBadge.textContent = e.sectionId;
            row.appendChild(sectionBadge);
        }

        const body = document.createElement('div');
        body.className = 'feedback-entry-comment';
        body.textContent = e.comment; // textContent = XSS-safe

        row.appendChild(meta);
        row.appendChild(body);
        listEl.appendChild(row);
    }

    if (countEl) {
        const n = listEl.querySelectorAll('.feedback-entry').length;
        countEl.textContent = FEEDBACK_STATE.hasMore ? `${n}+ shown` : `${n} total`;
    }
}

function formatFeedbackTimestamp(unixSeconds) {
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

// Expose for inline onclick handlers in index.html
window.openFeedbackModal = openFeedbackModal;
window.openSectionFeedbackModal = openSectionFeedbackModal;
window.closeFeedbackModal = closeFeedbackModal;
window.submitFeedback = submitFeedback;
window.loadMoreFeedback = loadMoreFeedback;

// Close feedback modal on Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        const modal = document.getElementById('feedback-modal');
        if (modal && modal.style.display === 'flex') closeFeedbackModal();
    }
});
