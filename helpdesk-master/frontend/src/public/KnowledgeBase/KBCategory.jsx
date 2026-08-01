import React from 'react'
import { Link } from 'react-router-dom';
import './styles-category.css';

const KBCategory = () => {
    return (
        <div>
            <div className="content-area">
                <nav className="breadcrumb-nav">
                    <ol className="breadcrumb">
                        <li className="breadcrumb-item"><Link to="/public/knowledgeBase">Knowledge Base</Link></li>
                        <li className="breadcrumb-item active">Analytics Operations</li>
                    </ol>
                </nav>

                <div className="category-header">
                    <h1 className="category-title">
                        <i className="fas fa-desktop me-3" style={{ color: 'var(--primary-blue)' }}></i>
                        Analytics Operations
                    </h1>
                    <p className="category-description">
                        Guides and playbooks for dashboards, data pipelines, quality checks, and rapid incident response.
                    </p>
                    <div className="category-stats">
                        <span><i className="fas fa-file-alt me-1"></i> 24 Articles</span>
                        <span><i className="fas fa-eye me-1"></i> 1,247 Views</span>
                        <span><i className="fas fa-clock me-1"></i> Last updated 2 days ago</span>
                    </div>
                </div>

                <div className="articles-container">
                    <div className="articles-main">
                        <div className="article-card">
                            <Link to="/public/kb/article" className="article-title">Dashboard refresh failure - triage playbook</Link>
                            <p className="article-excerpt">
                                A step-by-step guide to identify stalled refresh jobs, inspect source connectivity, and restore reporting access quickly.
                            </p>
                            <div className="article-meta">
                                <div className="article-tags">
                                    <span className="article-tag">Troubleshooting</span>
                                    <span className="article-tag">Incident Response</span>
                                </div>
                                <div className="helpful-stats">
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-up text-success"></i>
                                        <span>15</span>
                                    </div>
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-down text-danger"></i>
                                        <span>2</span>
                                    </div>
                                    <span>Updated 3 days ago</span>
                                </div>
                            </div>
                        </div>

                        <div className="article-card">
                            <Link to="/public/kb/article" className="article-title">Access request workflow for new analysts</Link>
                            <p className="article-excerpt">
                                Recommended steps for onboarding analysts, assigning roles, and verifying permissions before first use.
                            </p>
                            <div className="article-meta">
                                <div className="article-tags">
                                    <span className="article-tag">Access</span>
                                    <span className="article-tag">Onboarding</span>
                                </div>
                                <div className="helpful-stats">
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-up text-success"></i>
                                        <span>23</span>
                                    </div>
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-down text-danger"></i>
                                        <span>1</span>
                                    </div>
                                    <span>Updated 1 week ago</span>
                                </div>
                            </div>
                        </div>

                        <div className="article-card">
                            <Link to="/public/kb/article" className="article-title">Data quality checks before publication</Link>
                            <p className="article-excerpt">
                                Essential checks for missing values, duplicate records, and stale extracts before a report or dashboard goes live.
                            </p>
                            <div className="article-meta">
                                <div className="article-tags">
                                    <span className="article-tag">Data Quality</span>
                                    <span className="article-tag">Validation</span>
                                </div>
                                <div className="helpful-stats">
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-up text-success"></i>
                                        <span>31</span>
                                    </div>
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-down text-danger"></i>
                                        <span>0</span>
                                    </div>
                                    <span>Updated 2 weeks ago</span>
                                </div>
                            </div>
                        </div>

                        <div className="article-card">
                            <Link to="/public/kb/article" className="article-title">Performance tuning for live dashboards</Link>
                            <p className="article-excerpt">
                                Best practices for optimizing refresh speed, reducing load times, and keeping executive dashboards responsive.
                            </p>
                            <div className="article-meta">
                                <div className="article-tags">
                                    <span className="article-tag">Performance</span>
                                    <span className="article-tag">Optimization</span>
                                </div>
                                <div className="helpful-stats">
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-up text-success"></i>
                                        <span>18</span>
                                    </div>
                                    <div className="helpful-item">
                                        <i className="fas fa-thumbs-down text-danger"></i>
                                        <span>3</span>
                                    </div>
                                    <span>Updated 3 weeks ago</span>
                                </div>
                            </div>
                        </div>

                        <div className="pagination-section">
                            <nav>
                                <ul className="pagination">
                                    <li className="page-item disabled">
                                        <span className="page-link">Previous</span>
                                    </li>
                                    <li className="page-item active">
                                        <span className="page-link">1</span>
                                    </li>
                                    <li className="page-item">
                                        <a className="page-link" href="#">2</a>
                                    </li>
                                    <li className="page-item">
                                        <a className="page-link" href="#">3</a>
                                    </li>
                                    <li className="page-item">
                                        <a className="page-link" href="#">Next</a>
                                    </li>
                                </ul>
                            </nav>
                        </div>
                    </div>

                    <div className="articles-sidebar">
                        <div className="filter-section">
                            <h3 className="filter-title">Filter by Tags</h3>
                            <div className="filter-item">
                                <span>Troubleshooting</span>
                                <span className="filter-count">8</span>
                            </div>
                            <div className="filter-item">
                                <span>Access</span>
                                <span className="filter-count">5</span>
                            </div>
                            <div className="filter-item">
                                <span>Performance</span>
                                <span className="filter-count">4</span>
                            </div>
                            <div className="filter-item">
                                <span>Data Quality</span>
                                <span className="filter-count">3</span>
                            </div>
                            <div className="filter-item">
                                <span>Configuration</span>
                                <span className="filter-count">4</span>
                            </div>
                        </div>

                        <div className="filter-section">
                            <h3 className="filter-title">Most Helpful</h3>
                            <div className="filter-item">
                                <span>Dashboard triage steps</span>
                                <span className="filter-count">31</span>
                            </div>
                            <div className="filter-item">
                                <span>Access review checklist</span>
                                <span className="filter-count">23</span>
                            </div>
                            <div className="filter-item">
                                <span>Performance tuning tips</span>
                                <span className="filter-count">18</span>
                            </div>
                        </div>

                        <div className="filter-section">
                            <h3 className="filter-title">Related Categories</h3>
                            <div className="filter-item">
                                <span>Alerting</span>
                                <span className="filter-count">12</span>
                            </div>
                            <div className="filter-item">
                                <span>Governance</span>
                                <span className="filter-count">9</span>
                            </div>
                            <div className="filter-item">
                                <span>Reporting</span>
                                <span className="filter-count">7</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default KBCategory