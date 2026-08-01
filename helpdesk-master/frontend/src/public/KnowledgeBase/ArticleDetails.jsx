import React from 'react'
import { Link } from 'react-router-dom';
import './styles-article.css'

const ArticleDetails = () => {
    return (
        <div>
            <div class="content-area">
                <nav class="breadcrumb-nav">
                    <ol class="breadcrumb">
                        <li class="breadcrumb-item"><Link to="/public/kb/category">Knowledge Base</Link></li>
                        <li class="breadcrumb-item"><a href="kb-category.html">Analytics Operations</a></li>
                        <li class="breadcrumb-item active">Dashboard refresh failure</li>
                    </ol>
                </nav>

                <div class="article-container">
                    <div class="article-main">
                        <div class="article-header">
                            <h1 class="article-title">Dashboard refresh failure - triage playbook</h1>
                            <div class="article-meta">
                                <span><i class="fas fa-calendar me-1"></i> Updated 3 days ago</span>
                                <span><i class="fas fa-user me-1"></i> By Operations Team</span>
                                <span><i class="fas fa-eye me-1"></i> 342 views</span>
                                <span><i class="fas fa-clock me-1"></i> 8 min read</span>
                            </div>
                            <div class="article-tags">
                                <span class="article-tag">Troubleshooting</span>
                                <span class="article-tag">Incident Response</span>
                                <span class="article-tag">Dashboard</span>
                            </div>
                        </div>

                        <div class="article-content">
                            <p>When a dashboard or report refresh stalls, the impact can be immediate. This guide walks the operations team through a quick triage path so the right people can respond in minutes.</p>

                            <div class="warning-box">
                                <strong><i class="fas fa-exclamation-triangle me-2"></i>Important:</strong> Start by confirming the scope of the issue before making changes to the data source or refresh schedule.
                            </div>

                            <h2>Quick triage checklist</h2>
                            <p>Start with these checks to identify the scope of the issue:</p>
                            <ul>
                                <li>Is the issue affecting all dashboards or one specific view?</li>
                                <li>Did the refresh fail for a single source or all sources?</li>
                                <li>Are other scheduled jobs operating normally?</li>
                                <li>When did the issue first appear?</li>
                            </ul>

                            <h2>Step 1: Confirm the source connection</h2>
                            <p>Use the connector health view to verify whether the source system is reachable and passing credentials.</p>

                            <h3>Basic checks</h3>
                            <ol>
                                <li>Open the connector status page</li>
                                <li>Confirm last successful sync time</li>
                                <li>Review recent authentication or access errors</li>
                            </ol>

                            <div class="info-box">
                                <strong><i class="fas fa-info-circle me-2"></i>Tip:</strong> A red connector state often points to expiring credentials or a temporary outage before any dashboard issue is visible.
                            </div>

                            <h2>Step 2: Verify data quality rules</h2>
                            <p>Broken rules or schema drift can stop a refresh even when the source remains online.</p>

                            <h3>Checks to run</h3>
                            <ul>
                                <li>Inspect recent validation failures</li>
                                <li>Verify custom transformations did not break</li>
                                <li>Check whether expected columns are still present</li>
                            </ul>

                            <h2>Step 3: Review the job timeline</h2>
                            <p>Look at the latest execution logs for clues about timing, retries, or memory pressure.</p>

                            <h3>What to review</h3>
                            <div class="code-block">
                                Last successful run
                                Retry count
                                Duration of the failed step
                                Time-outs or connection resets
                            </div>

                            <h2>Step 4: Escalate with context</h2>
                            <p>If the issue is not resolved quickly, escalate with the impact, affected users, and planned recovery steps.</p>

                            <div class="warning-box">
                                <strong><i class="fas fa-exclamation-triangle me-2"></i>Warning:</strong> Capture timestamps, error messages, and the scope of affected users before hand-off.
                            </div>

                            <h2>Prevention and monitoring</h2>
                            <p>Use these habits to reduce future interruptions:</p>
                            <ul>
                                <li>Set alerts for refresh failures and connector health changes</li>
                                <li>Maintain a documented backup schedule for critical datasets</li>
                                <li>Review performance trends weekly</li>
                                <li>Document known data-quality issues and owners</li>
                            </ul>
                        </div>

                        <div class="article-footer">
                            <div class="helpful-section">
                                <div class="helpful-question">Was this article helpful?</div>
                                <div class="helpful-buttons">
                                    <button class="helpful-btn yes">
                                        <i class="fas fa-thumbs-up me-2"></i>Yes (15)
                                    </button>
                                    <button class="helpful-btn no">
                                        <i class="fas fa-thumbs-down me-2"></i>No (2)
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="article-sidebar">
                        <div class="panel-section">
                            <div class="panel-header">
                                <span class="panel-title">
                                    <i class="fas fa-list me-2"></i>
                                    Table of Contents
                                </span>
                                <i class="fas fa-chevron-up"></i>
                            </div>
                            <div class="panel-content">
                                <div class="toc-item"><a href="#" class="toc-link">Quick triage checklist</a></div>
                                <div class="toc-item"><a href="#" class="toc-link">Source connection</a></div>
                                <div class="toc-item"><a href="#" class="toc-link">Data quality rules</a></div>
                                <div class="toc-item"><a href="#" class="toc-link">Job timeline</a></div>
                                <div class="toc-item"><a href="#" class="toc-link">Escalation</a></div>
                            </div>
                        </div>

                        <div class="panel-section">
                            <div class="panel-header">
                                <span class="panel-title">
                                    <i class="fas fa-link me-2"></i>
                                    Related Articles
                                </span>
                                <i class="fas fa-chevron-up"></i>
                            </div>
                            <div class="panel-content">
                                <div class="related-article"><a href="#" class="related-title">Access request workflow for new analysts</a><div class="related-meta">Updated 1 week ago • 5 min read</div></div>
                                <div class="related-article"><a href="#" class="related-title">Performance tuning for live dashboards</a><div class="related-meta">Updated 3 weeks ago • 7 min read</div></div>
                                <div class="related-article"><a href="#" class="related-title">Data quality checks before publication</a><div class="related-meta">Updated 2 weeks ago • 12 min read</div></div>
                            </div>
                        </div>

                        <div class="panel-section">
                            <div class="panel-header">
                                <span class="panel-title">
                                    <i class="fas fa-tags me-2"></i>
                                    Article Tags
                                </span>
                                <i class="fas fa-chevron-up"></i>
                            </div>
                            <div class="panel-content">
                                <div class="article-tags">
                                    <span class="article-tag">Troubleshooting</span>
                                    <span class="article-tag">Dashboard</span>
                                    <span class="article-tag">Connectivity</span>
                                    <span class="article-tag">Data Quality</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ArticleDetails