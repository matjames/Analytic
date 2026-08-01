import React, { useState, useEffect, Fragment } from "react";
import { Link } from 'react-router-dom'
import API from "../../helpers/api";
import './styles-details.css';

const TicketDetails = ({ match }) => {

    const [loading, setLoading] = useState(false);
    const [ticket, setTicket] = useState({});

    const { id } = match.params;

    const loadTicket = async () => {
        setLoading(true);
        try {
            const res = await API.get(`/t/tickets/${id}`);
            setTicket(res?.data.ticket);
            setLoading(false);
        } catch (error) {
            console.log("error", error);
            setLoading(false);
        }
    };

    useEffect(() => {
        loadTicket();
    }, []);

    return (
        <Fragment>
            <div class="content-area">
                <nav class="breadcrumb-nav">
                    <ol class="breadcrumb">
                        <li class="breadcrumb-item"><Link to="/public/tickets">Operations Work Items</Link></li>
                        <li class="breadcrumb-item active">Work Item Details (#101)</li>
                    </ol>
                </nav>

                <div class="ticket-container">
                    <div class="ticket-main">
                        <div class="ticket-header">
                            <div class="ticket-info">
                                <div class="ticket-id">Work Item #101</div>
                                <div class="ticket-subject">Dashboard refresh stalled for district reporting</div>
                                <div class="ticket-meta">
                                    <span><i class="fas fa-user"></i> Grace A.</span>
                                    <span><i class="fas fa-calendar"></i> Created: Today 2:20 PM</span>
                                    <span><i class="fas fa-clock"></i> Last Updated: Today 4:15 PM</span>
                                </div>
                            </div>
                            <div class="ticket-actions">
                                <span class="status-badge status-open">Open</span>
                                <span class="priority-badge priority-high">High</span>
                            </div>
                        </div>

                        <div class="ticket-description">
                            <h3 class="section-title">Summary</h3>
                            <div class="description-content">
                                <p>The dashboard refresh has stalled for the district reporting suite, leaving executives without the latest figures. The incident appears to affect all users accessing the analytics view.</p>
                                <p>Observed steps:</p>
                                <ol>
                                    <li>Open the dashboard from the analytics portal</li>
                                    <li>Run the refresh action</li>
                                    <li>Observe the spinner and timeout message</li>
                                </ol>
                                <p>The issue started around 1:30 PM and is blocking downstream reporting.</p>
                            </div>
                        </div>

                        <div class="comments-section">
                            <h3 class="section-title">Updates</h3>

                            <div class="comment-item">
                                <div class="comment-avatar">GA</div>
                                <div class="comment-content">
                                    <div class="comment-header">
                                        <span class="comment-author">Grace A.</span>
                                        <span class="comment-time">Today 2:20 PM</span>
                                        <span class="comment-type">Reporter</span>
                                    </div>
                                    <div class="comment-text">
                                        I submitted the work item after noticing the district dashboard remained on the previous refresh window.
                                    </div>
                                </div>
                            </div>

                            <div class="comment-item">
                                <div class="comment-avatar">JK</div>
                                <div class="comment-content">
                                    <div class="comment-header">
                                        <span class="comment-author">John K.</span>
                                        <span class="comment-time">Today 2:45 PM</span>
                                        <span class="comment-type">Analyst</span>
                                    </div>
                                    <div class="comment-text">
                                        The connector status page reports no recent sync. I am investigating the data-source credentials and timing window.
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="add-comment">
                            <h4 class="section-title">Add Update</h4>
                            <form class="comment-form">
                                <div class="comment-input">
                                    <textarea class="form-control" rows="3" placeholder="Type your update here..."></textarea>
                                </div>
                                <div class="d-flex flex-column gap-2">
                                    <button type="submit" class="btn btn-primary">
                                        <i class="fas fa-paper-plane"></i> Send Update
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>

                    <div class="ticket-sidebar">
                        <div class="panel-section">
                            <div class="panel-header">
                                <span class="panel-title">
                                    <i class="fas fa-cog me-2"></i>
                                    Work Item Properties
                                </span>
                                <i class="fas fa-chevron-up"></i>
                            </div>
                            <div class="ticket-properties">
                                <div class="property-item">
                                    <span class="property-label">Status</span>
                                    <span class="property-value">Open</span>
                                </div>
                                <div class="property-item">
                                    <span class="property-label">Severity</span>
                                    <span class="property-value">High</span>
                                </div>
                                <div class="property-item">
                                    <span class="property-label">Workstream</span>
                                    <span class="property-value">Analytics</span>
                                </div>
                                <div class="property-item">
                                    <span class="property-label">Assigned To</span>
                                    <span class="property-value">John K.</span>
                                </div>
                                <div class="property-item">
                                    <span class="property-label">Program</span>
                                    <span class="property-value">District Reporting</span>
                                </div>
                            </div>
                        </div>

                        <div class="panel-section">
                            <div class="panel-header">
                                <span class="panel-title">
                                    <i class="fas fa-user me-2"></i>
                                    Contact Information
                                </span>
                                <i class="fas fa-chevron-up"></i>
                            </div>
                            <div class="panel-content">
                                <div class="contact-info">
                                    <div class="contact-avatar">GA</div>
                                    <div class="contact-name">Grace A.</div>
                                    <div class="contact-details">District Reporting Team</div>

                                    <div class="contact-meta">
                                        <strong>Email</strong><br />
                                        grace@example.com
                                    </div>

                                    <div class="contact-meta">
                                        <strong>Phone</strong><br />
                                        0786000000
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </Fragment>
    )
}

export default TicketDetails

