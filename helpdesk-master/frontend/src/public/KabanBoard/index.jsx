import React from 'react'
import './styles.css'

const Kaban = () => {
    return (
        <div>
            <div class="content-area">
                <nav class="breadcrumb-nav">
                    <ol class="breadcrumb">
                        <li class="breadcrumb-item"><a href="index.html">Operations</a></li>
                        <li class="breadcrumb-item active">Work Queue Board</li>
                    </ol>
                </nav>

                <div class="kanban-header">
                    <h2 class="kanban-title">Operations Work Queue</h2>
                    <div class="kanban-filters">
                        <select class="filter-select">
                            <option>All Analysts</option>
                            <option>Grace A.</option>
                            <option>John K.</option>
                            <option>Sarah N.</option>
                        </select>
                        <select class="filter-select">
                            <option>All Severity</option>
                            <option>Critical</option>
                            <option>High</option>
                            <option>Medium</option>
                        </select>
                        <select class="filter-select">
                            <option>All Workstreams</option>
                            <option>Analytics</option>
                            <option>Data Quality</option>
                            <option>Reporting</option>
                        </select>
                    </div>
                </div>

                <div class="kanban-board">
                    <div class="kanban-column column-new">
                        <div class="column-header">
                            <div class="column-title">
                                <i class="fas fa-plus-circle"></i>
                                New
                            </div>
                            <span class="column-count">3</span>
                        </div>
                        <div class="column-body" data-status="new">
                            <div class="ticket-card" draggable="true" data-ticket-id="101">
                                <div class="ticket-id">#101</div>
                                <div class="ticket-subject">Dashboard refresh failing for district reports</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-high">High</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">GA</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Analytics</span>
                                    <span class="ticket-time">2 hours ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="102">
                                <div class="ticket-id">#102</div>
                                <div class="ticket-subject">Access request pending for new analyst</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-medium">Medium</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">JK</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Access Management</span>
                                    <span class="ticket-time">4 hours ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="103">
                                <div class="ticket-id">#103</div>
                                <div class="ticket-subject">Alert threshold update needed for weekly monitoring</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-low">Low</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">SN</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Alerting</span>
                                    <span class="ticket-time">1 day ago</span>
                                </div>
                            </div>

                            <button class="add-ticket-btn">
                                <i class="fas fa-plus"></i> Add New Work Item
                            </button>
                        </div>
                    </div>

                    <div class="kanban-column column-open">
                        <div class="column-header">
                            <div class="column-title">
                                <i class="fas fa-folder-open"></i>
                                In Progress
                            </div>
                            <span class="column-count">5</span>
                        </div>
                        <div class="column-body" data-status="open">
                            <div class="ticket-card" draggable="true" data-ticket-id="98">
                                <div class="ticket-id">#98</div>
                                <div class="ticket-subject">Daily extraction delay from source system</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-high">High</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">GA</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Data Pipeline</span>
                                    <span class="ticket-time">3 hours ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="99">
                                <div class="ticket-id">#99</div>
                                <div class="ticket-subject">Missing facility codes in reporting export</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-medium">Medium</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">JK</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Data Quality</span>
                                    <span class="ticket-time">5 hours ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="100">
                                <div class="ticket-id">#100</div>
                                <div class="ticket-subject">Regional summary update blocked by validation rule</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-medium">Medium</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">SN</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Governance</span>
                                    <span class="ticket-time">6 hours ago</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="kanban-column column-pending">
                        <div class="column-header">
                            <div class="column-title">
                                <i class="fas fa-clock"></i>
                                Pending
                            </div>
                            <span class="column-count">2</span>
                        </div>
                        <div class="column-body" data-status="pending">
                            <div class="ticket-card" draggable="true" data-ticket-id="95">
                                <div class="ticket-id">#95</div>
                                <div class="ticket-subject">Waiting for stakeholder confirmation</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-medium">Medium</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">GA</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Reporting</span>
                                    <span class="ticket-time">2 days ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="96">
                                <div class="ticket-id">#96</div>
                                <div class="ticket-subject">Training material review pending</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-low">Low</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">JK</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Training</span>
                                    <span class="ticket-time">3 days ago</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="kanban-column column-resolved">
                        <div class="column-header">
                            <div class="column-title">
                                <i class="fas fa-check-circle"></i>
                                Resolved
                            </div>
                            <span class="column-count">4</span>
                        </div>
                        <div class="column-body" data-status="resolved">
                            <div class="ticket-card" draggable="true" data-ticket-id="92">
                                <div class="ticket-id">#92</div>
                                <div class="ticket-subject">Password reset completed for analytics users</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-low">Low</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">SN</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Access</span>
                                    <span class="ticket-time">1 day ago</span>
                                </div>
                            </div>

                            <div class="ticket-card" draggable="true" data-ticket-id="93">
                                <div class="ticket-id">#93</div>
                                <div class="ticket-subject">Weekly report template corrected</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-medium">Medium</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">GA</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Reporting</span>
                                    <span class="ticket-time">2 days ago</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="kanban-column column-closed">
                        <div class="column-header">
                            <div class="column-title">
                                <i class="fas fa-times-circle"></i>
                                Closed
                            </div>
                            <span class="column-count">6</span>
                        </div>
                        <div class="column-body" data-status="closed">
                            <div class="ticket-card" draggable="true" data-ticket-id="88">
                                <div class="ticket-id">#88</div>
                                <div class="ticket-subject">Alert routing restored after incident</div>
                                <div class="ticket-meta">
                                    <span class="ticket-priority priority-high">High</span>
                                    <div class="ticket-assignee">
                                        <div class="assignee-avatar">JK</div>
                                    </div>
                                </div>
                                <div class="ticket-footer">
                                    <span class="ticket-category">Operations</span>
                                    <span class="ticket-time">1 week ago</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Kaban;