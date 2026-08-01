import React, { Fragment, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { RequestsApi } from "../../helpers/api";

function useQuery() {
  const { search } = useLocation();
  return React.useMemo(() => new URLSearchParams(search), [search]);
}

export default function RequestStatus() {
  const query = useQuery();
  const [requestId, setRequestId] = useState("");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const getStatusBadgeClass = (status) => {
    if (!status) return "status-badge";
    switch (status.toLowerCase()) {
      case "approved":
        return "status-badge status-approved";
      case "rejected":
        return "status-badge status-rejected";
      case "cancelled":
        return "status-badge status-cancelled";
      case "pending":
      default:
        return "status-badge status-pending";
    }
  };

  useEffect(() => {
    const idFromQuery = query.get("id");
    if (idFromQuery) {
      setRequestId(idFromQuery);
      handleTrack(idFromQuery);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  const handleTrack = async (idOverride) => {
    const id = idOverride || requestId;
    if (!id) return;
    setLoading(true);
    setError("");
    setResult(null);
    try {
      const data = await RequestsApi.get(id);
      setResult(data);
    } catch (e) {
      setError(e.response?.data?.error || e.message || "Request not found");
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    handleTrack();
  };

  return (
    <Fragment>
      <h1 className="page-title">Request Tracking</h1>

      <div className="card search-card mb-3">
        <div className="card-header">Search by Request ID</div>
        <div className="card-body">
          <form id="trackingForm" onSubmit={handleSubmit}>
            <div className="row g-3 align-items-end">
              <div className="col-md-8">
                <label className="form-label">Request ID</label>
                <input
                  type="text"
                  className="form-control"
                  placeholder="Enter request ID"
                  required
                  value={requestId}
                  onChange={(e) => setRequestId(e.target.value)}
                />
              </div>
              <div className="col-md-4">
                <button type="submit" className="btn btn-primary w-100" disabled={loading}>
                  <i className="bi bi-search"></i> {loading ? "Tracking..." : "Track Request"}
                </button>
              </div>
            </div>
          </form>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger mt-3" role="alert">
          {error}
        </div>
      )}

      {result ? (
        <div id="requestDetails">
          <div className="request-info">
            <div className="info-item">
              <span className="info-label">Request ID</span>
              <span className="info-value">{result.id}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Request Type</span>
              <span className="info-value">{result.request_type}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Facility Name</span>
              <span className="info-value">{result.facility_data?.name || "—"}</span>
            </div>
            <div className="info-item">
              <span className="info-label">Submitted Date</span>
              <span className="info-value">
                {result.createdAt ? new Date(result.createdAt).toLocaleString() : "—"}
              </span>
            </div>
            <div className="info-item">
              <span className="info-label">Current Status</span>
              <span className="info-value">
                <span className={getStatusBadgeClass(result.current_status)}>
                  {result.current_status || "Unknown"}
                </span>
              </span>
            </div>
            <div className="info-item">
              <span className="info-label">Submitted By</span>
              <span className="info-value">
                {result.initiated_by_name || result.initiated_by_email || "—"}
              </span>
            </div>
          </div>

          <div className="timeline-card">
            <h5 className="mb-4">Request Timeline</h5>

            <div className="timeline">
              <div className="timeline-item">
                <div className="timeline-marker completed">
                  <i className="bi bi-check"></i>
                </div>
                <div className="timeline-content">
                  <div className="timeline-title">Request Submitted</div>
                  <div className="timeline-date">
                    {result.createdAt
                      ? new Date(result.createdAt).toLocaleString()
                      : "Unknown date"}
                  </div>
                  <div className="timeline-description">
                    New facility request has been successfully submitted for review.
                  </div>
                  <div className="timeline-user">
                    <i className="bi bi-person"></i>{" "}
                    Submitted by {result.initiated_by_name || result.initiated_by_email || "—"}
                  </div>
                </div>
              </div>

              {Array.isArray(result.approvals) && result.approvals.length > 0 ? (
                result.approvals.map((approval) => (
                  <div className="timeline-item" key={approval.id || `${approval.stage}-${approval.createdAt}`}>
                    <div className="timeline-marker completed">
                      <i className="bi bi-check"></i>
                    </div>
                    <div className="timeline-content">
                      <div className="timeline-title">
                        {approval.stage} - {approval.action}
                      </div>
                      <div className="timeline-date">
                        {approval.createdAt
                          ? new Date(approval.createdAt).toLocaleString()
                          : "Unknown date"}
                      </div>
                      <div className="timeline-description">
                        {approval.comments || "No additional comments provided."}
                      </div>
                      <div className="timeline-user">
                        <i className="bi bi-person"></i>{" "}
                        {approval.approver_name || approval.approver_email || "Unknown approver"}
                      </div>
                    </div>
                  </div>
                ))
              ) : (
                <div className="timeline-item">
                  <div className="timeline-marker current">
                    <i className="bi bi-arrow-right"></i>
                  </div>
                  <div className="timeline-content">
                    <div className="timeline-title">Awaiting Approvals</div>
                    <div className="timeline-date">In Progress</div>
                    <div className="timeline-description">
                      This request is currently in the queue for review and approval.
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      ) : (
        <div id="noResults" className="no-results">
          <i className="bi bi-search"></i>
          <h5>Search for a Request</h5>
          <p>Enter a request ID above to track its status and timeline</p>
        </div>
      )}
    </Fragment>
  );
}

