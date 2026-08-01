import React, { useEffect, useState, useMemo, useCallback } from "react";
import {
  FacilitiesApi,
  FacilityLevelsApi,
  OwnershipTypesApi,
  AuthorityTypesApi,
  UnitsApi,
} from "../../../helpers/api";
import { AVAILABLE_SERVICES } from "../../../helpers/services";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";

export default function FacilityManager() {
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [adminUnits, setAdminUnits] = useState([]);
  const [hierarchyTree, setHierarchyTree] = useState([]);
  const [showHierarchyModal, setShowHierarchyModal] = useState(false);
  const [form, setForm] = useState({
    name: "",
    short_name: "",
    historical_id: "",
    admin_unit_id: "",
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
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function loadLookups() {
    try {
      const [levels, ownerships, authorities, units, treeData] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
        UnitsApi.list(),
        UnitsApi.tree(),
      ]);
      setFacilityLevels(levels);
      setOwnershipTypes(ownerships);
      setAuthorityTypes(authorities);
      setAdminUnits(units);
      setHierarchyTree(treeData.tree || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load lookup data");
    }
  }

  const mapToRcNodes = useCallback((nodes) => {
    return nodes.map((n) => ({
      key: String(n.id),
      title: n.name,
      children: n.children ? mapToRcNodes(n.children) : [],
      raw: n,
    }));
  }, []);

  const rcTreeData = useMemo(() => mapToRcNodes(hierarchyTree), [hierarchyTree, mapToRcNodes]);

  // Expand top-level territories by default.
  const defaultExpandedKeys = useMemo(() => {
    return rcTreeData.map((node) => node.key);
  }, [rcTreeData]);

  function handleAdminUnitSelect(node) {
    if (node.raw && node.raw.id) {
      setForm({ ...form, admin_unit_id: String(node.raw.id) });
      setShowHierarchyModal(false);
    }
  }

  useEffect(() => {
    loadLookups();
  }, []);

  function resetForm() {
    setForm({
      name: "",
      short_name: "",
      historical_id: "",
      admin_unit_id: "",
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
    if (!form.admin_unit_id) {
      setError("Please select an admin unit");
      return;
    }
    if (!form.name || form.name.trim() === "") {
      setError("Please enter a field station name");
      return;
    }
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const payload = {
        ...form,
        admin_unit_id: form.admin_unit_id ? Number(form.admin_unit_id) : null,
        level: form.level || null,
        ownership: form.ownership || null,
        authority: form.authority || null,
        status: form.status || null,
        reporting: form.reporting || false,
        licensed: form.licensed || false,
        longitude: form.longitude ? Number(form.longitude) : null,
        latitude: form.latitude ? Number(form.latitude) : null,
        bed_capacity: form.bed_capacity ? Number(form.bed_capacity) : null,
        opening_date: form.opening_date || null,
        closing_date: form.closing_date || null,
        services: Array.isArray(form.services) ? form.services : [],
      };
      await FacilitiesApi.create(payload);
      resetForm();
      setSuccess("Field station record created successfully");
      setError("");
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save field station");
      setSuccess("");
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <div className="page-header">
        <h2>Direct Field Station Addition</h2>
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

      <form id="newFacilityForm" onSubmit={handleSubmit}>
        <div className="form-card">
          <h3 className="form-section-title">Basic Field Station Information</h3>
          <div className="form-row">
            <div className="form-group">
              <label className="form-label">Admin Unit<span className="required">*</span></label>
              <div style={{ display: "flex", gap: "0.5rem" }}>
                <input
                  type="text"
                  className="form-control"
                  readOnly
                  value={
                    form.admin_unit_id
                      ? adminUnits.find((u) => u.id === Number(form.admin_unit_id))?.name || ""
                      : ""
                  }
                  placeholder="Select admin unit from hierarchy"
                />
                <button
                  type="button"
                  className="btn btn-outline-primary"
                  onClick={() => setShowHierarchyModal(true)}
                >
                  Select
                </button>
              </div>
              <div className="form-text">Parent administrative unit for this field station</div>
            </div>
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
          </div>
          <div className="form-row">
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
              <label className="form-label">Historical ID</label>
              <input
                type="text"
                className="form-control"
                placeholder="Enter historical ID (optional)"
                value={form.historical_id}
                onChange={(e) => setForm({ ...form, historical_id: e.target.value })}
              />
            </div>
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

          <div className="form-group" style={{ marginTop: '0.75rem' }}>
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

        <div className="action-buttons">
          <button type="submit" className="btn btn-primary" disabled={saving}>
            <i className="bi bi-send"></i> {saving ? "Saving..." : "Create Field Station"}
          </button>
          <button type="button" className="btn btn-outline" onClick={resetForm}>
            <i className="bi bi-x-circle"></i> Clear Form
          </button>
        </div>
      </form>

      {/* Admin unit selector modal */}
      {showHierarchyModal && (
        <div
          className="modal fade show d-block"
          tabIndex="-1"
          style={{ background: "rgba(0,0,0,0.5)" }}
          onClick={() => setShowHierarchyModal(false)}
        >
          <div
            className="modal-dialog modal-lg modal-dialog-scrollable"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Select Admin Unit</h5>
                <button type="button" className="btn-close" onClick={() => setShowHierarchyModal(false)} />
              </div>
              <div className="modal-body">
                <Tree
                  treeData={rcTreeData}
                  defaultExpandedKeys={defaultExpandedKeys}
                  showIcon
                  selectable
                  onSelect={(keys, info) => handleAdminUnitSelect(info.node)}
                />
              </div>
              <div className="modal-footer">
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  onClick={() => setShowHierarchyModal(false)}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
