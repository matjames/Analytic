import React, { useState, useEffect, Fragment } from 'react'
import API from "../../helpers/api";
import TrendChart from './TrendChart';
import IssueDistribution from './IssueDistribution';

const Dashboard = () => {

    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState({});
    const [total, setTotal] = useState(0);
    const [issues, setIssues] = useState([]);

    const token = localStorage.getItem("token");
    const user = JSON.parse(localStorage.getItem("user") || "null")

    const loadTickets = async () => {
        setLoading(true);
        try {

            if (!token) {
                throw new Error("Authorization token not found");
            }

            let res;
            if (user?.role === 'admin') {
                res = await API.get(`/t/tickets/count`, {
                    headers: {
                        Authorization: `Bearer ${token}`
                    },
                });
            } else {
                res = await API.get(`/t/agents/count`, {
                    headers: {
                        Authorization: `Bearer ${token}`
                    },
                });
            }

            setCount(res?.data.statusCounts || {});
            setTotal(res?.data.results || 0);
            setIssues(res?.data.tickets || []);
            setLoading(false);
        } catch (error) {
            console.log("error", error);
            setLoading(false);
        }
    };

    useEffect(() => {
        loadTickets();
    }, []);

    const stats = [
        { label: 'Work Items', value: total || 0, icon: 'bx bx-list-ul', tone: 'widget-two-success' },
        { label: 'Open', value: count?.open || 0, icon: 'bx bx-copy-alt', tone: 'widget-two-primary' },
        { label: 'In Progress', value: count?.inprogress || 0, icon: 'bx bx-archive-in', tone: 'widget-two-warning' },
        { label: 'Resolved', value: count?.closed || 0, icon: 'bx bx-check-circle', tone: 'widget-two-info' },
        { label: 'Overdue', value: count?.overdue || 0, icon: 'bx bx-time-five', tone: 'widget-three' },
    ];

    return (
        <Fragment>
            <div className="row">
                <div className="col-12">
                    <div className="mb-4 d-sm-flex align-items-center justify-content-between">
                        <div>
                            <h4 className="mb-1 font-size-20 fw-bold">Operations Command Center</h4>
                            <p className="text-muted mb-0">Monitor active work items, response progress, and service bottlenecks in one view.</p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row g-3 mb-4">
                {stats.map((item, index) => (
                    <div className="col-xl-2 col-md-4 col-sm-6" key={index}>
                        <div className={`card mini-stats-wid ${item.tone} h-100 border-0`}>
                            <div className="card-body">
                                <div className="d-flex align-items-center justify-content-between">
                                    <div>
                                        <p className="text-muted fw-medium mb-1">{item.label}</p>
                                        <h4 className="mb-0">{item.value}</h4>
                                    </div>
                                    <div className="avatar-sm rounded-circle bg-primary mini-stat-icon">
                                        <span className="avatar-title rounded-circle bg-primary">
                                            <i className={`${item.icon} font-size-20`}></i>
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                ))}
            </div>

            <div className="row mb-4">
                <div className="col-12">
                    <div className="card border-0 shadow-sm">
                        <div className="card-body">
                            <div className="d-flex flex-wrap justify-content-between align-items-start gap-3">
                                <div>
                                    <h5 className="mb-1">Deployment readiness</h5>
                                    <p className="text-muted mb-0">Core operations modules are wired, organized, and ready for staging review.</p>
                                </div>
                                <div className="d-flex flex-wrap gap-2">
                                    <span className="badge bg-success">Intake live</span>
                                    <span className="badge bg-success">Queue live</span>
                                    <span className="badge bg-success">Knowledge base live</span>
                                    <span className="badge bg-success">Admin workspace live</span>
                                </div>
                            </div>
                            <div className="row mt-3 g-3">
                                <div className="col-md-3 col-sm-6">
                                    <div className="p-3 rounded-3 bg-light">
                                        <div className="fw-semibold">Work intake</div>
                                        <div className="small text-muted">Capture and route new work items</div>
                                    </div>
                                </div>
                                <div className="col-md-3 col-sm-6">
                                    <div className="p-3 rounded-3 bg-light">
                                        <div className="fw-semibold">Operations queue</div>
                                        <div className="small text-muted">Manage open, in-progress, and overdue items</div>
                                    </div>
                                </div>
                                <div className="col-md-3 col-sm-6">
                                    <div className="p-3 rounded-3 bg-light">
                                        <div className="fw-semibold">Knowledge enablement</div>
                                        <div className="small text-muted">Playbooks, articles, and training</div>
                                    </div>
                                </div>
                                <div className="col-md-3 col-sm-6">
                                    <div className="p-3 rounded-3 bg-light">
                                        <div className="fw-semibold">Governance</div>
                                        <div className="small text-muted">Agents, routing, and oversight</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row g-3">
                <div className="col-lg-4">
                    <div className="card border-0 shadow-sm h-100">
                        <div className="card-body">
                            <h5 className="card-title mb-3">By Program Level</h5>
                            <IssueDistribution data={issues} indicator='facilityLevel' />
                        </div>
                    </div>
                </div>
                <div className="col-lg-4">
                    <div className="card border-0 shadow-sm h-100">
                        <div className="card-body">
                            <h5 className="card-title mb-3">By Status</h5>
                            <IssueDistribution data={issues} indicator="status" />
                        </div>
                    </div>
                </div>
                <div className="col-lg-4">
                    <div className="card border-0 shadow-sm h-100">
                        <div className="card-body">
                            <h5 className="card-title mb-3">By Work Item Category</h5>
                            <IssueDistribution data={issues} indicator="issueCategory" />
                        </div>
                    </div>
                </div>
            </div>
            <div className="row mt-3">
                <div className="col-lg-12">
                    <div className="card border-0 shadow-sm">
                        <div className="card-body">
                            <h5 className="card-title mb-3">Resolution Trend</h5>
                            <TrendChart data={issues} />
                        </div>
                    </div>
                </div>
            </div>
        </Fragment>
    )
}

export default Dashboard