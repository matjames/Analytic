import React, { useEffect, useMemo, useState, Fragment } from "react";
import { useHistory, useParams } from "react-router-dom";
import { RequestsApi, FacilityLevelsApi, OwnershipTypesApi, AuthorityTypesApi, FacilitiesApi, UnitsApi } from "../../../helpers/api";

export default function RequestDetailPage() {
  const { id } = useParams();
  const history = useHistory();
  const user = JSON.parse(localStorage.getItem("user") || "null");

  const [request, setRequest] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [approvalComments, setApprovalComments] = useState("");
  const [rejectionReason, setRejectionReason] = useState("");
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [adminUnits, setAdminUnits] = useState([]);
  const [currentFacility, setCurrentFacility] = useState(null);
  const [facilityLoading, setFacilityLoading] = useState(false);

  useEffect(() => {
    loadRequest();
    loadLookups();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  async function loadLookups() {
    try {
      const [levels, ownerships, authorities, units] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
        UnitsApi.list(),
      ]);
      setFacilityLevels(levels || []);
      setOwnershipTypes(ownerships || []);
      setAuthorityTypes(authorities || []);
      setAdminUnits(units || []);
    } catch (e) {
      console.error("Failed to load lookup data:", e);
    }
  }

  async function loadRequest() {
    setLoading(true);
    setError("");
    try {
      const data = await RequestsApi.get(id);
      setRequest(data);
      // For update requests, also load the current facility details using mfl_uid
      if (data?.request_type === "update" && data.facility_mfl_uid) {
        loadCurrentFacility(data.facility_mfl_uid);
      } else {
        setCurrentFacility(null);
      }
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load request");
    } finally {
      setLoading(false);
    }
  }

  async function loadCurrentFacility(mflUid) {
    setFacilityLoading(true);
    try {
      // Uses /facilities/:id endpoint where :id is the mfl_uid
      // Backend queries mfl_details view using WHERE mfl_uid = $1
      const data = await FacilitiesApi.get(mflUid);
      setCurrentFacility(data);
    } catch (e) {
      console.error("Failed to load current facility details:", e);
      setError("Failed to load current facility details. Please refresh the page.");
    } finally {
      setFacilityLoading(false);
    }
  }

  const getStatusBadge = (status) => {
    const badges = {
      pending: "warning",
      approved: "success",
      rejected: "danger",
      cancelled: "secondary",
    };
    return badges[status] || "secondary";
  };

  const getStageLabel = (stage) => {
    const labels = {
      district_approver: "District Approver",
      moh_clinical: "National Operations Review",
      moh_publisher: "Registry Publishing",
      completed: "Completed",
    };
    return labels[stage] || stage;
  };

  const getWorkflowProgress = (req) => {
    const stages = [
      { key: "district_approver", label: "District Approver", order: 1 },
      { key: "moh_clinical", label: "National Operations Review", order: 2 },
      { key: "moh_publisher", label: "Registry Publishing", order: 3 },
    ];

    const approvals = req.approvals || [];
    const approvedStages = new Set(approvals.filter((a) => a.action === "approved").map((a) => a.stage));
    const rejectedStages = new Set(approvals.filter((a) => a.action === "rejected").map((a) => a.stage));

    return stages.map((stage) => {
      const isApproved = approvedStages.has(stage.key);
      const isRejected = rejectedStages.has(stage.key);
      const isCurrent = req.current_stage === stage.key && req.current_status === "pending";
      const isCompleted = req.current_status === "approved" && req.current_stage === "completed";

      return {
        ...stage,
        status: isRejected ? "rejected" : isApproved ? "approved" : isCurrent ? "current" : isCompleted ? "completed" : "pending",
        isApproved,
        isRejected,
        isCurrent,
        isCompleted,
      };
    });
  };

  const canApprove = (req) => {
    if (!req || req.current_status !== "pending") return false;
    const roleStageMap = {
      district_approver: "district_approver",
      district: "district_approver",
      moh_clinical: "moh_clinical",
      moh_publisher: "moh_publisher",
    };
    return roleStageMap[user?.role] === req.current_stage;
  };

  async function handleApprove() {
    if (!request) return;
    setSaving(true);
    setError("");
    try {
      await RequestsApi.approve(request.id, { comments: approvalComments });
      setApprovalComments("");
      await loadRequest();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to approve request");
    } finally {
      setSaving(false);
    }
  }

  async function handleReject() {
    if (!request || !rejectionReason.trim()) {
      setError("Rejection reason is required");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await RequestsApi.reject(request.id, {
        rejection_reason: rejectionReason,
        comments: approvalComments,
      });
      setRejectionReason("");
      setApprovalComments("");
      await loadRequest();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to reject request");
    } finally {
      setSaving(false);
    }
  }

  const workflowProgress = useMemo(() => (request ? getWorkflowProgress(request) : []), [request]);

  // Memoize facilityData to avoid dependency issues
  const facilityData = useMemo(() => (request?.facility_data || {}), [request]);

  // Build list of changed fields for update requests (must be before early returns)
  const getChangedFields = useMemo(() => {
    if (!request || request.request_type !== "update" || !facilityData) {
      return [];
    }
    
    // If current facility is not loaded yet, show all requested fields
    if (!currentFacility) {
      const fields = [
        { key: "name", label: "Field Station Name" },
        { key: "level", label: "Level" },
        { key: "ownership", label: "Ownership" },
        { key: "authority", label: "Authority" },
        { key: "status", label: "Status" },
        { key: "licensed", label: "Operationally Verified" },
        { key: "address", label: "Address" },
        { key: "contact_personname", label: "Contact Person" },
        { key: "contact_personemail", label: "Contact Email" },
        { key: "contact_personmobile", label: "Contact Mobile" },
        { key: "latitude", label: "Latitude" },
        { key: "longitude", label: "Longitude" },
        { key: "opening_date", label: "Opening Date" },
        { key: "bed_capacity", label: "Agent Capacity" },
        { key: "region_uid", label: "Region" },
        { key: "district_uid", label: "District" },
        { key: "admin_unit_id", label: "Subcounty / Admin Unit" },
        { key: "services", label: "Services" },
      ];
      
      // Return fields that have non-empty values in facilityData
      return fields.filter(field => {
        if (field.key === "subcounty_id") {
          const hasAdminUnitId = facilityData["admin_unit_id"] !== null && 
                                 facilityData["admin_unit_id"] !== undefined && 
                                 facilityData["admin_unit_id"] !== "";
          if (hasAdminUnitId) return false;
        }
        const newValue = facilityData[field.key];
        return newValue !== null && newValue !== undefined && newValue !== "";
      });
    }

    // Helper to check if a value has actually changed (inline version)
    const hasValueChanged = (newValue, currentValue) => {
      // If new value is null, undefined, or empty string, it hasn't changed
      if (newValue === null || newValue === undefined || newValue === "") {
        return false;
      }
      // For arrays, check if they're different
      if (Array.isArray(newValue) && Array.isArray(currentValue)) {
        return JSON.stringify([...newValue].sort()) !== JSON.stringify([...currentValue].sort());
      }
      // For boolean values, always show if present (since false is a valid change)
      if (typeof newValue === "boolean") {
        return newValue !== currentValue;
      }
      // For other values, compare directly
      return String(newValue) !== String(currentValue || "");
    };

    // Helper to get current value for a field (inline version)
    const getCurrentValue = (fieldName) => {
      const current = currentFacility;
      if (!current) return null;

      switch (fieldName) {
        case "name":
          return current.name;
        case "level":
          // Current facility has level as string name from mfl_details
          // But we need to compare with mfl_uid from request, so get the mfl_uid
          const currentLevelName = current.level?.name || current.level_name || current.facility_level?.name || (typeof current.level === 'string' ? current.level : null);
          if (currentLevelName) {
            // Find the mfl_uid for this level name
            const levelObj = facilityLevels.find(l => l.name === currentLevelName);
            return levelObj?.mfl_uid || currentLevelName;
          }
          return null;
        case "ownership":
          // Current facility has ownership as string name from mfl_details
          // But we need to compare with mfl_uid from request, so get the mfl_uid
          const currentOwnershipName = current.ownership?.name || current.ownership_name || (typeof current.ownership === 'string' ? current.ownership : null);
          if (currentOwnershipName) {
            // Find the mfl_uid for this ownership name
            const ownershipObj = ownershipTypes.find(o => o.name === currentOwnershipName);
            return ownershipObj?.mfl_uid || currentOwnershipName;
          }
          return null;
        case "authority":
          // Current facility has authority as string name from mfl_details
          // But we need to compare with mfl_uid from request, so get the mfl_uid
          const currentAuthorityName = current.authority?.name || current.authority_name || (typeof current.authority === 'string' ? current.authority : null);
          if (currentAuthorityName) {
            // Find the mfl_uid for this authority name
            const authorityObj = authorityTypes.find(a => a.name === currentAuthorityName);
            return authorityObj?.mfl_uid || currentAuthorityName;
          }
          return null;
        case "status":
          return current.status;
        case "licensed":
          return current.licensed;
        case "address":
          return current.address;
        case "contact_personname":
          return current.contact_personname;
        case "contact_personemail":
          return current.contact_personemail;
        case "contact_personmobile":
          return current.contact_personmobile;
        case "latitude":
          return current.latitude;
        case "longitude":
          return current.longitude;
        case "opening_date":
          return current.opening_date;
        case "bed_capacity":
          return current.bed_capacity;
        case "services":
          if (Array.isArray(current.services)) return current.services;
          if (typeof current.services === "string") {
            try {
              return JSON.parse(current.services);
            } catch {
              return current.services.split(",").map(s => s.trim());
            }
          }
          return [];
        case "region_uid":
          return current.region?.mfl_uid || current.region_uid || null;
        case "district_uid":
          return current.district?.mfl_uid || current.district_uid || null;
        case "admin_unit_id":
        case "subcounty_id":
          return current.admin_unit_id || current.subcounty_id || null;
        default:
          return null;
      }
    };

    const fields = [
      { key: "name", label: "Field Station Name" },
      { key: "level", label: "Level" },
      { key: "ownership", label: "Ownership" },
      { key: "authority", label: "Authority" },
      { key: "status", label: "Status" },
      { key: "licensed", label: "Operationally Verified" },
      { key: "address", label: "Address" },
      { key: "contact_personname", label: "Contact Person" },
      { key: "contact_personemail", label: "Contact Email" },
      { key: "contact_personmobile", label: "Contact Mobile" },
      { key: "latitude", label: "Latitude" },
      { key: "longitude", label: "Longitude" },
      { key: "opening_date", label: "Opening Date" },
      { key: "bed_capacity", label: "Agent Capacity" },
      { key: "region_uid", label: "Region" },
      { key: "district_uid", label: "District" },
      { key: "admin_unit_id", label: "Subcounty / Admin Unit" },
      { key: "subcounty_id", label: "Subcounty / Admin Unit" },
      { key: "services", label: "Services" },
    ];

    // Filter to only show fields that have actually changed
    // Also deduplicate admin_unit_id and subcounty_id (they're the same field)
    const changedFields = fields.filter(field => {
      // Skip subcounty_id if admin_unit_id is already in the list (they're the same)
      if (field.key === "subcounty_id") {
        const hasAdminUnitId = facilityData["admin_unit_id"] !== null && 
                               facilityData["admin_unit_id"] !== undefined && 
                               facilityData["admin_unit_id"] !== "";
        if (hasAdminUnitId) return false;
      }
      
      const newValue = facilityData[field.key];
      const currentValue = getCurrentValue(field.key);
      return hasValueChanged(newValue, currentValue);
    });

    return changedFields;
  }, [request, facilityData, currentFacility, facilityLevels, ownershipTypes, authorityTypes]);

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

  const getAdminUnitName = (unitId) => {
    if (!unitId) return "—";
    const unit = adminUnits.find((u) => u.id === parseInt(unitId) || u.id === unitId);
    return unit?.name || unitId;
  };

  // Helper to format new value for display
  const formatNewValue = (fieldName, newValue) => {
    switch (fieldName) {
      case "level":
        return newValue ? getLevelName(newValue) : "";
      case "ownership":
        return newValue ? getOwnershipName(newValue) : "";
      case "authority":
        return newValue ? getAuthorityName(newValue) : "";
      case "licensed":
        return typeof newValue === "boolean" ? newValue : "";
      case "services":
        return Array.isArray(newValue) && newValue.length > 0 ? newValue.join(", ") : "";
      case "region_uid":
      case "district_uid":
        // For now, just show the UID. In a real app, you might want to look up the name
        return newValue || "";
      case "admin_unit_id":
      case "subcounty_id":
        return newValue ? getAdminUnitName(newValue) : "";
      default:
        return newValue || "";
    }
  };

  // Helper to format current value for display
  const formatCurrentValue = (fieldName) => {
    const current = currentFacility;
    if (!current) return "N/A";

    switch (fieldName) {
      case "name":
        return current.name || "N/A";
      case "level":
        return current.level?.name || current.level_name || current.facility_level?.name || "N/A";
      case "ownership":
        return current.ownership?.name || current.ownership_name || "N/A";
      case "authority":
        return current.authority?.name || current.authority_name || "N/A";
      case "status":
        return current.status || "N/A";
      case "licensed":
        return current.licensed !== undefined ? (current.licensed ? "Yes" : "No") : "N/A";
      case "address":
        return current.address || "N/A";
      case "contact_personname":
        return current.contact_personname || "N/A";
      case "contact_personemail":
        return current.contact_personemail || "N/A";
      case "contact_personmobile":
        return current.contact_personmobile || "N/A";
      case "latitude":
        return current.latitude || "N/A";
      case "longitude":
        return current.longitude || "N/A";
      case "opening_date":
        return current.opening_date || "N/A";
      case "bed_capacity":
        return current.bed_capacity || "N/A";
      case "services":
        if (Array.isArray(current.services) && current.services.length > 0) {
          return current.services.join(", ");
        }
        if (typeof current.services === "string" && current.services) {
          return current.services;
        }
        return "None";
      case "region_uid":
        return current.region?.name || current.region_name || "N/A";
      case "district_uid":
        return current.district?.name || current.district_name || "N/A";
      case "admin_unit_id":
      case "subcounty_id":
        return current.subcounty?.name || current.subcounty_name || current.admin_unit?.name || "N/A";
      default:
        return "N/A";
    }
  };

  if (loading) {
    return (
      <div class="page-header">
        <h2>Request Details</h2>
        <div>Loading request details...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div class="page-header">
        <h2>Request Details</h2>
        <div class="alert alert-danger">{error}</div>
      </div>
    );
  }

  if (!request) {
    return (
      <div class="page-header">
        <h2>Request Details</h2>
        <div class="alert alert-warning">Request not found</div>
      </div>
    );
  }

  const documents = request?.documents || [];
  const approvals = request?.approvals || [];

  return (<>

    <div class="page-header">
      <div class="page-header-left">
        <h2>Request Details</h2>
        <div class="subtitle">Review request information and approvals</div>
      </div>
      <button class="btn-back" onClick={() => history.push("/requests")}>
        <i class="bi bi-arrow-left"></i> Back to List
      </button>
    </div>

    {error && <div class="alert alert-danger">{error}</div>}

    <div class="request-info-card">
      <div class="request-meta">
        <div class="meta-item">
          <div class="meta-label">Request</div>
          <div class="meta-value">#{request.id} · {request.request_type || "—"}</div>
        </div>
        <div class="meta-item">
          <div class="meta-label">Status</div>
          <div class="meta-value">
            <span class={`status-badge ${request.current_status || "pending"}`}>{request.current_status || "pending"}</span>
          </div>
        </div>
        <div class="meta-item">
          <div class="meta-label">Current Stage</div>
          <div class="meta-value">{getStageLabel(request.current_stage)}</div>
        </div>
        <div class="meta-item">
          <div class="meta-label">Initiated By</div>
          <div class="meta-value">{request.initiated_by_name || request.initiated_by_email || "—"}</div>
        </div>
        <div class="meta-item">
          <div class="meta-label">Created</div>
          <div class="meta-value">
            {request.createdAt
              ? new Date(request.createdAt).toLocaleString("en-US", {
                  year: "numeric",
                  month: "2-digit",
                  day: "2-digit",
                  hour: "2-digit",
                  minute: "2-digit",
                  second: "2-digit",
                })
              : "—"}
          </div>
        </div>
      </div>
    </div>

    <div class="workflow-section">
      <div class="section-title">
        <i class="bi bi-diagram-3"></i>
        Workflow Progress
      </div>
      <div class="workflow-steps">
        {workflowProgress.length > 0 && <div class={`workflow-line ${request.current_status === "approved" ? "completed" : ""}`}></div>}
        {workflowProgress.map((step, index) => {
          const stepClass = step.status === "approved" ? "completed" : step.status === "rejected" ? "rejected" : step.status === "current" ? "current" : "";
          return (
            <div key={step.key} class="workflow-step">
              <div class={`step-circle ${stepClass}`}>
                {step.isApproved && <i class="bi bi-check"></i>}
                {step.isRejected && <i class="bi bi-x"></i>}
                {step.isCurrent && <i class="bi bi-clock"></i>}
              </div>
              <div class="step-label">{step.label}</div>
              <span class={`step-status ${step.status}`}>
                {step.isApproved ? "Approved" : step.isRejected ? "Rejected" : step.isCurrent ? "Current" : "Pending"}
              </span>
            </div>
          );
        })}
      </div>
    </div>

    {request.request_type === "update" ? (
      <div class="facility-data-section">
        <div class="section-title">
          <i class="bi bi-geo-alt"></i>
          Field Station Update Details
        </div>

        {facilityLoading && (
          <div class="text-muted" style={{ padding: "0.75rem 0" }}>
            Loading current field station details...
          </div>
        )}

        {getChangedFields.length === 0 ? (
          <div class="text-muted" style={{ padding: "1rem" }}>
            No changes detected in this update request.
          </div>
        ) : (
          <div class="table-responsive">
            <table class="table table-bordered">
              <thead style={{ backgroundColor: "#f8f9fa" }}>
                <tr>
                  <th style={{ width: "25%" }}>Field</th>
                  <th style={{ width: "35%" }}>Current Value</th>
                  <th style={{ width: "40%" }}>New Value</th>
                </tr>
              </thead>
              <tbody>
                {getChangedFields.map((field) => {
                  const newValue = facilityData[field.key];
                  const currentValueDisplay = formatCurrentValue(field.key);
                  const newValueDisplay = formatNewValue(field.key, newValue);

                  // Special handling for licensed (boolean with badge)
                  if (field.key === "licensed") {
                    return (
                      <tr key={field.key}>
                        <td><strong>{field.label}</strong></td>
                        <td>
                          <span class={`status-badge ${currentFacility?.licensed ? "approved" : "rejected"}`}>
                            {currentFacility?.licensed ? "Yes" : "No"}
                          </span>
                        </td>
                        <td>
                          <span class={`status-badge ${newValue ? "approved" : "rejected"}`}>
                            {newValue ? "Yes" : "No"}
                          </span>
                        </td>
                      </tr>
                    );
                  }

                  return (
                    <tr key={field.key}>
                      <td><strong>{field.label}</strong></td>
                      <td>{currentValueDisplay}</td>
                      <td>{newValueDisplay}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    ) : (
      <div class="facility-data-section">
        <div class="section-title">
          <i class="bi bi-geo-alt"></i>
          Field Station Data
        </div>
        <div class="data-grid">
          <div class="data-item">
            <span class="data-label">Field Station Name</span>
            <span class="data-value">{facilityData.name || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Address</span>
            <span class="data-value">{facilityData.address || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Longitude</span>
            <span class="data-value">{facilityData.longitude || "N/A"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Short Name</span>
            <span class="data-value">{facilityData.short_name || facilityData.name || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Contact Person</span>
            <span class="data-value">{facilityData.contact_personname || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Latitude</span>
            <span class="data-value">{facilityData.latitude || "N/A"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Ownership</span>
            <span class="data-value">{getOwnershipName(facilityData.ownership)}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Contact Email</span>
            <span class="data-value">{facilityData.contact_personemail || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Admin Unit ID</span>
            <span class="data-value">{facilityData.admin_unit_id || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Authority</span>
            <span class="data-value">{getAuthorityName(facilityData.authority)}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Contact Mobile</span>
            <span class="data-value">{facilityData.contact_personmobile || "—"}</span>
          </div>
          <div class="data-item">
            <span class="data-label">Operationally Verified</span>
            <span class="data-value">
              <span class={`status-badge ${facilityData.licensed ? "approved" : "rejected"}`}>
                {facilityData.licensed ? "Yes" : "No"}
              </span>
            </span>
          </div>
          <div class="data-item">
            <span class="data-label">Agent Capacity</span>
            <span class="data-value">{facilityData.bed_capacity || "N/A"}</span>
          </div>
          {facilityData.level && (
            <div class="data-item">
              <span class="data-label">Level</span>
              <span class="data-value">{getLevelName(facilityData.level)}</span>
            </div>
          )}
          {facilityData.contact_persontitle && (
            <div class="data-item">
              <span class="data-label">Contact Title</span>
              <span class="data-value">{facilityData.contact_persontitle}</span>
            </div>
          )}
          {facilityData.opening_date && (
            <div class="data-item">
              <span class="data-label">Opening Date</span>
              <span class="data-value">{facilityData.opening_date}</span>
            </div>
          )}
        </div>

        {facilityData.services && facilityData.services.length > 0 && (
          <>
            <div class="section-title" style={{marginTop: '1.5rem'}}>
              <i class="bi bi-heart-pulse"></i>
              Services
            </div>
            <div class="services-badges">
              {facilityData.services.map((service, index) => (
                <span key={index} class="service-badge">{service}</span>
              ))}
            </div>
          </>
        )}
      </div>
    )}

    <div class="approval-history">
      <div class="section-title">
        <i class="bi bi-clock-history"></i>
        Approval History
      </div>
      {approvals.length === 0 ? (
        <div class="text-muted" style={{ padding: "1rem" }}>No approval history yet.</div>
      ) : (
        approvals.map((approval) => (
          <div key={approval.id} class="history-item">
            <div class="history-header">
              <div>
                <span class="history-title">{getStageLabel(approval.stage)} - </span>
                <span class={`status-badge ${approval.action}`}>{approval.action}</span>
              </div>
              <span class="history-time">
                {approval.createdAt
                  ? new Date(approval.createdAt).toLocaleString("en-US", {
                      year: "numeric",
                      month: "2-digit",
                      day: "2-digit",
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                    })
                  : "—"}
              </span>
            </div>
            <div class="history-by">By: {approval.approver_name || approval.approver_email || "—"}</div>
            {approval.comments && (
              <div class="history-comment">Comments: {approval.comments}</div>
            )}
          </div>
        ))
      )}
    </div>

    <div class="documents-section">
      <div class="section-title">
        <i class="bi bi-file-earmark-text"></i>
        Supporting Documents
      </div>
      {documents.length === 0 ? (
        <div class="text-muted" style={{ padding: "1rem" }}>No documents attached.</div>
      ) : (
        documents.map((doc) => (
          <div key={doc.id} class="document-item">
            <div class="document-info">
              <div class="document-name">{doc.original_filename || doc.filename || "Document"}</div>
              <div class="document-meta">
                {doc.file_size ? `${(doc.file_size / 1024).toFixed(2)} KB` : ""}
                {doc.file_size && doc.mime_type ? " • " : ""}
                {doc.mime_type || ""}
              </div>
            </div>
            <button
              class="btn-download"
              onClick={async () => {
                try {
                  const blob = await RequestsApi.downloadDocument(request.id, doc.id);
                  const url = window.URL.createObjectURL(blob);
                  const a = document.createElement("a");
                  a.href = url;
                  a.download = doc.original_filename || doc.filename || "document";
                  document.body.appendChild(a);
                  a.click();
                  window.URL.revokeObjectURL(url);
                  document.body.removeChild(a);
                } catch (e) {
                  alert("Failed to download document: " + (e?.response?.data?.error || e.message));
                }
              }}
            >
              <i class="bi bi-download"></i> Download
            </button>
          </div>
        ))
      )}
    </div>

    {canApprove(request) && (
      <div class="review-actions-section">
        <div class="section-title">
          <i class="bi bi-pencil-square"></i>
          Review Actions
        </div>
        <div class="review-form">
          <div class="form-group">
            <label class="form-label">Comments</label>
            <textarea
              class="form-textarea"
              placeholder="Enter your review comments here..."
              value={approvalComments}
              onChange={(e) => setApprovalComments(e.target.value)}
            ></textarea>
          </div>
          {request.current_status === "pending" && (
            <div class="form-group">
              <label class="form-label">Rejection Reason (required if rejecting)</label>
              <textarea
                class="form-textarea"
                placeholder="Enter rejection reason..."
                value={rejectionReason}
                onChange={(e) => setRejectionReason(e.target.value)}
              ></textarea>
            </div>
          )}
          <div class="action-buttons">
            <button
              class="btn btn-approve"
              onClick={handleApprove}
              disabled={saving || request.current_status !== "pending"}
            >
              <i class="bi bi-check-circle"></i>
              Approve Request
            </button>
            <button
              class="btn btn-reject"
              onClick={handleReject}
              disabled={saving || request.current_status !== "pending" || !rejectionReason.trim()}
              style={{ 
                backgroundColor: "#dc3545", 
                color: "white", 
                borderColor: "#dc3545",
                opacity: (saving || request.current_status !== "pending" || !rejectionReason.trim()) ? 0.6 : 1
              }}
            >
              <i class="bi bi-x-circle"></i>
              Reject Request
            </button>
          </div>
        </div>
      </div>
    )}

  </>

  );
}
