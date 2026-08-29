// Keywords Suggestions Module
// Provides domain keyword suggestions for report discoverability

// Keyword suggestions organized by common data-warehouse domains.
// This is intentionally sector-neutral so any StatGate dataset can use it.
export const KEYWORD_SUGGESTIONS = {
    // Demographics & Population
    'demographics': [
        'population', 'age group', 'gender', 'sex disaggregation',
        'household', 'district', 'region', 'national', 'residence'
    ],

    // Operations & Performance
    'operations': [
        'operations', 'service delivery', 'workflow', 'throughput',
        'response time', 'performance', 'utilisation', 'capacity',
        'admissions', 'referrals', 'visits', 'consultations',
        'procedures', 'services'
    ],

    // Finance & Budget
    'finance': [
        'revenue', 'expenditure', 'budget', 'expenditure tracking',
        'procurement', 'spend', 'cost', 'funding', 'allocation',
        'disbursement', 'variance', 'forecast'
    ],

    // Workforce & HR
    'workforce': [
        'staff', 'personnel', 'workforce', 'headcount', 'hiring',
        'attrition', 'absenteism', 'trainings', 'certifications',
        'attendance', 'skills'
    ],

    // Infrastructure & Assets
    'infrastructure': [
        'facilities', 'infrastructure', 'assets', 'equipment',
        'vehicles', 'buildings', 'maintenance', 'inventory',
        'utilities', 'connectivity'
    ],

    // Education & Learning
    'learning': [
        'enrollment', 'attendance', 'graduation', 'curriculum',
        'assessment', 'results', 'performance', 'tutoring',
        'scholarships', 'outcomes'
    ],

    // Agriculture & Environment
    'agri-environment': [
        'crops', 'yield', 'harvest', 'livestock', 'inputs',
        'irrigation', 'weather', 'soil', 'production', 'market',
        'environment', 'climate', 'emissions'
    ],

    // Economy & Trade
    'economy': [
        'gdp', 'inflation', 'trade', 'imports', 'exports', 'prices',
        'taxes', 'revenue', 'markets', 'enterprise', 'investment'
    ],

    // Governance & Compliance
    'governance': [
        'compliance', 'policies', 'regulation', 'audit', 'oversight',
        'permits', 'licences', 'risk', 'transparency', 'accountability',
        'cases', 'complaints', 'resolution'
    ],

    // Grants & Programmes
    'programmes': [
        'program', 'project', 'grant', 'donor', 'beneficiary',
        'coverage', 'uptake', 'outreach', 'disbursement',
        'evaluation', 'interventions'
    ],

    // Time & Trends
    'trends': [
        'trend', 'monthly', 'quarterly', 'annual', 'seasonality',
        'comparison', 'growth', 'change over time', 'forecast',
        'baseline', 'target', 'variance'
    ],

    // General/Default
    'default': [
        'analytics', 'reporting', 'monitoring', 'indicators',
        'performance', 'statistics', 'trends', 'analysis'
    ]
};

// Get suggestions based on report category or title
export function getSuggestionsForCategory(category, title) {
    const categoryLower = (category || '').toLowerCase();
    const titleLower = (title || '').toLowerCase();
    const combined = categoryLower + ' ' + titleLower;

    let suggestions = new Set();

    // Check each keyword category for matches
    for (const [key, keywords] of Object.entries(KEYWORD_SUGGESTIONS)) {
        if (key === 'default') continue;

        if (combined.includes(key)) {
            keywords.forEach(kw => suggestions.add(kw));
        }
    }

    // If no specific matches, use default suggestions
    if (suggestions.size === 0) {
        KEYWORD_SUGGESTIONS.default.forEach(kw => suggestions.add(kw));
    }

    // Always add some general ones
    ['analytics', 'reporting', 'monitoring'].forEach(kw => suggestions.add(kw));

    return Array.from(suggestions).slice(0, 15); // Limit to 15 suggestions
}

// Parse current keywords from input
export function parseKeywords(keywordsStr) {
    if (!keywordsStr) return [];
    return keywordsStr.split(',')
        .map(k => k.trim().toLowerCase())
        .filter(k => k.length > 0);
}

// Format keywords for display/storage
export function formatKeywords(keywordsArray) {
    return keywordsArray.join(', ');
}

// Initialize keyword suggestions UI
export function initKeywordSuggestions(inputEl, suggestionsContainerEl, getCategoryFn) {
    if (!inputEl || !suggestionsContainerEl) return;

    const chipsContainer = suggestionsContainerEl.querySelector('#suggestion-chips');
    if (!chipsContainer) return;

    // Render suggestion chips
    function renderSuggestions() {
        const category = getCategoryFn ? getCategoryFn() : '';
        const title = document.getElementById('report-title')?.value || '';
        const suggestions = getSuggestionsForCategory(category, title);
        const currentKeywords = parseKeywords(inputEl.value);

        chipsContainer.innerHTML = suggestions.map(keyword => {
            const isSelected = currentKeywords.includes(keyword.toLowerCase());
            return `<span class="keyword-chip ${isSelected ? 'selected' : ''}" data-keyword="${keyword}">${keyword}</span>`;
        }).join('');

        // Add click handlers
        chipsContainer.querySelectorAll('.keyword-chip').forEach(chip => {
            chip.addEventListener('click', () => {
                const keyword = chip.dataset.keyword;
                toggleKeyword(keyword);
            });
        });
    }

    // Toggle a keyword on/off
    function toggleKeyword(keyword) {
        const currentKeywords = parseKeywords(inputEl.value);
        const keywordLower = keyword.toLowerCase();

        if (currentKeywords.includes(keywordLower)) {
            // Remove keyword
            const newKeywords = currentKeywords.filter(k => k !== keywordLower);
            inputEl.value = formatKeywords(newKeywords);
        } else {
            // Add keyword
            currentKeywords.push(keywordLower);
            inputEl.value = formatKeywords(currentKeywords);
        }

        // Trigger input event
        inputEl.dispatchEvent(new Event('input', { bubbles: true }));
        renderSuggestions();
    }

    // Re-render when input changes
    inputEl.addEventListener('input', () => {
        renderSuggestions();
    });

    // Re-render when title changes (affects suggestions)
    const titleInput = document.getElementById('report-title');
    if (titleInput) {
        titleInput.addEventListener('input', () => {
            renderSuggestions();
        });
    }

    // Initial render
    renderSuggestions();

    return { renderSuggestions, toggleKeyword };
}
