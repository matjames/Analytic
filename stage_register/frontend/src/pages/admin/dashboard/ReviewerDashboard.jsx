import React, { Fragment, useEffect, useState } from "react";
import { useHistory } from "react-router-dom";
import { RequestsApi } from "../../../helpers/api";

export default function ReviewerDashboard() {
  const user = JSON.parse(localStorage.getItem("user") || "null");
  const history = useHistory();
  const [summary, setSummary] = useState({
    pending: 0,
    approved: 0,
    rejected: 0,
    total: 0,
    mine: 0,
  });
  const [recent, setRecent] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [districtName, setDistrictName] = useState("");

  function mapRoleToStage(role) {
    const map = {
      district_approver: "district_approver",
      district: "district_approver",
      moh_clinical: "moh_clinical",
      moh_publisher: "moh_publisher",
    };
    return map[role] || "";
  }

  const getStatusBadgeClass = (status) => {
    const badges = {
      pending: "pending",
      approved: "approved",
      rejected: "rejected",
      cancelled: "cancelled",
    };
    return badges[status] || "";
  };

  const getTypeLabel = (type) => {
    const labels = {
      new_addition: "New_addition",
      update: "Update",
      deactivation: "Deactivation",
    };
    return labels[type] || type || "—";
  };

  const getStageLabel = (stage) => {
    const labels = {
      district_approver: "District Approver",
      moh_clinical: "National Operations Review",
      moh_publisher: "Registry Publishing",
      completed: "Completed",
    };
    return labels[stage] || stage || "—";
  };

  useEffect(() => {
    const load = async () => {
      const token = localStorage.getItem("token");
      if (!token) return;

      setLoading(true);
      setError("");
      try {
        // Load stats, requests, and district info in parallel
        const [statsRes, requestsRes, districtInfoRes] = await Promise.all([
          RequestsApi.getStats().catch(() => null),
          RequestsApi.list({ page: 1, pageSize: 20 }),
          RequestsApi.getDistrictInfo().catch(() => null),
        ]);

        // Set district name if available
        if (districtInfoRes?.district?.name) {
          setDistrictName(districtInfoRes.district.name);
        }

        const rows = requestsRes.rows || [];
        const currentStage = mapRoleToStage(user?.role);

        // Use stats API if available, otherwise calculate from requests
        let pending = 0;
        let approved = 0;
        let rejected = 0;
        let total = 0;
        let mine = 0;

        if (statsRes) {
          pending = statsRes.pending || 0;
          approved = statsRes.approved || 0;
          rejected = statsRes.rejected || 0;
          total = statsRes.total || 0;
        } else {
          pending = rows.filter((r) => r.current_status === "pending").length;
          approved = rows.filter((r) => r.current_status === "approved").length;
          rejected = rows.filter((r) => r.current_status === "rejected").length;
          total = requestsRes.total || rows.length;
        }

        // Calculate "My Queue" - requests pending at reviewer's stage
        mine = rows.filter(
          (r) => r.current_status === "pending" && r.current_stage === currentStage
        ).length;

        setSummary({
          pending,
          approved,
          rejected,
          total,
          mine,
        });
        setRecent(rows);
      } catch (e) {
        if (e?.response?.status === 401) return;
        setError(e?.response?.data?.error || e.message || "Failed to load dashboard");
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [user?.role]);

  return (
    <Fragment>
      <div class="page-header">
        <h2>Reviewer Dashboard{districtName && ` - ${districtName}`}</h2>
      </div>

      {error && <div class="alert alert-danger">{error}</div>}

      <div class="quick-stats">
        <div class="stat-card pending">
          <div class="stat-label">Pending</div>
          <div class="stat-value">{loading ? "—" : summary.pending}</div>
        </div>
        <div class="stat-card approved">
          <div class="stat-label">Approved</div>
          <div class="stat-value">{loading ? "—" : summary.approved}</div>
        </div>
        <div class="stat-card rejected">
          <div class="stat-label">Rejected</div>
          <div class="stat-value">{loading ? "—" : summary.rejected}</div>
        </div>
        <div class="stat-card queue">
          <div class="stat-label">My Queue</div>
          <div class="stat-value">{loading ? "—" : summary.mine}</div>
        </div>
      </div>

      <div class="table-section">
        <h3 class="section-title">Recent Requests</h3>
        <div class="table-responsive">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Field Station</th>
                <th>Type</th>
                <th>Status</th>
                <th>Stage</th>
                <th>Submitted</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan="7" class="text-center">Loading requests...</td>
                </tr>
              ) : recent.length === 0 ? (
                <tr>
                  <td colSpan="7" class="text-center">No requests found.</td>
                </tr>
              ) : (
                recent.map((req) => {
                  const facilityData = req.facility_data || {};
                  const statusClass = getStatusBadgeClass(req.current_status);
                  return (
                    <tr key={req.id}>
                      <td>{req.id}</td>
                      <td>{facilityData.name || "—"}</td>
                      <td>{getTypeLabel(req.request_type)}</td>
                      <td>
                        <span class={`status-badge ${statusClass}`}>
                          {req.current_status ? req.current_status.charAt(0).toUpperCase() + req.current_status.slice(1) : "—"}
                        </span>
                      </td>
                      <td><span class="stage-badge">{getStageLabel(req.current_stage)}</span></td>
                      <td>
                        {req.createdAt
                          ? new Date(req.createdAt).toLocaleString("en-US", {
                            year: "numeric",
                            month: "2-digit",
                            day: "2-digit",
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })
                          : "—"}
                      </td>
                      <td>
                        <div class="action-btns">
                          <button 
                            class="btn btn-view"
                            onClick={() => history.push(`/requests/${req.id}`)}
                          >
                            <i class="bi bi-eye"></i> View
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

      </div>

    </Fragment>
  );
}
