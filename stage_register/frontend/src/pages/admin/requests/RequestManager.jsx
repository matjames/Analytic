import React, { Fragment, useEffect, useState } from "react";
import { RequestsApi, FacilityLevelsApi, OwnershipTypesApi, AuthorityTypesApi } from "../../../helpers/api";
import { AVAILABLE_SERVICES } from "../../../helpers/services";

import { useHistory, useLocation } from "react-router-dom";

export default function RequestManager() {
  const user = JSON.parse(localStorage.getItem("user") || "null");
  const history = useHistory();
  const location = useLocation();
  const [requests, setRequests] = useState([]);
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [requestType, setRequestType] = useState("new_addition"); // new_addition, update, deactivation
  const [selectedFacility, setSelectedFacility] = useState(null);
  const [facilitySearchQuery, setFacilitySearchQuery] = useState("");
  const [availableFacilities, setAvailableFacilities] = useState([]);
  const [loadingFacilities, setLoadingFacilities] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [total, setTotal] = useState(0);
  const [filterStatus, setFilterStatus] = useState("");
  const [filterType, setFilterType] = useState("");

  const handleStatusChange = (e) => {
    setFilterStatus(e.target.value);
    setPage(1);
  };

  const handleTypeChange = (e) => {
    setFilterType(e.target.value);
    setPage(1);
  };

  const handleClearFilters = () => {
    setFilterStatus("");
    setFilterType("");
    setPage(1);
  };

  // Read status from URL query parameters
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const statusParam = params.get("status");
    if (statusParam) {
      setFilterStatus(statusParam);
    } else {
      setFilterStatus("");
    }
  }, [location.search]);

  const [form, setForm] = useState({
    facility_name: "",
    subcounty_id: "",
    level: "",
    ownership: "",
    authority: "",
    licensed: false,
    address: "",
    contact_personemail: "",
    contact_personmobile: "",
    contact_personname: "",
    contact_persontitle: "",
    longitude: "",
    latitude: "",
    opening_date: "",
    bed_capacity: "",
    services: [],
  });
  const [district, setDistrict] = useState(null);
  const [subcounties, setSubcounties] = useState([]);
  const [documents, setDocuments] = useState([]);

  useEffect(() => {
    // Only load data if user is authenticated
    const token = localStorage.getItem("token");
    const currentUser = JSON.parse(localStorage.getItem("user") || "null");
    if (!token || !currentUser) return;

    loadRequests();
    loadLookups(); // Load lookups for all users to display names
    if (currentUser?.role === "public" || currentUser?.role === "district_initiator") {
      loadDistrictInfo();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, filterStatus, filterType]);

  async function loadDistrictInfo() {
    try {
      const info = await RequestsApi.getDistrictInfo();
      setDistrict(info.district);
      setSubcounties(info.subcounties || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load district information");
    }
  }

  async function loadFacilities(searchQuery = "") {
    if (requestType === "new_addition") return;
    setLoadingFacilities(true);
    try {
      const facilities = await RequestsApi.getFacilities(searchQuery);
      setAvailableFacilities(facilities || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load facilities");
    } finally {
      setLoadingFacilities(false);
    }
  }

  useEffect(() => {
    if (requestType === "update" || requestType === "deactivation") {
      loadFacilities(facilitySearchQuery);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestType, facilitySearchQuery]);

  async function loadLookups() {
    try {
      const [levels, ownerships, authorities] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
      ]);
      setFacilityLevels(levels);
      setOwnershipTypes(ownerships);
      setAuthorityTypes(authorities);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load lookup data");
    }
  }

  async function loadRequests() {
    // Check if user is still authenticated before making API call
    const token = localStorage.getItem("token");
    if (!token) return;

    setLoading(true);
    setError("");
    try {
      const params = {
        page,
        pageSize,
      };
      if (filterStatus) params.status = filterStatus;
      if (filterType) params.request_type = filterType;

      const data = await RequestsApi.list(params);
      setRequests(data.rows || []);
      setTotal(data.total || 0);
    } catch (e) {
      // Don't set error if user is logged out (401)
      if (e?.response?.status === 401) return;
      setError(e?.response?.data?.error || e.message || "Failed to load requests");
    } finally {
      setLoading(false);
    }
  }

  function handleDocumentChange(e) {
    const files = Array.from(e.target.files || []);
    setDocuments([...documents, ...files]);
  }

  function removeDocument(index) {
    setDocuments(documents.filter((_, i) => i !== index));
  }

  async function handleCreateRequest() {
    setSaving(true);
    setError("");
    try {
      // Validate based on request type
      if (requestType === "new_addition") {
        if (!form.facility_name.trim()) {
          setError("Facility name is required");
          setSaving(false);
          return;
        }
        if (!form.subcounty_id) {
          setError("Subcounty is required");
          setSaving(false);
          return;
        }
      } else if (requestType === "update" || requestType === "deactivation") {
        if (!selectedFacility) {
          setError("Please select a facility");
          setSaving(false);
          return;
        }
        if (!documents || documents.length === 0) {
          setError("Supporting documents are required for update and deactivation requests");
          setSaving(false);
          return;
        }
      }

      // Prepare facility_data based on request type
      let facilityData = null;
      if (requestType === "new_addition" || requestType === "update") {
        facilityData = {
          name: form.facility_name || null,
          short_name: form.facility_name || null,
          admin_unit_id: form.subcounty_id ? Number(form.subcounty_id) : null,
          level: form.level || null,
          ownership: form.ownership || null,
          authority: form.authority || null,
          licensed: form.licensed || false,
          address: form.address || null,
          contact_personemail: form.contact_personemail || null,
          contact_personmobile: form.contact_personmobile || null,
          contact_personname: form.contact_personname || null,
          contact_persontitle: form.contact_persontitle || null,
          longitude: form.longitude ? parseFloat(form.longitude) : null,
          latitude: form.latitude ? parseFloat(form.latitude) : null,
          opening_date: form.opening_date || null,
          bed_capacity: form.bed_capacity ? parseInt(form.bed_capacity) : null,
          services: form.services || [],
        };
      } else if (requestType === "deactivation") {
        // For deactivation, store the reason in facility_data
        facilityData = {
          deactivation_reason: form.address || null, // Reusing address field for reason
        };
      }

      await RequestsApi.create(
        requestType,
        facilityData,
        documents,
        selectedFacility ? selectedFacility.id : null
      );

      // Reset form
      setForm({
        facility_name: "",
        subcounty_id: "",
        level: "",
        ownership: "",
        authority: "",
        licensed: false,
        address: "",
        contact_personemail: "",
        contact_personmobile: "",
        contact_personname: "",
        contact_persontitle: "",
        longitude: "",
        latitude: "",
        opening_date: "",
        bed_capacity: "",
        services: [],
      });
      setDocuments([]);
      setShowCreateModal(false);
      await loadRequests();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to create request");
    } finally {
      setSaving(false);
    }
  }

  function handleViewDetails(request) {
    history.push(`/requests/${request.id}`);
  }

  function getStatusBadge(status) {
    const badges = {
      pending: "warning",
      approved: "success",
      rejected: "danger",
      cancelled: "secondary",
    };
    return badges[status] || "secondary";
  }

  function getStageLabel(stage) {
    const labels = {
      district_approver: "District Approver",
      moh_clinical: "National Operations Review",
      moh_publisher: "Registry Publishing",
      completed: "Completed",
    };
    return labels[stage] || stage;
  }

  function getRequestWorkflowStatus(request) {
    if (request.current_status === "approved") {
      return { text: "Fully Approved", badge: "success" };
    }
    if (request.current_status === "rejected") {
      return { text: "Rejected", badge: "danger" };
    }
    const stageLabel = getStageLabel(request.current_stage) || "Pending";
    return { text: stageLabel, badge: "warning" };
  }

  function getTypeLabel(type) {
    const labels = {
      new_addition: "New Addition",
      update: "Update",
      deactivation: "Deactivation",
    };
    return labels[type] || type;
  }

  // Helper functions to get names from IDs
  const getOwnershipName = (mflUid) => {
    if (!mflUid) return "—";
    const ownership = ownershipTypes.find((o) => o.mfl_uid === mflUid || o.id === mflUid);
    return ownership?.name || mflUid;
  };

  const getAuthorityName = (mflUid) => {
    if (!mflUid) return "—";
    const authority = authorityTypes.find((a) => a.mfl_uid === mflUid || a.id === mflUid);
    return authority?.name || mflUid;
  };

  const getLevelName = (mflUid) => {
    if (!mflUid) return "—";
    const level = facilityLevels.find((l) => l.mfl_uid === mflUid || l.id === mflUid);
    return level?.name || mflUid;
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <Fragment>
      <div class="page-header">
        <div class="page-title">
          <h2>Field Station Requests</h2>
          <div class="page-subtitle">Review and approve field station registry requests</div>
        </div>
      </div>

      {error && <div class="alert alert-danger">{error}</div>}

      <div class="filters-section">
        <div class="filter-group">
          <label>Status:</label>
          <select id="filter-status" value={filterStatus} onChange={handleStatusChange}>
            <option value="">All Statuses</option>
            <option value="pending">Pending</option>
            <option value="approved">Approved</option>
            <option value="rejected">Rejected</option>
          </select>
        </div>
        <div class="filter-group">
          <label>Type:</label>
          <select id="filter-type" value={filterType} onChange={handleTypeChange}>
            <option value="">All Types</option>
            <option value="new_addition">New Addition</option>
            <option value="update">Update</option>
            <option value="deactivation">Deactivation</option>
          </select>
        </div>
        <button class="btn-clear" onClick={handleClearFilters}>
          <i class="bi bi-x"></i> Clear Filters
        </button>
      </div>

      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Type</th>
              <th>Field Station</th>
              <th>Status</th>
              <th>Stage</th>
              <th>Ownership</th>
              <th>Initiated By</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan="9" class="text-center">Loading requests...</td>
              </tr>
            ) : requests.length === 0 ? (
              <tr>
                <td colSpan="9" class="text-center">No requests found.</td>
              </tr>
            ) : (
              requests.map((req) => {
                const facilityData = req.facility_data || {};
                // Handle null, undefined, or empty string status
                const status = (req.current_status && req.current_status.trim()) || "pending";
                const statusBadge = getStatusBadge(status);
                const workflowStatus = getRequestWorkflowStatus(req);
                return (
                  <tr key={req.id}>
                    <td>#{req.id}</td>
                    <td>{getTypeLabel(req.request_type)}</td>
                    <td>{facilityData.name || "—"}</td>
                    <td>
                      <span class={`badge badge-${statusBadge}`}>
                        {status.charAt(0).toUpperCase() + status.slice(1)}
                      </span>
                    </td>
                    <td>
                      <span class={`badge ${workflowStatus.badge === "warning" ? "badge-in-review" : `badge-${workflowStatus.badge}`}`}>
                        {workflowStatus.text}
                      </span>
                    </td>
                    <td>{getOwnershipName(facilityData.ownership)}</td>
                    <td>{req.initiated_by_name || req.initiated_by_email || "—"}</td>
                    <td>
                      {req.createdAt
                        ? new Date(req.createdAt).toLocaleString("en-US", {
                            year: "numeric",
                            month: "2-digit",
                            day: "2-digit",
                            hour: "2-digit",
                            minute: "2-digit",
                          })
                        : "—"}
                    </td>
                    <td>
                      <button class="btn-action" onClick={() => handleViewDetails(req)}>
                        View
                      </button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        {totalPages > 1 && (
          <div class="pagination-wrapper" style={{ marginTop: "1rem", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div class="pagination-info">
              Showing {((page - 1) * pageSize) + 1} to {Math.min(page * pageSize, total)} of {total} requests
            </div>
            <nav>
              <ul class="pagination" style={{ margin: 0, display: "flex", gap: "0.5rem" }}>
                <li class={`page-item ${page === 1 ? "disabled" : ""}`}>
                  <button
                    class="page-link"
                    type="button"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                  >
                    Previous
                  </button>
                </li>
                {Array.from({ length: Math.min(totalPages, 10) }).map((_, idx) => {
                  const pageNum = idx + 1;
                  return (
                    <li key={pageNum} class={`page-item ${pageNum === page ? "active" : ""}`}>
                      <button class="page-link" type="button" onClick={() => setPage(pageNum)}>
                        {pageNum}
                      </button>
                    </li>
                  );
                })}
                <li class={`page-item ${page === totalPages ? "disabled" : ""}`}>
                  <button
                    class="page-link"
                    type="button"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                  >
                    Next
                  </button>
                </li>
              </ul>
            </nav>
          </div>
        )}
      </div>

    </Fragment>
  );
}
