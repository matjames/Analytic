import React, { useEffect, useMemo, useState } from "react";
import { useHistory, useParams } from "react-router-dom";
import Tree from "rc-tree";
import "rc-tree/assets/index.css";
import {
  AuthorityTypesApi,
  FacilitiesApi,
  FacilityLevelsApi,
  OwnershipTypesApi,
  UnitsApi,
} from "../../../helpers/api";
import { AVAILABLE_SERVICES } from "../../../helpers/services";

export default function FacilityFormPage() {
  const history = useHistory();
  const { id } = useParams();
  const isEdit = Boolean(id);

  const [form, setForm] = useState({
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

  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [adminUnits, setAdminUnits] = useState([]);
  const [hierarchyTree, setHierarchyTree] = useState([]);
  const [showHierarchyModal, setShowHierarchyModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const serviceOptions = useMemo(
    () =>
      AVAILABLE_SERVICES.map((s) =>
        typeof s === "string" ? { code: s, name: s } : s
      ),
    []
  );

  function mapToRcNodes(nodes) {
    return nodes.map((n) => ({
      key: String(n.id),
      title: n.name,
      children: n.children ? mapToRcNodes(n.children) : [],
      raw: n,
    }));
  }

  const rcTreeData = useMemo(() => mapToRcNodes(hierarchyTree || []), [hierarchyTree]);

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

  async function loadFacility() {
    if (!isEdit) return;
    setLoading(true);
    setError("");
    try {
      const data = await FacilitiesApi.get(id, {
        include: "admin_unit,facility_level,ownership,authority",
      });
      setForm({
        short_name: data.short_name || "",
        historical_id: data.historical_id || "",
        admin_unit_id: data.admin_unit?.id || data.admin_unit_id || "",
        level: data.level?.mfl_uid || data.level_mfl_uid || data.facility_level?.mfl_uid || "",
        ownership: data.ownership?.mfl_uid || data.ownership_mfl_uid || "",
        authority: data.authority?.mfl_uid || data.authority_mfl_uid || "",
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
      setError(e?.response?.data?.error || e.message || "Failed to load facility");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadLookups();
  }, []);

  useEffect(() => {
    loadFacility();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  function toggleService(code) {
    const exists = form.services.includes(code);
    setForm({
      ...form,
      services: exists
        ? form.services.filter((s) => s !== code)
        : [...form.services, code],
    });
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.admin_unit_id) {
      setError("Please select an admin unit");
      return;
    }
    setSaving(true);
    setError("");
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
      if (isEdit) {
        await FacilitiesApi.update(id, payload);
      } else {
        await FacilitiesApi.create(payload);
      }
      history.push("/facilities");
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to save facility");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="card shadow-sm">
      <div className="card-header d-flex justify-content-between align-items-center">
        <div>
          <div className="fw-semibold">{isEdit ? "Edit Field Station" : "Add Field Station"}</div>
          <div className="text-muted small">
            Provide field station details; the registry UID is derived from the selected administrative unit.
          </div>
        </div>
        <button
          type="button"
          className="btn btn-outline-secondary btn-sm"
          onClick={() => history.push("/facilities")}
        >
          Back to list
        </button>
      </div>

      <div className="card-body">
        {error && <div className="alert alert-danger py-1 small mb-3">{error}</div>}
        {loading ? (
          <div className="text-muted">Loading field station...</div>
        ) : (
          <form className="p-3 bg-light rounded" onSubmit={handleSubmit}>
            <h6 className="mb-3 fw-semibold">Basic Information</h6>
            <div className="row g-2 mb-3">
              <div className="col-md-4">
                <label className="form-label form-label-sm">
                  Admin Unit <span className="text-danger">*</span>
                </label>
                <div className="input-group input-group-sm">
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
                <small className="text-muted">
                  Field station name is derived from the selected administrative unit.
                </small>
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Short Name</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Optional"
                  value={form.short_name}
                  onChange={(e) => setForm({ ...form, short_name: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Historical ID</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Optional"
                  value={form.historical_id}
                  onChange={(e) => setForm({ ...form, historical_id: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Station Tier</label>
                <select
                  className="form-select form-select-sm"
                  value={form.level}
                  onChange={(e) => setForm({ ...form, level: e.target.value })}
                >
                  <option value="">Select...</option>
                  {facilityLevels.map((l) => (
                    <option key={l.mfl_uid || l.id} value={l.mfl_uid || ""}>
                      {l.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Status</label>
                <select
                  className="form-select form-select-sm"
                  value={form.status}
                  onChange={(e) => setForm({ ...form, status: e.target.value })}
                >
                  <option value="">Select...</option>
                  <option value="Functional">Functional</option>
                  <option value="Non-Functional">Non-Functional</option>
                </select>
              </div>
            </div>

            <div className="row g-2 mb-3">
              <div className="col-md-3">
                <label className="form-label form-label-sm">Ownership</label>
                <select
                  className="form-select form-select-sm"
                value={form.ownership}
                onChange={(e) => setForm({ ...form, ownership: e.target.value })}
                >
                  <option value="">Select...</option>
                  {ownershipTypes.map((o) => (
                  <option key={o.mfl_uid || o.id} value={o.mfl_uid || ""}>
                      {o.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="col-md-3">
                <label className="form-label form-label-sm">Authority</label>
                <select
                  className="form-select form-select-sm"
                value={form.authority}
                onChange={(e) => setForm({ ...form, authority: e.target.value })}
                >
                  <option value="">Select...</option>
                  {authorityTypes.map((a) => (
                  <option key={a.mfl_uid || a.id} value={a.mfl_uid || ""}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">
                  <input
                    type="checkbox"
                    className="form-check-input me-1"
                    checked={form.reporting}
                    onChange={(e) => setForm({ ...form, reporting: e.target.checked })}
                  />
                  Reporting
                </label>
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">
                  <input
                    type="checkbox"
                    className="form-check-input me-1"
                    checked={form.licensed}
                    onChange={(e) => setForm({ ...form, licensed: e.target.checked })}
                  />
                  Operationally Verified
                </label>
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Agent Capacity</label>
                <input
                  type="number"
                  className="form-control form-control-sm"
                  placeholder="0"
                  value={form.bed_capacity}
                  onChange={(e) => setForm({ ...form, bed_capacity: e.target.value })}
                />
              </div>
            </div>

            <h6 className="mb-3 fw-semibold mt-4">Location & Contact</h6>
            <div className="row g-2 mb-3">
              <div className="col-md-8">
                <label className="form-label form-label-sm">Address</label>
                <textarea
                  className="form-control form-control-sm"
                  placeholder="Field station address"
                  rows="2"
                  value={form.address}
                  onChange={(e) => setForm({ ...form, address: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Latitude</label>
                <input
                  type="number"
                  step="any"
                  className="form-control form-control-sm"
                  placeholder="0.0000000"
                  value={form.latitude}
                  onChange={(e) => setForm({ ...form, latitude: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Longitude</label>
                <input
                  type="number"
                  step="any"
                  className="form-control form-control-sm"
                  placeholder="0.0000000"
                  value={form.longitude}
                  onChange={(e) => setForm({ ...form, longitude: e.target.value })}
                />
              </div>
            </div>

            <div className="row g-2 mb-3">
              <div className="col-md-3">
                <label className="form-label form-label-sm">Contact Person Name</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Name"
                  value={form.contact_personname}
                  onChange={(e) => setForm({ ...form, contact_personname: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Title</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Title"
                  value={form.contact_persontitle}
                  onChange={(e) => setForm({ ...form, contact_persontitle: e.target.value })}
                />
              </div>
              <div className="col-md-2">
                <label className="form-label form-label-sm">Mobile</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Mobile"
                  value={form.contact_personmobile}
                  onChange={(e) => setForm({ ...form, contact_personmobile: e.target.value })}
                />
              </div>
              <div className="col-md-3">
                <label className="form-label form-label-sm">Email</label>
                <input
                  className="form-control form-control-sm"
                  placeholder="Email"
                  value={form.contact_personemail}
                  onChange={(e) => setForm({ ...form, contact_personemail: e.target.value })}
                />
              </div>
            </div>

            <h6 className="mb-3 fw-semibold mt-4">Dates</h6>
            <div className="row g-2 mb-3">
              <div className="col-md-3">
                <label className="form-label form-label-sm">Opening Date</label>
                <input
                  type="date"
                  className="form-control form-control-sm"
                  value={form.opening_date}
                  onChange={(e) => setForm({ ...form, opening_date: e.target.value })}
                />
              </div>
              <div className="col-md-3">
                <label className="form-label form-label-sm">Closing Date</label>
                <input
                  type="date"
                  className="form-control form-control-sm"
                  value={form.closing_date}
                  onChange={(e) => setForm({ ...form, closing_date: e.target.value })}
                />
              </div>
            </div>

            <h6 className="mb-3 fw-semibold mt-4">Services</h6>
            <div className="row g-2 mb-3">
              {serviceOptions.map((svc) => (
                <div className="col-md-3" key={svc.code}>
                  <label className="form-label form-label-sm">
                    <input
                      type="checkbox"
                      className="form-check-input me-1"
                      checked={form.services.includes(svc.code)}
                      onChange={() => toggleService(svc.code)}
                    />
                    {svc.name}
                  </label>
                </div>
              ))}
            </div>

            <div className="d-flex gap-2">
              <button type="submit" className="btn btn-primary btn-sm" disabled={saving}>
                {saving ? "Saving..." : isEdit ? "Update Field Station" : "Create Field Station"}
              </button>
              <button
                type="button"
                className="btn btn-outline-secondary btn-sm"
                onClick={() => history.push("/facilities")}
              >
                Cancel
              </button>
            </div>
          </form>
        )}
      </div>

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
    </div>
  );
}
