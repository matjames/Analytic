import React, { Fragment } from 'react'
import { useHistory } from 'react-router-dom';
import './styles.css';

const KnowledgeBase = () => {
    const history = useHistory();

    const handleCategoryClick = () => {
        history.push('/public/kb/category');
    }

    return (
        <Fragment>
            <div class="content-area">
                <nav class="breadcrumb-nav">
                    <ol class="breadcrumb">
                        <li class="breadcrumb-item"><a href="index.html">Operations</a></li>
                        <li class="breadcrumb-item active">Knowledge Base</li>
                    </ol>
                </nav>

                <div class="page-header">
                    <h1 class="page-title">
                        <i class="fas fa-book me-2"></i>
                        Knowledge Base
                    </h1>
                    <p class="page-subtitle">Find playbooks, runbooks, and workflows for the StatGate analytics operations team.</p>
                </div>

                <div class="search-section">
                    <div class="search-bar">
                        <input type="text" class="search-input" placeholder="Search for guides, workflows, or alerts..." />
                        <i class="fas fa-search search-icon"></i>
                    </div>
                    <div class="filter-tabs">
                        <a href="#" class="filter-tab active">All Articles</a>
                        <a href="#" class="filter-tab">Getting Started</a>
                        <a href="#" class="filter-tab">Workflows</a>
                        <a href="#" class="filter-tab">Access</a>
                        <a href="#" class="filter-tab">Data Quality</a>
                        <a href="#" class="filter-tab">Reports & Dashboards</a>
                    </div>
                </div>

                <div class="kb-content">
                    <div class="kb-main">
                        <div class="category-section">
                            <div class="section-header">
                                <h2 class="section-title">Browse by Category</h2>
                                <span class="text-muted">6 categories</span>
                            </div>

                            <div class="category-grid">
                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-clipboard-list"></i>
                                    </div>
                                    <h3 class="category-name">Operations Workflows</h3>
                                    <p class="category-description">Step-by-step playbooks for incident triage, escalation, and handoff.</p>
                                    <span class="category-count">15 articles</span>
                                </div>

                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-users"></i>
                                    </div>
                                    <h3 class="category-name">Access Requests</h3>
                                    <p class="category-description">Guides for onboarding analysts, permissions, and role changes.</p>
                                    <span class="category-count">8 articles</span>
                                </div>

                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-cog"></i>
                                    </div>
                                    <h3 class="category-name">Configuration</h3>
                                    <p class="category-description">How to tune dashboards, alerts, and data-source settings.</p>
                                    <span class="category-count">12 articles</span>
                                </div>

                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-chart-bar"></i>
                                    </div>
                                    <h3 class="category-name">Reports & Dashboards</h3>
                                    <p class="category-description">Templates and best practices for daily and weekly reporting.</p>
                                    <span class="category-count">6 articles</span>
                                </div>

                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-database"></i>
                                    </div>
                                    <h3 class="category-name">Data Quality</h3>
                                    <p class="category-description">Investigate missing values, failed loads, and field validation gaps.</p>
                                    <span class="category-count">9 articles</span>
                                </div>

                                <div class="category-card" onClick={handleCategoryClick} style={{ cursor: 'pointer' }}>
                                    <div class="category-icon">
                                        <i class="fas fa-question-circle"></i>
                                    </div>
                                    <h3 class="category-name">Troubleshooting</h3>
                                    <p class="category-description">Common platform issues and recommended response steps.</p>
                                    <span class="category-count">18 articles</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="kb-sidebar">
                        <div class="sidebar-section">
                            <div class="sidebar-header">
                                <span class="sidebar-title">Filter by Topic</span>
                            </div>
                            <div class="sidebar-content">
                                <div class="filter-item">
                                    <input type="checkbox" class="filter-checkbox" id="getting-started" />
                                    <label class="filter-label" for="getting-started">Getting Started</label>
                                    <span class="filter-count">12</span>
                                </div>
                                <div class="filter-item">
                                    <input type="checkbox" class="filter-checkbox" id="advanced" />
                                    <label class="filter-label" for="advanced">Advanced Analysis</label>
                                    <span class="filter-count">8</span>
                                </div>
                                <div class="filter-item">
                                    <input type="checkbox" class="filter-checkbox" id="api" />
                                    <label class="filter-label" for="api">API & Connectors</label>
                                    <span class="filter-count">15</span>
                                </div>
                                <div class="filter-item">
                                    <input type="checkbox" class="filter-checkbox" id="mobile" />
                                    <label class="filter-label" for="mobile">Mobile Reporting</label>
                                    <span class="filter-count">6</span>
                                </div>
                            </div>
                        </div>

                        <div class="sidebar-section">
                            <div class="sidebar-header">
                                <span class="sidebar-title">Popular Articles</span>
                            </div>
                            <div class="sidebar-content">
                                <ul class="popular-articles">
                                    <li class="popular-article">
                                        <a href="#" class="article-link">How to triage a dashboard incident</a>
                                        <div class="article-meta">Updated 2 days ago</div>
                                    </li>
                                    <li class="popular-article">
                                        <a href="#" class="article-link">Setting up alert thresholds</a>
                                        <div class="article-meta">Updated 1 week ago</div>
                                    </li>
                                    <li class="popular-article">
                                        <a href="#" class="article-link">Managing analyst permissions</a>
                                        <div class="article-meta">Updated 3 days ago</div>
                                    </li>
                                    <li class="popular-article">
                                        <a href="#" class="article-link">Connector authentication guide</a>
                                        <div class="article-meta">Updated 5 days ago</div>
                                    </li>
                                    <li class="popular-article">
                                        <a href="#" class="article-link">Investigating data quality gaps</a>
                                        <div class="article-meta">Updated 1 day ago</div>
                                    </li>
                                </ul>
                            </div>
                        </div>

                        <div class="sidebar-section">
                            <div class="sidebar-header">
                                <span class="sidebar-title">Need More Help?</span>
                            </div>
                            <div class="sidebar-content">
                                <p style={{ fontSize: '12px', color: 'var(--text-light)', marginBottom: '15px' }}>
                                    Need a second pair of eyes on an incident? Escalate to the operations team.
                                </p>
                                <button class="btn btn-primary btn-sm" style={{ width: '100%', borderRadius: '0' }}>
                                    Contact Operations
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </Fragment>
    );
};

export default KnowledgeBase