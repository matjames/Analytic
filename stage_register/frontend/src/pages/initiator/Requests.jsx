import React, { Fragment, useEffect, useMemo, useState } from "react";
import { useHistory } from "react-router-dom";
import { RequestsApi } from "../../helpers/api";

export default function Requests() {
  const history = useHistory();
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [filterStatus, setFilterStatus] = useState("");
  const [filterType, setFilterType] = useState("");
  const [stats, setStats] = useState({
    total: 0,
    pending: 0,
    approved: 0,
    rejected: 0,
  });

  useEffect(() => {
    loadRequests();
    loadStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, filterStatus, filterType]);

  async function loadRequests() {
    const token = localStorage.getItem("token");
    if (!token) return;

    setLoading(true);
    setError("");
    try {
      const params = { page, pageSize };
      if (filterStatus) params.status = filterStatus;
      if (filterType) params.request_type = filterType;
      const data = await RequestsApi.list(params);
      setRequests(data.rows || []);
      setTotal(data.total || 0);
    } catch (e) {
      if (e?.response?.status === 401) return;
      setError(e?.response?.data?.error || e.message || "Failed to load requests");
    } finally {
      setLoading(false);
    }
  }

  async function loadStats() {
    const token = localStorage.getItem("token");
    if (!token) return;

    try {
      const data = await RequestsApi.getStats();
      setStats({
        total: data.total || 0,
        pending: data.pending || 0,
        approved: data.approved || 0,
        rejected: data.rejected || 0,
      });
    } catch (e) {
      // Stats failure shouldn't block the page
      // Optionally log or ignore
    }
  }

  const totalPages = useMemo(() => Math.max(1, Math.ceil(total / pageSize)), [total, pageSize]);

  const getStatusBadgeClass = (status) => {
    const badges = {
      pending: "badge-pending",
      approved: "badge-approved",
      rejected: "badge-rejected",
      cancelled: "badge-cancelled",
    };
    return `badge ${badges[status] || "badge-secondary"}`;
  };

  const getTypeLabel = (type) => {
    const labels = {
      new_addition: "New Addition",
      update: "Update",
      deactivation: "Deactivation",
    };
    return labels[type] || type || "—";
  };

  const handleView = (id) => {
    history.push(`/initiator/requests/status?id=${id}`);
  };

  const handleStatusChange = (e) => {
    setPage(1);
    setFilterStatus(e.target.value);
  };

  const handleTypeChange = (e) => {
    setPage(1);
    setFilterType(e.target.value);
  };

  const handlePreviousPage = () => {
    setPage((prev) => Math.max(1, prev - 1));
  };

  const handleNextPage = () => {
    setPage((prev) => Math.min(totalPages, prev + 1));
  };

  const handlePageClick = (pageNum) => {
    setPage(pageNum);
  };

  const fromItem = (page - 1) * pageSize + 1;
  const toItem = Math.min(page * pageSize, total);

  return (
    <Fragment>
      <div className="page-header">
        <div>
          <h2 className="page-title">Facility Requests</h2>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card primary">
          <div className="stat-label">Total Requests</div>
          <h3 className="stat-value">{stats.total}</h3>
        </div>
        <div className="stat-card warning">
          <div className="stat-label">Pending</div>
          <h3 className="stat-value">{stats.pending}</h3>
        </div>
        <div className="stat-card success">
          <div className="stat-label">Approved</div>
          <h3 className="stat-value">{stats.approved}</h3>
        </div>
        <div className="stat-card danger">
          <div className="stat-label">Rejected</div>
          <h3 className="stat-value">{stats.rejected}</h3>
        </div>
      </div>

      <div className="filter-section">
        <div className="filter-row">
          <div>
            <label className="form-label">Request Type</label>
            <select className="form-select" value={filterType} onChange={handleTypeChange}>
              <option value="">All Types</option>
              <option value="new_addition">New Addition</option>
              <option value="update">Update</option>
              <option value="deactivation">Deactivation</option>
            </select>
          </div>
          <div>
            <label className="form-label">Status</label>
            <select className="form-select" value={filterStatus} onChange={handleStatusChange}>
              <option value="">All Status</option>
              <option value="pending">Pending</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </div>
        </div>
      </div>

      <div className="table-section">
        <div className="table-header">
          <h3 className="table-title">Recent Requests</h3>
        </div>
        {error && <div className="alert alert-danger small mb-2">{error}</div>}
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>Facility Name</th>
              <th>Type</th>
              <th>Submitted By</th>
              <th>Date</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan="7" className="text-center">
                  Loading requests...
                </td>
              </tr>
            ) : requests.length === 0 ? (
              <tr>
                <td colSpan="7" className="text-center">
                  No requests found.
                </td>
              </tr>
            ) : (
              requests.map((req) => {
                const facilityData = req.facility_data || {};
                return (
                  <tr key={req.id}>
                    <td>{req.id}</td>
                    <td>{facilityData.name || "—"}</td>
                    <td>{getTypeLabel(req.request_type)}</td>
                    <td>{req.initiated_by_name || req.initiated_by_email || "—"}</td>
                    <td>
                      {req.createdAt ? new Date(req.createdAt).toLocaleDateString() : "—"}
                    </td>
                    <td>
                      <span className={getStatusBadgeClass(req.current_status)}>
                        {req.current_status || "—"}
                      </span>
                    </td>
                    <td>
                      <button
                        className="action-btn btn-sm-primary"
                        type="button"
                        onClick={() => handleView(req.id)}
                      >
                        View
                      </button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
        <div className="pagination-wrapper">
          <div className="pagination-info">
            {total > 0
              ? `Showing ${fromItem} to ${toItem} of ${total} requests`
              : "No requests to display"}
          </div>
          <nav>
            <ul className="pagination">
              <li className={`page-item ${page === 1 ? "disabled" : ""}`}>
                <button className="page-link" type="button" onClick={handlePreviousPage}>
                  Previous
                </button>
              </li>
              {Array.from({ length: totalPages }).map((_, idx) => {
                const pageNum = idx + 1;
                return (
                  <li
                    key={pageNum}
                    className={`page-item ${pageNum === page ? "active" : ""}`}
                  >
                    <button
                      className="page-link"
                      type="button"
                      onClick={() => handlePageClick(pageNum)}
                    >
                      {pageNum}
                    </button>
                  </li>
                );
              })}
              <li className={`page-item ${page === totalPages ? "disabled" : ""}`}>
                <button className="page-link" type="button" onClick={handleNextPage}>
                  Next
                </button>
              </li>
            </ul>
          </nav>
        </div>
      </div>
    </Fragment>
  );
}

