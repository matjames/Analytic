import React, { Fragment, useEffect, useState, useRef } from "react";
import { RequestsApi } from "../../helpers/api";
import './styles.css'

const formatIdentifier = (identifier) => {
  if (!identifier || typeof identifier !== "string") return identifier || "—";
  // Remove first 6 characters (800802) if identifier is longer than 6 characters
  return identifier.length > 6 ? identifier.slice(6) : identifier;
};

export default function RequestDeactivate() {
  const [reason, setReason] = useState("");
  const [documents, setDocuments] = useState([]);
  const [facilitySearchQuery, setFacilitySearchQuery] = useState("");
  const [availableFacilities, setAvailableFacilities] = useState([]);
  const [selectedFacility, setSelectedFacility] = useState(null);
  const [loadingFacilities, setLoadingFacilities] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const searchContainerRef = useRef(null);

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      if (facilitySearchQuery && facilitySearchQuery.trim().length >= 2) {
        loadFacilities(facilitySearchQuery);
      } else {
        setAvailableFacilities([]);
        setShowSearchResults(false);
      }
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [facilitySearchQuery]);

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

  async function loadFacilities(searchQuery = "") {
    if (!searchQuery || searchQuery.trim().length < 2) {
      setAvailableFacilities([]);
      setShowSearchResults(false);
      return;
    }
    setLoadingFacilities(true);
    setError("");
    try {
      const facilities = await RequestsApi.getFacilities(searchQuery);
      setAvailableFacilities(facilities || []);
      setShowSearchResults(true);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load facilities");
      setAvailableFacilities([]);
      setShowSearchResults(false);
    } finally {
      setLoadingFacilities(false);
    }
  }

  function handleFacilitySelect(facility) {
    setSelectedFacility(facility);
    setFacilitySearchQuery(facility.name || "");
    setShowSearchResults(false);
    setAvailableFacilities([]);
  }

  const [documentType, setDocumentType] = useState("");

  const allowedDocExtensions = /\.(pdf|jpe?g|png|gif|webp)$/i;

  function handleDocumentChange(e) {
    const files = Array.from(e.target.files || []);
    if (!documentType) {
      setError("Please select a supporting document type before choosing files.");
      e.target.value = "";
      return;
    }
    const valid = [];
    const invalid = [];
    files.forEach((file) => {
      if (allowedDocExtensions.test(file.name || "")) valid.push(file);
      else invalid.push(file.name || "unknown");
    });
    if (invalid.length > 0) {
      setError(`Only PDF and image files (JPG, PNG, GIF, WebP) are allowed. Rejected: ${invalid.join(", ")}`);
      e.target.value = "";
      return;
    }
    const wrapped = valid.map((file) => ({ file, type: documentType }));
    setDocuments((prev) => [...prev, ...wrapped]);
    setDocumentType("");
    e.target.value = "";
  }

  function removeDocument(index) {
    setDocuments((prev) => prev.filter((_, i) => i !== index));
  }

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    setSuccess("");

    if (!selectedFacility) {
      setError("Please select a facility to deactivate");
      setSubmitting(false);
      return;
    }
    if (!reason.trim()) {
      setError("Reason for deactivation is required");
      setSubmitting(false);
      return;
    }

    if (!documents || documents.length === 0) {
      setError("Supporting documents are required for deactivation requests");
      setSubmitting(false);
      return;
    }

    if (documents.some((d) => !d.type)) {
      setError("Please select a type for each supporting document.");
      setSubmitting(false);
      return;
    }

    try {
      if (!selectedFacility?.mfl_uid) {
        setError("Facility mfl_uid is missing; cannot submit request.");
        setSubmitting(false);
        return;
      }
      await RequestsApi.create(
        "deactivation",
        { deactivation_reason: reason },
        documents,
        selectedFacility.mfl_uid
      );
      setSuccess("Deactivation request submitted.");
      setReason("");
      setDocuments([]);
      setSelectedFacility(null);
      setFacilitySearchQuery("");
      setAvailableFacilities([]);
    } catch (err) {
      setError(err?.response?.data?.error || err.message || "Failed to submit deactivation");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Fragment>
      <div className="page-header">
        <h2>Deactivate Facility</h2>
      </div>

      <div className="alert alert-info">
        <i className="bi bi-info-circle"></i> Select a facility to submit a deactivation request. The request will be reviewed and approved before the facility is deactivated.
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
          <h3 className="form-section-title">Select Facility to Deactivate</h3>
          <div className="form-group" style={{ position: "relative" }} ref={searchContainerRef}>
            <label className="form-label">Search Facility<span className="required">*</span></label>
            <input
              type="text"
              className="form-control"
              placeholder="Search by name, historical ID, or identifier..."
              value={facilitySearchQuery}
              onChange={(e) => {
                setFacilitySearchQuery(e.target.value);
                if (!e.target.value) {
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
            <div className="form-text">Type to search for the facility you want to deactivate (searches by name, historical ID, or identifier)</div>
            
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
            
            {showSearchResults && availableFacilities.length === 0 && facilitySearchQuery.length >= 2 && !loadingFacilities && (
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
                <div style={{ color: "#6c757d" }}>No facilities found</div>
              </div>
            )}
          </div>

          {selectedFacility && (
            <div className="facility-info" style={{ display: "flex", flexWrap: "wrap", gap: "1.5rem", marginTop: "1rem", padding: "1rem", background: "#f8f9fa", borderRadius: "0.25rem" }}>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Facility Name</div>
                <div className="facility-info-value">{selectedFacility.name}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>MFL UID</div>
                <div className="facility-info-value">{selectedFacility.mfl_uid || "N/A"}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Identifier</div>
                <div className="facility-info-value">{formatIdentifier(selectedFacility.identifier) || "N/A"}</div>
              </div>
              <div className="facility-info-item">
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Status</div>
                <div className="facility-info-value">{selectedFacility.status || "N/A"}</div>
              </div>
            </div>
          )}
        </div>

        {selectedFacility && (
          <div className="form-card">
            <h3 className="form-section-title">Deactivation Details</h3>
            <div className="form-group">
              <label className="form-label">Reason for Deactivation<span className="required">*</span></label>
              <textarea
                className="form-control"
                placeholder="Provide a detailed reason for deactivating this facility"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                required
                rows={5}
              />
              <div className="form-text">Please provide a detailed explanation for deactivating this facility</div>
            </div>

            <div className="form-group" style={{ marginTop: "0.75rem" }}>
              <label className="form-label">Supporting Documents<span className="required">*</span></label>
              <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <select
                  className="form-select"
                  style={{ maxWidth: "260px" }}
                  value={documentType}
                  onChange={(e) => setDocumentType(e.target.value)}
                >
                  <option value="">Select document type</option>
                  <option value="District CAO Letter">District CAO Letter</option>
                  <option value="Operating License">Operating License</option>
                  <option value="District Council Minutes">District Council Minutes</option>
                </select>
                <input
                  type="file"
                  className="form-control"
                  multiple
                  accept=".pdf,.jpg,.jpeg,.png,.gif,.webp,application/pdf,image/jpeg,image/png,image/gif,image/webp"
                  onChange={handleDocumentChange}
                  required
                />
              </div>
              <div className="form-text">You can upload multiple documents — PDF and images (JPG, PNG, GIF, WebP) only. Documents are required for deactivation requests.</div>
            </div>

            {documents.length > 0 && (
              <div className="form-group" style={{ marginTop: "0.75rem" }}>
                <label className="form-label">Selected Documents</label>
                <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
                  {documents.map((doc, index) => (
                    <div
                      key={index}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        padding: "0.5rem",
                        background: "#f8f9fa",
                        borderRadius: "0.25rem",
                        gap: "0.5rem",
                      }}
                    >
                      <span
                        style={{
                          fontSize: "0.875rem",
                          flex: 1,
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {doc.file?.name || ""}
                      </span>
                      <span
                        style={{
                          fontSize: "0.8rem",
                          color: "var(--text-muted)",
                          minWidth: "200px",
                        }}
                      >
                        {doc.type}
                      </span>
                      <button
                        type="button"
                        className="btn btn-sm btn-outline-danger"
                        onClick={() => removeDocument(index)}
                        style={{ marginLeft: "0.5rem" }}
                      >
                        <i className="bi bi-x"></i> Remove
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        <div className="action-buttons">
          <button type="submit" className="btn btn-primary" disabled={submitting || !selectedFacility}>
            <i className="bi bi-send"></i> {submitting ? "Submitting..." : "Submit Deactivation Request"}
          </button>
          <button
            type="button"
            className="btn btn-outline"
            onClick={() => {
              setSelectedFacility(null);
              setFacilitySearchQuery("");
              setAvailableFacilities([]);
              setShowSearchResults(false);
              setReason("");
              setDocuments([]);
              setError("");
              setSuccess("");
            }}
          >
            <i className="bi bi-arrow-counterclockwise"></i> Reset Form
          </button>
        </div>
      </form>
    </Fragment>
  );
}

