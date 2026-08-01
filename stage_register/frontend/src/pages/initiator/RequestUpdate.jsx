import React, { Fragment, useEffect, useMemo, useState, useRef } from "react";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";
import {
  AuthorityTypesApi,
  FacilitiesApi,
  FacilityLevelsApi,
  LevelsApi,
  OwnershipTypesApi,
  RequestsApi,
  UnitsApi,
} from "../../helpers/api";
import { AVAILABLE_SERVICES } from "../../helpers/services";

const formatIdentifier = (identifier) => {
  if (!identifier || typeof identifier !== "string") return identifier || "—";
  // Remove first 6 characters (800802) if identifier is longer than 6 characters
  return identifier.length > 6 ? identifier.slice(6) : identifier;
};

export default function RequestUpdate() {
  const [form, setForm] = useState({
    facility_name: "",
    level: "",
    ownership: "",
    authority: "",
    status: "",
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
    admin_unit_id: "",
    subcounty_id: "",
    region_uid: "",
    district_uid: "",
  });
  const [documents, setDocuments] = useState([]);
  const [selectedFacility, setSelectedFacility] = useState(null);
  const [facilitySearchQuery, setFacilitySearchQuery] = useState("");
  const [availableFacilities, setAvailableFacilities] = useState([]);
  const [loadingFacilities, setLoadingFacilities] = useState(false);
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [treeData, setTreeData] = useState([]);
  const [levelsMap, setLevelsMap] = useState(new Map());
  const [showHierarchyModal, setShowHierarchyModal] = useState(false);
  const [hierarchySelectionType, setHierarchySelectionType] = useState(null); // 'region', 'district', or 'subcounty'
  const [selectedRegionName, setSelectedRegionName] = useState("");
  const [selectedDistrictName, setSelectedDistrictName] = useState("");
  const [selectedSubcountyName, setSelectedSubcountyName] = useState("");
  const [subcounties, setSubcounties] = useState([]);
  const [loadingSubcounties, setLoadingSubcounties] = useState(false);
  const [loadingLookups, setLoadingLookups] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const searchContainerRef = useRef(null);

  useEffect(() => {
    loadLookups();
  }, []);

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

  const serviceColumns = useMemo(() => {
    const chunk = [];
    const perCol = Math.ceil(AVAILABLE_SERVICES.length / 3);
    for (let i = 0; i < AVAILABLE_SERVICES.length; i += perCol) {
      chunk.push(AVAILABLE_SERVICES.slice(i, i + perCol));
    }
    return chunk;
  }, []);

  async function loadLookups() {
    setLoadingLookups(true);
    setError("");
    try {
      const [levels, ownerships, authorities, adminLevels, treeRes] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
        LevelsApi.list(),
        UnitsApi.tree(),
      ]);
      setFacilityLevels(levels || []);
      setOwnershipTypes(ownerships || []);
      setAuthorityTypes(authorities || []);
      setLevelsMap(new Map(adminLevels.map((l) => [l.id, l])));
      setTreeData(treeRes.tree || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load lookup data");
    } finally {
      setLoadingLookups(false);
    }
  }

  // Filter tree to show regions (level 2), districts (level 3), and subcounties (level 4)
  const filteredTree = useMemo(() => {
    if (!treeData.length) return [];
    const filter = (nodes) =>
      nodes
        .map((n) => {
          const level = levelsMap.get(n.levelId);
          if (level && level.level_number > 4) return null; // Only show up to subcounty level
          const kids = n.children ? filter(n.children) : [];
          return { ...n, children: kids };
        })
        .filter(Boolean);
    return filter(treeData);
  }, [treeData, levelsMap]);

  // Convert to rc-tree format
  const rcTreeData = useMemo(() => {
    const mapNodes = (nodes) =>
      nodes.map((n) => ({
        key: String(n.id),
        title: n.name,
        children: n.children ? mapNodes(n.children) : [],
        raw: n,
      }));
    return filteredTree.length ? mapNodes(filteredTree) : [];
  }, [filteredTree]);

  // Expand top-level territories by default.
  const defaultExpandedKeys = useMemo(() => {
    return rcTreeData.map((node) => node.key);
  }, [rcTreeData]);

  function handleHierarchySelect(node) {
    if (!node.raw || !node.raw.mfl_uid) return;
    
    const level = levelsMap.get(node.raw.levelId);
    if (!level) return;

    if (hierarchySelectionType === 'region' && level.level_number === 2) {
      setForm({ ...form, region_uid: node.raw.mfl_uid });
      setSelectedRegionName(node.raw.name || "");
      // Clear district and subcounty when region changes
      setForm(prev => ({ ...prev, district_uid: "", subcounty_id: "", admin_unit_id: "" }));
      setSelectedDistrictName("");
      setSelectedSubcountyName("");
    } else if (hierarchySelectionType === 'district' && level.level_number === 3) {
      setForm({ ...form, district_uid: node.raw.mfl_uid });
      setSelectedDistrictName(node.raw.name || "");
      // Clear subcounty when district changes
      setForm(prev => ({ ...prev, subcounty_id: "", admin_unit_id: "" }));
      setSelectedSubcountyName("");
      // Load subcounties for the selected district
      loadSubcountiesForDistrict(node.raw.mfl_uid);
    } else if (hierarchySelectionType === 'subcounty' && level.level_number === 4) {
      setForm({ ...form, subcounty_id: String(node.raw.id), admin_unit_id: String(node.raw.id) });
      setSelectedSubcountyName(node.raw.name || "");
    }
    setShowHierarchyModal(false);
    setHierarchySelectionType(null);
  }

  async function loadSubcountiesForDistrict(districtMflUid) {
    if (!districtMflUid) {
      setSubcounties([]);
      return;
    }
    setLoadingSubcounties(true);
    try {
      // Use the subtree endpoint to get subcounties under this district
      // Subcounties are level 4
      const { apiClient } = await import("../../helpers/api/client");
      const response = await apiClient.get(`/adminunits/${districtMflUid}/subtree`, {
        params: { levelId: 4 }
      });
      setSubcounties(response.data.descendants || []);
    } catch (e) {
      console.error("Failed to load subcounties:", e);
      // Fallback: try to get all subcounties (level 4)
      try {
        const allSubcounties = await UnitsApi.list({ levelId: 4 });
        setSubcounties(allSubcounties || []);
      } catch (fallbackError) {
        setSubcounties([]);
      }
    } finally {
      setLoadingSubcounties(false);
    }
  }

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

  async function loadFacilityDetails(mflUid) {
    if (!mflUid) {
      setSelectedFacility(null);
      setForm({
        facility_name: "",
        level: "",
        ownership: "",
        authority: "",
        status: "",
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
        admin_unit_id: "",
        subcounty_id: "",
        region_uid: "",
        district_uid: "",
      });
      setSelectedRegionName("");
      setSelectedDistrictName("");
      setSelectedSubcountyName("");
      setSubcounties([]);
      return;
    }
    setLoadingFacilities(true);
    setError("");
    try {
      // Uses /facilities/:id endpoint where :id is the mfl_uid
      // Backend queries mfl_details view using WHERE mfl_uid = $1
      const data = await FacilitiesApi.get(mflUid);
      setSelectedFacility(data);
      
      // Set hierarchy names from mfl_list data - handle both nested objects and string formats
      const regionName = data.region_name || 
                        data.region?.name || 
                        (typeof data.region === 'string' ? data.region : "") || 
                        "";
      const districtName = data.district_name || 
                          data.district?.name || 
                          (typeof data.district === 'string' ? data.district : "") || 
                          "";
      const subcountyName = data.subcounty_name || 
                           data.subcounty?.name || 
                           (typeof data.subcounty === 'string' ? data.subcounty : "") || 
                           "";
      
      setSelectedRegionName(regionName);
      setSelectedDistrictName(districtName);
      setSelectedSubcountyName(subcountyName);
      
      // Load subcounties for the facility's district
      const districtUid = data.district_uid || data.district?.mfl_uid;
      if (districtUid) {
        loadSubcountiesForDistrict(districtUid);
      }
      
      // Find mfl_uid from name for level, ownership, authority (since mfl_details only has names)
      const findMflUidByName = (name, lookupArray) => {
        if (!name || !lookupArray) return "";
        const found = lookupArray.find(item => item.name === name);
        return found?.mfl_uid || "";
      };
      
      const levelMflUid = data.level?.mfl_uid || data.level_mfl_uid || data.facility_level?.mfl_uid || 
        findMflUidByName(data.level?.name, facilityLevels);
      const ownershipMflUid = data.ownership?.mfl_uid || data.ownership_mfl_uid || 
        findMflUidByName(data.ownership?.name, ownershipTypes);
      const authorityMflUid = data.authority?.mfl_uid || data.authority_mfl_uid || 
        findMflUidByName(data.authority?.name, authorityTypes);
      
      setForm({
        facility_name: data.name || "",
        level: levelMflUid,
        ownership: ownershipMflUid,
        authority: authorityMflUid,
        status: data.status || "",
        licensed: data.licensed || false,
        address: data.address || "",
        contact_personemail: data.contact_personemail || "",
        contact_personmobile: data.contact_personmobile || "",
        contact_personname: data.contact_personname || "",
        contact_persontitle: data.contact_persontitle || "",
        longitude: data.longitude || "",
        latitude: data.latitude || "",
        opening_date: data.opening_date || "",
        bed_capacity: data.bed_capacity || "",
        services: Array.isArray(data.services)
          ? data.services
          : data.services
          ? typeof data.services === "string"
            ? JSON.parse(data.services)
            : []
          : [],
        admin_unit_id: data.admin_unit?.id || data.admin_unit_id || "",
        subcounty_id: data.subcounty?.id || data.subcounty_id || "",
        region_uid: data.region_uid || data.region?.mfl_uid || "",
        district_uid: data.district_uid || data.district?.mfl_uid || "",
      });
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load facility details");
      setSelectedFacility(null);
    } finally {
      setLoadingFacilities(false);
    }
  }

  function handleFacilitySelect(facility) {
    if (!facility?.mfl_uid) {
      setError("Facility mfl_uid is missing; cannot load details.");
      return;
    }
    setSelectedFacility(facility);
    setFacilitySearchQuery(facility.name || "");
    setShowSearchResults(false);
    setAvailableFacilities([]);
    loadFacilityDetails(facility.mfl_uid);
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
      setError("Please select a facility to update");
      setSubmitting(false);
      return;
    }

    if (!form.contact_personmobile.trim() || !form.contact_personemail.trim() || !form.address.trim()) {
      const msg = "Primary phone number, email address, and physical address are required.";
      setError(msg);
      window.alert(msg);
      setSubmitting(false);
      return;
    }

    if (!documents || documents.length === 0) {
      setError("Supporting documents are required for update requests");
      setSubmitting(false);
      return;
    }

    const facilityData = {
      name: form.facility_name || null,
      short_name: form.facility_name || null,
      level: form.level || null,
      ownership: form.ownership || null,
      authority: form.authority || null,
      status: form.status || null,
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
      admin_unit_id: form.subcounty_id ? parseInt(form.subcounty_id) : null,
      region_uid: form.region_uid || null,
      district_uid: form.district_uid || null,
    };

    try {
      if (!selectedFacility?.mfl_uid) {
        setError("Facility mfl_uid is missing; cannot submit request.");
        setSubmitting(false);
        return;
      }
      await RequestsApi.create("update", facilityData, documents, selectedFacility.mfl_uid);
      setSuccess("Update request submitted.");
      setForm({
        facility_name: "",
        level: "",
        ownership: "",
        authority: "",
        status: "",
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
        admin_unit_id: "",
        subcounty_id: "",
        region_uid: "",
        district_uid: "",
      });
      setDocuments([]);
      setSelectedFacility(null);
      setFacilitySearchQuery("");
      setAvailableFacilities([]);
      setSubcounties([]);
      setSelectedRegionName("");
      setSelectedDistrictName("");
      setSelectedSubcountyName("");
    } catch (err) {
      setError(err?.response?.data?.error || err.message || "Failed to submit update request");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Fragment>
      <div className="page-header">
        <h2>Facility Update Request Form</h2>
      </div>

      <div className="alert alert-info">
        <i className="bi bi-info-circle"></i> Select a facility to update its information. Changes will be submitted for approval.
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
          <h3 className="form-section-title">Select Facility to Update</h3>
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
                  setForm({
                    facility_name: "",
                    level: "",
                    ownership: "",
                    authority: "",
                    status: "",
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
                    admin_unit_id: "",
                    subcounty_id: "",
                  });
                  setSubcounties([]);
                }
              }}
              onFocus={() => {
                if (availableFacilities.length > 0) {
                  setShowSearchResults(true);
                }
              }}
              required
            />
            <div className="form-text">Type to search for the facility you want to update (searches by name, historical ID, or identifier)</div>
            
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
              {selectedFacility.level?.name && (
                <div className="facility-info-item">
                  <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Level</div>
                  <div className="facility-info-value">{selectedFacility.level?.name}</div>
                </div>
              )}
              {selectedFacility.ownership?.name && (
                <div className="facility-info-item">
                  <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Ownership</div>
                  <div className="facility-info-value">{selectedFacility.ownership?.name}</div>
                </div>
              )}
              {selectedFacility.authority?.name && (
                <div className="facility-info-item">
                  <div className="facility-info-label" style={{ fontWeight: "bold", marginBottom: "0.25rem" }}>Authority</div>
                  <div className="facility-info-value">{selectedFacility.authority?.name}</div>
                </div>
              )}
            </div>
          )}
        </div>

        {selectedFacility && (
          <>
        <div className="form-card">
          <h3 className="form-section-title">Update Basic Information</h3>
          <table
            className="table table-bordered"
            style={{ marginBottom: "0" }}
          >
            <thead style={{ backgroundColor: "#f8f9fa" }}>
              <tr>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "25%",
                  }}
                >
                  Field
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "35%",
                  }}
                >
                  Current Value
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "40%",
                  }}
                >
                  New Value
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Facility Name
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.name || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter new name"
                    style={{ margin: 0 }}
                    value={form.facility_name}
                    onChange={(e) => setForm({ ...form, facility_name: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Level
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.level?.name || selectedFacility.level_name || selectedFacility.facility_level?.name || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <select
                    className="form-select"
                    style={{ margin: 0 }}
                    value={form.level}
                    onChange={(e) => setForm({ ...form, level: e.target.value })}
                  >
                    <option value="">Keep current</option>
                    {facilityLevels.map((l) => (
                      <option key={l.mfl_uid || l.id} value={l.mfl_uid || ""}>
                        {l.name}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Ownership
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.ownership?.name || selectedFacility.ownership_name || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <select
                    className="form-select"
                    style={{ margin: 0 }}
                    value={form.ownership}
                    onChange={(e) => setForm({ ...form, ownership: e.target.value })}
                  >
                    <option value="">Keep current</option>
                    {ownershipTypes.map((o) => (
                      <option key={o.mfl_uid || o.id} value={o.mfl_uid || ""}>
                        {o.name}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Authority
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.authority?.name || selectedFacility.authority_name || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <select
                    className="form-select"
                    style={{ margin: 0 }}
                    value={form.authority}
                    onChange={(e) => setForm({ ...form, authority: e.target.value })}
                  >
                    <option value="">Keep current</option>
                    {authorityTypes.map((a) => (
                      <option key={a.mfl_uid || a.id} value={a.mfl_uid || ""}>
                        {a.name}
                      </option>
                    ))}
                  </select>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Status
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.status || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <select
                    className="form-select"
                    style={{ margin: 0 }}
                    value={form.status || ""}
                    onChange={(e) => setForm({ ...form, status: e.target.value })}
                  >
                    <option value="">Keep current</option>
                    <option value="Functional">Functional</option>
                    <option value="Non-Functional">Non-Functional</option>
                  </select>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Licensed
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.licensed ? "Yes" : "No"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <select
                    className="form-select"
                    style={{ margin: 0 }}
                    value={form.licensed ? "true" : "false"}
                    onChange={(e) => setForm({ ...form, licensed: e.target.value === "true" })}
                  >
                    <option value="">Keep current</option>
                    <option value="true">Yes</option>
                    <option value="false">No</option>
                  </select>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div className="form-card">
          <h3 className="form-section-title">Administrative Information</h3>
          <table
            className="table table-bordered"
            style={{ marginBottom: "0" }}
          >
            <thead style={{ backgroundColor: "#f8f9fa" }}>
              <tr>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "25%",
                  }}
                >
                  Field
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "35%",
                  }}
                >
                  Current Value
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "40%",
                  }}
                >
                  New Value
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Region
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.region?.name || 
                   (typeof selectedFacility.region === 'string' ? selectedFacility.region : null) ||
                   selectedFacility.region_name || 
                   "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <div className="input-group" style={{ margin: 0 }}>
                    <input
                      type="text"
                      className="form-control"
                      style={{ margin: 0 }}
                      value={selectedRegionName}
                      placeholder="Select region from hierarchy"
                      readOnly
                    />
                    <button
                      type="button"
                      className="btn btn-outline-secondary"
                      onClick={() => {
                        setHierarchySelectionType('region');
                        setShowHierarchyModal(true);
                      }}
                    >
                      Select
                    </button>
                  </div>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  District
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.district?.name || 
                   (typeof selectedFacility.district === 'string' ? selectedFacility.district : null) ||
                   selectedFacility.district_name || 
                   "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <div className="input-group" style={{ margin: 0 }}>
                    <input
                      type="text"
                      className="form-control"
                      style={{ margin: 0 }}
                      value={selectedDistrictName}
                      placeholder="Select district from hierarchy"
                      readOnly
                    />
                    <button
                      type="button"
                      className="btn btn-outline-secondary"
                      onClick={() => {
                        setHierarchySelectionType('district');
                        setShowHierarchyModal(true);
                      }}
                    >
                      Select
                    </button>
                  </div>
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Sub-County
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.subcounty?.name || 
                   (typeof selectedFacility.subcounty === 'string' ? selectedFacility.subcounty : null) ||
                   selectedFacility.subcounty_name || 
                   "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <div className="input-group" style={{ margin: 0 }}>
                    <input
                      type="text"
                      className="form-control"
                      style={{ margin: 0 }}
                      value={selectedSubcountyName}
                      placeholder="Select subcounty from hierarchy"
                      readOnly
                    />
                    <button
                      type="button"
                      className="btn btn-outline-secondary"
                      onClick={() => {
                        setHierarchySelectionType('subcounty');
                        setShowHierarchyModal(true);
                      }}
                    >
                      Select
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div className="form-card">
          <h3 className="form-section-title">Location & Contact Information</h3>
          <table
            className="table table-bordered"
            style={{ marginBottom: "0" }}
          >
            <thead style={{ backgroundColor: "#f8f9fa" }}>
              <tr>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "25%",
                  }}
                >
                  Field
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "35%",
                  }}
                >
                  Current Value
                </th>
                <th
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    width: "40%",
                  }}
                >
                  New Value
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Address
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.address || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <textarea
                    className="form-control"
                    placeholder="Enter new address"
                    style={{ margin: 0, minHeight: "60px" }}
                    value={form.address}
                    onChange={(e) => setForm({ ...form, address: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Contact Person Name
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.contact_personname || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter contact person name"
                    style={{ margin: 0 }}
                    value={form.contact_personname}
                    onChange={(e) => setForm({ ...form, contact_personname: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Contact Person Title
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.contact_persontitle || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Enter contact person title"
                    style={{ margin: 0 }}
                    value={form.contact_persontitle}
                    onChange={(e) => setForm({ ...form, contact_persontitle: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Contact Person Email
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.contact_personemail || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="email"
                    className="form-control"
                    placeholder="contact@facility.com"
                    style={{ margin: 0 }}
                    value={form.contact_personemail}
                    onChange={(e) => setForm({ ...form, contact_personemail: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Contact Person Mobile
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.contact_personmobile || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="tel"
                    className="form-control"
                    placeholder="+256 XXX XXX XXX"
                    style={{ margin: 0 }}
                    value={form.contact_personmobile}
                    onChange={(e) => setForm({ ...form, contact_personmobile: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Latitude
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.latitude || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="e.g., 0.3476"
                    style={{ margin: 0 }}
                    value={form.latitude}
                    onChange={(e) => setForm({ ...form, latitude: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Longitude
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.longitude || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="e.g., 32.5825"
                    style={{ margin: 0 }}
                    value={form.longitude}
                    onChange={(e) => setForm({ ...form, longitude: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Opening Date
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.opening_date || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="date"
                    className="form-control"
                    style={{ margin: 0 }}
                    value={form.opening_date}
                    onChange={(e) => setForm({ ...form, opening_date: e.target.value })}
                  />
                </td>
              </tr>
              <tr>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    fontWeight: 500,
                  }}
                >
                  Bed Capacity
                </td>
                <td
                  style={{
                    padding: "0.5rem",
                    fontSize: "0.85rem",
                    color: "#666",
                  }}
                >
                  {selectedFacility.bed_capacity || "N/A"}
                </td>
                <td style={{ padding: "0.5rem" }}>
                  <input
                    type="number"
                    className="form-control"
                    placeholder="Enter bed capacity"
                    style={{ margin: 0 }}
                    min="0"
                    value={form.bed_capacity}
                    onChange={(e) => setForm({ ...form, bed_capacity: e.target.value })}
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div className="form-card">
          <h3 className="form-section-title">Services Offered</h3>
          <div className="form-group">
            <label className="form-label">Current Services</label>
            <div style={{ padding: "0.5rem", fontSize: "0.85rem", color: "#666", marginBottom: "0.5rem" }}>
              {selectedFacility.services && Array.isArray(selectedFacility.services) && selectedFacility.services.length > 0
                ? selectedFacility.services.join(", ")
                : selectedFacility.services && typeof selectedFacility.services === "string"
                ? selectedFacility.services
                : "None"}
            </div>
            <label className="form-label">Update Services</label>
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

          <div className="form-card">
            <div className="form-group" style={{ marginTop: "0.75rem" }}>
              <label className="form-label">
                Supporting Documents<span className="required">*</span>
              </label>
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
              <div className="form-text">
                Upload supporting documents — PDF and images (JPG, PNG, GIF, WebP) only. Documents are required for update requests.
              </div>
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
                      <span style={{ fontSize: "0.8rem", color: "var(--text-muted)", minWidth: "200px" }}>
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

        <div className="action-buttons">
          <button type="submit" className="btn btn-primary" disabled={submitting || !selectedFacility}>
            <i className="bi bi-check-circle"></i> {submitting ? "Submitting..." : "Submit Update Request"}
          </button>
          <button
            type="button"
            className="btn btn-outline"
            onClick={() => {
              setSelectedFacility(null);
              setFacilitySearchQuery("");
              setAvailableFacilities([]);
              setShowSearchResults(false);
              setForm({
                facility_name: "",
                level: "",
                ownership: "",
                authority: "",
                status: "",
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
                admin_unit_id: "",
                subcounty_id: "",
                region_uid: "",
                district_uid: "",
              });
              setDocuments([]);
              setSubcounties([]);
              setSelectedRegionName("");
              setSelectedDistrictName("");
              setSelectedSubcountyName("");
              setError("");
              setSuccess("");
            }}
          >
            <i className="bi bi-arrow-counterclockwise"></i> Reset Form
          </button>
        </div>
          </>
        )}
      </form>

      {/* Hierarchy Selection Modal */}
      {showHierarchyModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={() => {
            setShowHierarchyModal(false);
            setHierarchySelectionType(null);
          }}
        >
          <div
            className="modal-dialog modal-lg modal-dialog-scrollable"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">
                  Select {hierarchySelectionType === 'region' ? 'Region' : hierarchySelectionType === 'district' ? 'District' : 'Sub-County'}
                </h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => {
                    setShowHierarchyModal(false);
                    setHierarchySelectionType(null);
                  }}
                />
              </div>
              <div className="modal-body">
                {rcTreeData.length > 0 ? (
                  <Tree
                    treeData={rcTreeData}
                    defaultExpandedKeys={defaultExpandedKeys}
                    showIcon
                    selectable
                    onSelect={(keys, info) => handleHierarchySelect(info.node)}
                  />
                ) : (
                  <div>Loading hierarchy...</div>
                )}
              </div>
              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => {
                    setShowHierarchyModal(false);
                    setHierarchySelectionType(null);
                  }}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </Fragment>
  );
}
