import React, { useEffect, useState, useRef } from "react";
import {
  FacilitiesApi,
  FacilityLevelsApi,
  OwnershipTypesApi,
  AuthorityTypesApi,
} from "../../../helpers/api";
import { AVAILABLE_SERVICES } from "../../../helpers/services";

const formatIdentifier = (identifier) => {
  if (!identifier || typeof identifier !== "string") return identifier || "—";
  // Remove first 6 characters (800802) if identifier is longer than 6 characters
  return identifier.length > 6 ? identifier.slice(6) : identifier;
};

const DirectUpdates = () => {
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [availableFacilities, setAvailableFacilities] = useState([]);
  const [selectedFacilityId, setSelectedFacilityId] = useState("");
  const [selectedFacilityMflUid, setSelectedFacilityMflUid] = useState("");
  const [selectedFacility, setSelectedFacility] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const searchContainerRef = useRef(null);
  const [form, setForm] = useState({
    name: "",
    short_name: "",
    level: "",
    ownership: "",
    authority: "",
    status: "",
    reporting: false,
    licensed: false,
    address: "",
    contact_personemail: "",
    contact_personmobile: "",
    contact_personname: "",
    contact_persontitle: "",
    longitude: "",
    latitude: "",
    opening_date: "",
    closing_date: "",
    bed_capacity: "",
    services: [],
  });

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
      resetForm();
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await FacilitiesApi.get(id, {
        include: "admin_unit,facility_level,ownership,authority",
      });
      setSelectedFacility(data);
      // Store mfl_uid for future API calls
      if (data.mfl_uid) {
        setSelectedFacilityMflUid(data.mfl_uid);
      }
      
      // Helper function to find mfl_uid by name (API might return names instead of mfl_uid)
      const findMflUidByName = (name, lookupArray) => {
        if (!name || !lookupArray || lookupArray.length === 0) return "";
        // If name is already an mfl_uid (check if it matches any mfl_uid in lookup)
        const foundById = lookupArray.find(item => item.mfl_uid === name);
        if (foundById) return name;
        // Otherwise, find by name
        const found = lookupArray.find(item => item.name === name || item.name === name?.trim());
        return found?.mfl_uid || "";
      };
      
      // Get mfl_uid values - handle both name and mfl_uid formats
      const levelValue = data.level?.mfl_uid || data.level_mfl_uid || 
        (typeof data.level === 'string' ? findMflUidByName(data.level, facilityLevels) : "") ||
        (data.facility_level?.mfl_uid) || "";
      const ownershipValue = data.ownership?.mfl_uid || data.ownership_mfl_uid || 
        (typeof data.ownership === 'string' ? findMflUidByName(data.ownership, ownershipTypes) : "") || "";
      const authorityValue = data.authority?.mfl_uid || data.authority_mfl_uid || 
        (typeof data.authority === 'string' ? findMflUidByName(data.authority, authorityTypes) : "") || "";
      
      setForm({
        name: data.name || "",
        short_name: data.short_name || "",
        level: levelValue,
        ownership: ownershipValue,
        authority: authorityValue,
        status: data.status || "",
        reporting: data.reporting || false,
        licensed: data.licensed || false,
        address: data.address || "",
        contact_personemail: data.contact_personemail || "",
        contact_personmobile: data.contact_personmobile || "",
        contact_personname: data.contact_personname || "",
        contact_persontitle: data.contact_persontitle || "",
        longitude: data.longitude || "",
        latitude: data.latitude || "",
        opening_date: data.opening_date || "",
        closing_date: data.closing_date || "",
        bed_capacity: data.bed_capacity || "",
        services: Array.isArray(data.services)
          ? data.services
          : data.services
          ? typeof data.services === "string"
            ? JSON.parse(data.services)
            : []
          : [],
      });
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

  function resetForm() {
    setForm({
      name: "",
      short_name: "",
      level: "",
      ownership: "",
      authority: "",
      status: "",
      reporting: false,
      licensed: false,
      address: "",
      contact_personemail: "",
      contact_personmobile: "",
      contact_personname: "",
      contact_persontitle: "",
      longitude: "",
      latitude: "",
      opening_date: "",
      closing_date: "",
      bed_capacity: "",
      services: [],
    });
    setError("");
    setSuccess("");
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!selectedFacilityId) {
      setError("Please select a field station to update");
      return;
    }
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const payload = {
        name: form.name || null,
        short_name: form.short_name || null,
        level: form.level || null,
        ownership: form.ownership || null,
        authority: form.authority || null,
        status: form.status || null,
        reporting: form.reporting || false,
        licensed: form.licensed || false,
        address: form.address || null,
        contact_personemail: form.contact_personemail || null,
        contact_personmobile: form.contact_personmobile || null,
        contact_personname: form.contact_personname || null,
        contact_persontitle: form.contact_persontitle || null,
        longitude: form.longitude ? Number(form.longitude) : null,
        latitude: form.latitude ? Number(form.latitude) : null,
        bed_capacity: form.bed_capacity ? Number(form.bed_capacity) : null,
        opening_date: form.opening_date || null,
        closing_date: form.closing_date || null,
        services: Array.isArray(form.services) ? form.services : [],
      };
      await FacilitiesApi.update(selectedFacilityMflUid || selectedFacilityId, payload);
      setSuccess("Field station updated successfully");
      setError("");
      // Reset form and selection after successful update
      setSelectedFacilityId("");
      setSelectedFacilityMflUid("");
      setSelectedFacility(null);
      setSearchQuery("");
      setAvailableFacilities([]);
      setShowSearchResults(false);
      resetForm();
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to update field station");
      setSuccess("");
    } finally {
      setSaving(false);
    }
  }

  useEffect(() => {
    loadLookups();
  }, []);

  // Update form values when lookups are loaded and facility is selected
  useEffect(() => {
    if (selectedFacility && facilityLevels.length > 0 && ownershipTypes.length > 0 && authorityTypes.length > 0) {
      // Helper function to find mfl_uid by name
      const findMflUidByName = (name, lookupArray) => {
        if (!name || !lookupArray || lookupArray.length === 0) return "";
        // If name is already an mfl_uid (check if it matches any mfl_uid in lookup)
        const foundById = lookupArray.find(item => item.mfl_uid === name);
        if (foundById) return name;
        // Otherwise, find by name
        const found = lookupArray.find(item => item.name === name || item.name === name?.trim());
        return found?.mfl_uid || "";
      };
      
      // Get mfl_uid values - handle both name and mfl_uid formats
      const levelValue = selectedFacility.level?.mfl_uid || selectedFacility.level_mfl_uid || 
        (typeof selectedFacility.level === 'string' ? findMflUidByName(selectedFacility.level, facilityLevels) : "") ||
        (selectedFacility.facility_level?.mfl_uid) || "";
      const ownershipValue = selectedFacility.ownership?.mfl_uid || selectedFacility.ownership_mfl_uid || 
        (typeof selectedFacility.ownership === 'string' ? findMflUidByName(selectedFacility.ownership, ownershipTypes) : "") || "";
      const authorityValue = selectedFacility.authority?.mfl_uid || selectedFacility.authority_mfl_uid || 
        (typeof selectedFacility.authority === 'string' ? findMflUidByName(selectedFacility.authority, authorityTypes) : "") || "";
      
      // Update form with resolved mfl_uid values
      setForm(prev => ({
        ...prev,
        level: levelValue,
        ownership: ownershipValue,
        authority: authorityValue,
      }));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedFacility, facilityLevels, ownershipTypes, authorityTypes]);

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
        <h2>Direct Field Station Update</h2>
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

      <form id="facilityUpdateForm" onSubmit={handleSubmit}>
        <div className="form-card">
          <h3 className="form-section-title">Select Field Station to Update</h3>
          <div className="form-group" style={{ position: "relative" }} ref={searchContainerRef}>
            <label className="form-label">Search Field Station<span className="required">*</span></label>
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
                  resetForm();
                }
              }}
              onFocus={() => {
                if (availableFacilities.length > 0) {
                  setShowSearchResults(true);
                }
              }}
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
                <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Status</div>
                <div className="facility-info-value">{selectedFacility.status || "N/A"}</div>
              </div>
            </div>
          )}
        </div>

        {selectedFacility && (
          <>
            <div className="form-card">
              <h3 className="form-section-title">Basic Field Station Information</h3>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Field Station Name<span className="required">*</span></label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter field station name"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    required
                  />
                  <div className="form-text">This will be saved in the admin_units table</div>
                </div>
                <div className="form-group">
                  <label className="form-label">Short Name</label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter short name (optional)"
                    value={form.short_name}
                    onChange={(e) => setForm({ ...form, short_name: e.target.value })}
                  />
                  <div className="form-text">Common or trade name (optional)</div>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Station Tier</label>
                  <select
                    className="form-select"
                    value={form.level}
                    onChange={(e) => setForm({ ...form, level: e.target.value })}
                  >
                    <option value="">Select station tier</option>
                    {facilityLevels.map((l) => (
                      <option key={l.mfl_uid || l.id} value={l.mfl_uid || ""}>
                        {l.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Ownership Type</label>
                  <select
                    className="form-select"
                    value={form.ownership}
                    onChange={(e) => setForm({ ...form, ownership: e.target.value })}
                  >
                    <option value="">Select ownership</option>
                    {ownershipTypes.map((o) => (
                      <option key={o.mfl_uid || o.id} value={o.mfl_uid || ""}>
                        {o.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Authority Type</label>
                  <select
                    className="form-select"
                    value={form.authority}
                    onChange={(e) => setForm({ ...form, authority: e.target.value })}
                  >
                    <option value="">Select authority</option>
                    {authorityTypes.map((a) => (
                      <option key={a.mfl_uid || a.id} value={a.mfl_uid || ""}>
                        {a.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Status</label>
                  <select
                    className="form-select"
                    value={form.status}
                    onChange={(e) => setForm({ ...form, status: e.target.value })}
                  >
                    <option value="">Select status</option>
                    <option value="Functional">Functional</option>
                    <option value="Non-Functional">Non-Functional</option>
                  </select>
                </div>
                <div className="form-group">
                  <label className="form-label">Opening Date</label>
                  <input
                    type="date"
                    className="form-control"
                    value={form.opening_date}
                    onChange={(e) => setForm({ ...form, opening_date: e.target.value })}
                  />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">
                    <input
                      type="checkbox"
                      checked={form.reporting}
                      onChange={(e) => setForm({ ...form, reporting: e.target.checked })}
                      style={{ marginRight: "0.5rem" }}
                    />
                    Reporting Field Station
                  </label>
                </div>
                <div className="form-group">
                  <label className="form-label">
                    <input
                      type="checkbox"
                      checked={form.licensed}
                      onChange={(e) => setForm({ ...form, licensed: e.target.checked })}
                      style={{ marginRight: "0.5rem" }}
                    />
                    Operationally Verified Station
                  </label>
                </div>
              </div>
            </div>

            <div className="form-card">
              <h3 className="form-section-title">Location & Contact</h3>
              <div className="form-group">
                <label className="form-label">Address</label>
                <textarea
                  className="form-control"
                  placeholder="Enter complete physical address"
                  value={form.address}
                  onChange={(e) => setForm({ ...form, address: e.target.value })}
                />
              </div>
            </div>

            <div className="form-card">
              <h3 className="form-section-title">Contact Person Details</h3>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Contact Person Name</label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter contact person name"
                    value={form.contact_personname}
                    onChange={(e) => setForm({ ...form, contact_personname: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Contact Person Title</label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter title (e.g., Field Supervisor)"
                    value={form.contact_persontitle}
                    onChange={(e) => setForm({ ...form, contact_persontitle: e.target.value })}
                  />
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Contact Person Email</label>
                  <input
                    type="email"
                    className="form-control"
                    placeholder="contact@statgate.example"
                    value={form.contact_personemail}
                    onChange={(e) => setForm({ ...form, contact_personemail: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Contact Person Mobile</label>
                  <input
                    type="tel"
                    className="form-control"
                    placeholder="+256 XXX XXX XXX"
                    value={form.contact_personmobile}
                    onChange={(e) => setForm({ ...form, contact_personmobile: e.target.value })}
                  />
                </div>
              </div>
            </div>

            <div className="form-card">
              <h3 className="form-section-title">Geographic Location</h3>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Latitude</label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="e.g., 0.3476"
                    value={form.latitude}
                    onChange={(e) => setForm({ ...form, latitude: e.target.value })}
                  />
                  <div className="form-text">Decimal degrees format</div>
                </div>
                <div className="form-group">
                  <label className="form-label">Longitude</label>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="e.g., 32.5825"
                    value={form.longitude}
                    onChange={(e) => setForm({ ...form, longitude: e.target.value })}
                  />
                  <div className="form-text">Decimal degrees format</div>
                </div>
              </div>
            </div>

            <div className="form-card">
              <h3 className="form-section-title">Field Operations Profile</h3>
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">Agent Capacity</label>
                  <input
                    type="number"
                    className="form-control"
                    placeholder="Enter planned agent capacity"
                    min="0"
                    value={form.bed_capacity}
                    onChange={(e) => setForm({ ...form, bed_capacity: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Closing Date</label>
                  <input
                    type="date"
                    className="form-control"
                    value={form.closing_date}
                    onChange={(e) => setForm({ ...form, closing_date: e.target.value })}
                  />
                </div>
              </div>

              <div className="form-group" style={{ marginTop: "0.75rem" }}>
                <label className="form-label">Operational Capabilities</label>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: "0.5rem", marginTop: "0.5rem" }}>
                  {AVAILABLE_SERVICES.map((service) => (
                    <label
                      key={service}
                      style={{ display: "flex", alignItems: "center", fontSize: "0.85rem", cursor: "pointer" }}
                    >
                      <input
                        type="checkbox"
                        style={{ marginRight: "0.5rem" }}
                        checked={form.services.includes(service)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setForm({ ...form, services: [...form.services, service] });
                          } else {
                            setForm({ ...form, services: form.services.filter((s) => s !== service) });
                          }
                        }}
                      />
                      {service}
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </>
        )}

        <div className="action-buttons">
          <button type="submit" className="btn btn-primary" disabled={saving || !selectedFacility}>
            <i className="bi bi-check-circle"></i> {saving ? "Updating..." : "Update Field Station"}
          </button>
          <button
            type="button"
            className="btn btn-outline"
              onClick={() => {
              setSelectedFacilityId("");
              setSelectedFacilityMflUid("");
              setSelectedFacility(null);
              setSearchQuery("");
              setAvailableFacilities([]);
              setShowSearchResults(false);
              setError("");
              setSuccess("");
              resetForm();
            }}
          >
            <i className="bi bi-arrow-counterclockwise"></i> Reset Form
          </button>
        </div>
      </form>

    </div>
  );
};

export default DirectUpdates;
