import React, { useEffect, useState, useRef } from "react";
import { FacilitiesApi } from "../../../helpers/api";

const formatIdentifier = (identifier) => {
  if (!identifier || typeof identifier !== "string") return identifier || "—";
  // Remove first 6 characters (800802) if identifier is longer than 6 characters
  return identifier.length > 6 ? identifier.slice(6) : identifier;
};

const DirectDeactivate = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [availableFacilities, setAvailableFacilities] = useState([]);
  const [selectedFacilityId, setSelectedFacilityId] = useState("");
  const [selectedFacilityMflUid, setSelectedFacilityMflUid] = useState("");
  const [selectedFacility, setSelectedFacility] = useState(null);
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [deactivating, setDeactivating] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [showConfirmModal, setShowConfirmModal] = useState(false);
  const searchContainerRef = useRef(null);

  async function searchFacilities(query) {
    if (!query || query.length < 2) {
      setAvailableFacilities([]);
      setShowSearchResults(false);
      return;
    }
    try {
      const params = { q: query, pageSize: 50 };
      const res = await FacilitiesApi.listPaged(params);
      setAvailableFacilities(res.rows || []);
      setShowSearchResults(true);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to search field stations");
      setAvailableFacilities([]);
      setShowSearchResults(false);
    }
  }

  async function loadFacility(id) {
    if (!id) {
      setSelectedFacility(null);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await FacilitiesApi.get(id);
      setSelectedFacility(data);
      // Store mfl_uid for future API calls
      if (data.mfl_uid) {
        setSelectedFacilityMflUid(data.mfl_uid);
      }
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load field station");
      setSelectedFacility(null);
    } finally {
      setLoading(false);
    }
  }

  function handleFacilitySelect(facility) {
    // Use mfl_uid for API calls
    const facilityMflUid = facility.mfl_uid || facility.id;
    setSelectedFacilityId(String(facility.id || facility.mfl_uid));
    setSelectedFacilityMflUid(facilityMflUid);
    setSearchQuery(facility.name || "");
    setShowSearchResults(false);
    setAvailableFacilities([]);
    loadFacility(facilityMflUid);
  }

  function handleSubmit(e) {
    e.preventDefault();
    if (!selectedFacilityId) {
      setError("Please select a field station to deactivate");
      return;
    }
    if (!reason.trim()) {
      setError("Reason for deactivation is required");
      return;
    }
    setShowConfirmModal(true);
  }

  async function confirmDeactivation() {
    setShowConfirmModal(false);
    setDeactivating(true);
    setError("");
    setSuccess("");
    try {
      // Update facility status to Non-Functional
      await FacilitiesApi.update(selectedFacilityMflUid || selectedFacilityId, {
        status: "Non-Functional",
      });
      setSuccess(`Field station "${selectedFacility?.name || 'selected station'}" has been deactivated successfully.`);
      // Reset form
      setSelectedFacilityId("");
      setSelectedFacilityMflUid("");
      setSelectedFacility(null);
      setReason("");
      setSearchQuery("");
      setAvailableFacilities([]);
      setShowSearchResults(false);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to deactivate field station");
    } finally {
      setDeactivating(false);
    }
  }

  function handleCancel() {
    setSelectedFacilityId("");
    setSelectedFacilityMflUid("");
    setSelectedFacility(null);
    setReason("");
    setSearchQuery("");
    setAvailableFacilities([]);
    setShowSearchResults(false);
    setError("");
    setSuccess("");
  }

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      if (searchQuery) {
        searchFacilities(searchQuery);
      } else {
        setAvailableFacilities([]);
        setShowSearchResults(false);
      }
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [searchQuery]);

  // Close search results when clicking outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (searchContainerRef.current && !searchContainerRef.current.contains(event.target)) {
        setShowSearchResults(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  return (
    <div>
      <div className="page-header">
        <h2>Direct Field Station Deactivation</h2>
      </div>

      {error && (
        <div className="alert alert-danger alert-dismissible fade show" role="alert">
          <i className="bi bi-exclamation-circle"></i> {error}
          <button
            type="button"
            className="btn-close"
            onClick={() => setError("")}
            aria-label="Close"
          />
        </div>
      )}

      {success && (
        <div className="alert alert-success alert-dismissible fade show" role="alert">
          <i className="bi bi-check-circle"></i> {success}
          <button
            type="button"
            className="btn-close"
            onClick={() => setSuccess("")}
            aria-label="Close"
          />
        </div>
      )}

      <form id="deactivateForm" onSubmit={handleSubmit}>
        <div className="form-card">
          <h3 className="form-section-title">Select Field Station to Deactivate</h3>
          <div className="form-group" ref={searchContainerRef} style={{ position: "relative" }}>
            <label className="form-label">Search Field Station <span className="required">*</span></label>
            <input
              type="text"
              className="form-control"
              placeholder="Search by name, historical ID, or identifier..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                if (!e.target.value) {
                  setSelectedFacilityId("");
                  setSelectedFacilityMflUid("");
                  setSelectedFacility(null);
                  setReason("");
                }
              }}
              onFocus={() => {
                if (availableFacilities.length > 0) {
                  setShowSearchResults(true);
                }
              }}
              required
            />
            <div className="form-text">Search by field station name, historical ID, or identifier.</div>
            
            {showSearchResults && availableFacilities.length > 0 && (
              <div
                style={{
                  position: "absolute",
                  top: "100%",
                  left: 0,
                  right: 0,
                  zIndex: 1000,
                  backgroundColor: "white",
                  border: "1px solid #ced4da",
                  borderRadius: "0.25rem",
                  marginTop: "0.25rem",
                  maxHeight: "300px",
                  overflowY: "auto",
                  boxShadow: "0 2px 8px rgba(0,0,0,0.15)",
                }}
              >
                {availableFacilities.map((facility) => (
                  <div
                    key={facility.id}
                    onClick={() => handleFacilitySelect(facility)}
                    style={{
                      padding: "0.75rem",
                      cursor: "pointer",
                      borderBottom: "1px solid #f0f0f0",
                      transition: "background-color 0.2s",
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.backgroundColor = "#f8f9fa";
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.backgroundColor = "white";
                    }}
                  >
                    <div style={{ fontWeight: "500" }}>{facility.name}</div>
                    <div style={{ fontSize: "0.875rem", color: "#6c757d", marginTop: "0.25rem" }}>
                      {facility.mfl_uid && `MFL: ${facility.mfl_uid}`}
                      {facility.identifier && ` | Identifier: ${formatIdentifier(facility.identifier)}`}
                      {facility.historical_id && ` | Historical ID: ${facility.historical_id}`}
                    </div>
                  </div>
                ))}
              </div>
            )}
            
            {showSearchResults && availableFacilities.length === 0 && searchQuery.length >= 2 && (
              <div
                style={{
                  position: "absolute",
                  top: "100%",
                  left: 0,
                  right: 0,
                  zIndex: 1000,
                  backgroundColor: "white",
                  border: "1px solid #ced4da",
                  borderRadius: "0.25rem",
                  marginTop: "0.25rem",
                  padding: "0.75rem",
                  boxShadow: "0 2px 8px rgba(0,0,0,0.15)",
                }}
              >
                <div style={{ color: "#6c757d" }}>No field stations found</div>
              </div>
            )}
            
            {loading && (
              <div className="form-text" style={{ marginTop: "0.5rem" }}>Loading field station details...</div>
            )}
          </div>

          {selectedFacility && (
            <div className="facility-info" style={{ display: "flex", flexWrap: "wrap", gap: "1.5rem", marginTop: "1rem", padding: "1rem", background: "#f8f9fa", borderRadius: "0.25rem" }}>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Field Station Name</div>
                <div className="facility-info-value">{selectedFacility.name}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Registry UID</div>
                <div className="facility-info-value">{selectedFacility.mfl_uid || "N/A"}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Identifier</div>
                <div className="facility-info-value">{formatIdentifier(selectedFacility.identifier) || "N/A"}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Current Status</div>
                <div className="facility-info-value">
                  <span style={{ 
                    padding: "0.25rem 0.5rem", 
                    borderRadius: "0.25rem",
                    backgroundColor: selectedFacility.status === "Functional" ? "#d4edda" : "#f8d7da",
                    color: selectedFacility.status === "Functional" ? "#155724" : "#721c24"
                  }}>
                    {selectedFacility.status || "N/A"}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {selectedFacility && (
          <div className="form-card">
            <h3 className="form-section-title">Deactivation Details</h3>
            <div className="form-group">
              <label className="form-label">Reason for Deactivation <span className="required">*</span></label>
              <textarea
                className="form-control"
                placeholder="Provide a reason for deactivation"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                required
                rows={4}
              />
              <div className="form-text">Please provide a detailed explanation for deactivating this field station.</div>
            </div>

            <div className="action-buttons">
              <button type="submit" className="btn btn-primary" disabled={deactivating || !selectedFacility}>
                <i className="bi bi-x-circle"></i> {deactivating ? "Deactivating..." : "Deactivate Field Station"}
              </button>
              <button type="button" className="btn btn-secondary" onClick={handleCancel}>
                <i className="bi bi-x"></i> Cancel
              </button>
            </div>
          </div>
        )}
      </form>

      {/* Confirmation Modal */}
      {showConfirmModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={() => setShowConfirmModal(false)}
        >
          <div
            className="modal-dialog modal-dialog-centered"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Confirm Deactivation</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => setShowConfirmModal(false)}
                  aria-label="Close"
                />
              </div>
              <div className="modal-body">
                <p>
                  Are you sure you want to deactivate <strong>"{selectedFacility?.name || 'this field station'}"</strong>?
                </p>
                <p className="text-muted mb-0">
                  This will set the field station status to <strong>Inactive</strong>.
                </p>
                {selectedFacility && (
                  <div className="mt-3 p-3 bg-light rounded" style={{ display: "flex", flexWrap: "wrap", gap: "1.5rem" }}>
                    <div>
                      <strong>Field Station:</strong> {selectedFacility.name}
                    </div>
                    <div>
                      <strong>Registry UID:</strong> {selectedFacility.mfl_uid || "N/A"}
                    </div>
                    <div>
                      <strong>Identifier:</strong> {selectedFacility.identifier || "N/A"}
                    </div>
                    <div>
                      <strong>Current Status:</strong>{" "}
                      <span
                        style={{
                          padding: "0.25rem 0.5rem",
                          borderRadius: "0.25rem",
                          backgroundColor: selectedFacility.status === "Functional" ? "#d4edda" : "#f8d7da",
                          color: selectedFacility.status === "Functional" ? "#155724" : "#721c24",
                        }}
                      >
                        {selectedFacility.status || "N/A"}
                      </span>
                    </div>
                  </div>
                )}
                {reason && (
                  <div className="mt-3">
                    <strong>Reason:</strong>
                    <div className="mt-1 p-2 bg-light rounded">{reason}</div>
                  </div>
                )}
              </div>
              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowConfirmModal(false)}
                  disabled={deactivating}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="btn btn-danger"
                  onClick={confirmDeactivation}
                  disabled={deactivating}
                >
                  <i className="bi bi-x-circle"></i>{" "}
                  {deactivating ? "Deactivating..." : "Confirm Deactivation"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DirectDeactivate;
