import React, { Fragment, useEffect, useMemo, useState } from "react";
import { FacilitiesApi, RequestsApi } from "../../helpers/api";
import './styles.css'

const Dashboard = () => {

  const [summary, setSummary] = useState({
    facilities: 0,
    total: 0,
    approved: 0,
    rejected: 0,
  });
  const [recent, setRecent] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  return (
    <Fragment>
            <div class="page-header">
                <h2>My Facilities</h2>
                <div class="page-header-actions">
                    <button class="btn btn-secondary">
                        <i class="bi bi-download"></i> Export
                    </button>
                    <button class="btn btn-primary">
                        <i class="bi bi-plus-circle"></i> New Request
                    </button>
                </div>
            </div>

            <div class="stats-grid">
                <div class="stat-card">
                    <h4>Total Facilities</h4>
                    <div class="stat-value">8</div>
                </div>
                <div class="stat-card" style={{borderLeftColor: 'var(--success-color)'}}>
                    <h4>Active</h4>
                    <div class="stat-value" style={{color: 'var(--success-color)'}}>6</div>
                </div>
                <div class="stat-card" style={{borderLeftColor: 'var(--warning-color)'}}>
                    <h4>Pending Approval</h4>
                    <div class="stat-value" style={{color: 'var(--warning-color)'}}>1</div>
                </div>
                <div class="stat-card" style={{borderLeftColor: 'var(--danger-color)'}}>
                    <h4>Inactive</h4>
                    <div class="stat-value" style={{color: 'var(--danger-color)'}}>1</div>
                </div>
            </div>

            <div class="filter-section">
                <div class="filter-row">
                    <div class="filter-group">
                        <label>Search Facility</label>
                        <input type="text" placeholder="Enter facility name or MFL ID..."/>
                    </div>
                    <div class="filter-group">
                        <label>Facility Type</label>
                        <select>
                            <option>All Types</option>
                            <option>Hospital</option>
                            <option>Health Center</option>
                            <option>Clinic</option>
                            <option>Dispensary</option>
                        </select>
                    </div>
                    <div class="filter-group">
                        <label>Status</label>
                        <select>
                            <option>All Status</option>
                            <option>Active</option>
                            <option>Pending</option>
                            <option>Inactive</option>
                        </select>
                    </div>
                </div>
            </div>

            <div class="table-section">
                <div class="table-header">
                    <h3><i class="bi bi-hospital"></i> My Registered Facilities (8 records)</h3>
                </div>
                <table>
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>Facility Name</th>
                            <th>Type</th>
                            <th>Level</th>
                            <th>District</th>
                            <th>Status</th>
                            <th>MFL ID</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr>
                            <td>1</td>
                            <td>Kampala Central Health Center</td>
                            <td>Health Center</td>
                            <td>HC III</td>
                            <td>Kampala</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">KMP-HC3-001</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>2</td>
                            <td>Mbarara Regional Referral Hospital</td>
                            <td>Hospital</td>
                            <td>Regional</td>
                            <td>Mbarara</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">MBR-RH-002</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>3</td>
                            <td>Entebbe Medical Clinic</td>
                            <td>Clinic</td>
                            <td>Private</td>
                            <td>Wakiso</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">ENT-CL-003</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>4</td>
                            <td>Jinja Main Hospital</td>
                            <td>Hospital</td>
                            <td>District</td>
                            <td>Jinja</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">JNJ-DH-004</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>5</td>
                            <td>Gulu Community Health Center</td>
                            <td>Health Center</td>
                            <td>HC II</td>
                            <td>Gulu</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">GUL-HC2-005</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>6</td>
                            <td>Masaka District Hospital</td>
                            <td>Hospital</td>
                            <td>District</td>
                            <td>Masaka</td>
                            <td><span class="status-badge status-active">ACTIVE</span></td>
                            <td><span class="mfl-id">MSK-DH-006</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update">
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>7</td>
                            <td>Mbale Regional Clinic</td>
                            <td>Clinic</td>
                            <td>Private</td>
                            <td>Mbale</td>
                            <td><span class="status-badge status-pending">PENDING</span></td>
                            <td><span class="mfl-id">MBL-CL-007</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update" disabled>
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr>
                            <td>8</td>
                            <td>Lira Health Post</td>
                            <td>Health Post</td>
                            <td>HC I</td>
                            <td>Lira</td>
                            <td><span class="status-badge status-inactive">INACTIVE</span></td>
                            <td><span class="mfl-id">LIR-HP-008</span></td>
                            <td>
                                <div class="action-buttons">
                                    <button class="action-btn action-btn-view" title="View Details">
                                        <i class="bi bi-eye"></i> View
                                    </button>
                                    <button class="action-btn action-btn-edit" title="Request Update" disabled>
                                        <i class="bi bi-pencil"></i> Update
                                    </button>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>

                <div class="pagination">
                    <button>Previous</button>
                    <button class="active">1</button>
                    <button>2</button>
                    <button>3</button>
                    <button>Next</button>
                </div>
            </div>
    </Fragment>
  );
}

export default Dashboard;

