import React, { Fragment, useEffect, useMemo, useState } from "react";
import {
  AuthorityTypesApi,
  FacilityLevelsApi,
  OwnershipTypesApi,
  RequestsApi,
} from "../../helpers/api";
import { AVAILABLE_SERVICES } from "../../helpers/services";

export default function RequestCreate() {
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
    operating_hours: "",
  });
  const [documents, setDocuments] = useState([]);
  const [documentType, setDocumentType] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [district, setDistrict] = useState(null);
  const [subcounties, setSubcounties] = useState([]);
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [loadingLookups, setLoadingLookups] = useState(false);

  useEffect(() => {
    loadLookups();
    loadDistrictInfo();
  }, []);

  async function loadLookups() {
    setLoadingLookups(true);
    setError("");
    try {
      const [levels, ownerships, authorities] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
      ]);
      setFacilityLevels(levels || []);
      setOwnershipTypes(ownerships || []);
      setAuthorityTypes(authorities || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load lookup data");
    } finally {
      setLoadingLookups(false);
    }
  }

  async function loadDistrictInfo() {
    try {
      const info = await RequestsApi.getDistrictInfo();
      setDistrict(info.district || null);
      setSubcounties(info.subcounties || []);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load district information");
    }
  }

  const serviceColumns = useMemo(() => {
    const chunk = [];
    const perCol = Math.ceil(AVAILABLE_SERVICES.length / 3);
    for (let i = 0; i < AVAILABLE_SERVICES.length; i += perCol) {
      chunk.push(AVAILABLE_SERVICES.slice(i, i + perCol));
    }
    return chunk;
  }, []);

  function handleChange(e) {
    const { name, value, type, checked } = e.target;
    setForm((prev) => ({
      ...prev,
      [name]: type === "checkbox" ? checked : value,
    }));
  }

  function handleServiceToggle(service) {
    setForm((prev) => {
      const current = new Set(prev.services || []);
      if (current.has(service)) {
        current.delete(service);
      } else {
        current.add(service);
      }
      return { ...prev, services: Array.from(current) };
    });
  }

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

  function handleDocumentTypeChange(index, value) {
    setDocuments((prev) =>
      prev.map((doc, i) => (i === index ? { ...doc, type: value } : doc))
    );
  }

  function removeDocument(index) {
    setDocuments((prev) => prev.filter((_, i) => i !== index));
  }

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    setSuccess("");

    if (!form.facility_name.trim()) {
      setError("Facility name is required");
      setSubmitting(false);
      return;
    }
    if (!form.subcounty_id) {
      setError("Subcounty is required");
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
      setError("Supporting documents are required");
      setSubmitting(false);
      return;
    }

    const facilityData = {
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
      operating_hours: form.operating_hours || null,
    };

    try {
      await RequestsApi.create("new_addition", facilityData, documents, null);
      setSuccess("Request submitted successfully.");
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
        operating_hours: "",
      });
      setDocuments([]);
    } catch (err) {
      setError(err?.response?.data?.error || err.message || "Failed to submit request");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Fragment>
        <div class="page-header">
          <h2>New Field Station Registration</h2>
        </div>

        <div class="alert alert-info">
          <i class="bi bi-info-circle"></i> 
          Register a field survey station for agent deployment, enumeration, and operational sampling. All requests are subject to approval.
        </div>

        {error && (
          <div className="alert alert-danger py-1 small mb-3">{error}</div>
        )}
        {success && (
          <div className="alert alert-success py-1 small mb-3">{success}</div>
        )}

        <form id="newFacilityForm" onSubmit={handleSubmit}>
          <div class="form-card">
            <h3 class="form-section-title">Field Station Profile</h3>
            <div
              class="form-row"
              style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: "1rem" }}
            >
              <div class="form-group">
                <label class="form-label">Field Station Name<span class="required">*</span></label>
                <input
                  type="text"
                  class="form-control"
                  name="facility_name"
                  value={form.facility_name}
                  onChange={handleChange}
                  placeholder="Enter field station name"
                  required
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Station Tier<span class="required">*</span></label>
                <select
                  class="form-select"
                  name="level"
                  value={form.level}
                  onChange={handleChange}
                  required
                  disabled={submitting || loadingLookups}
                >
                  <option value="">
                    {loadingLookups ? "Loading station tiers..." : "Select station tier"}
                  </option>
                  {facilityLevels.map((l) => (
                    <option key={l.mfl_uid || l.id} value={l.mfl_uid || l.id}>
                      {l.name}
                    </option>
                  ))}
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">Operating Model<span class="required">*</span></label>
                <select
                  class="form-select"
                  name="ownership"
                  value={form.ownership}
                  onChange={handleChange}
                  required
                  disabled={submitting || loadingLookups}
                >
                  <option value="">
                    {loadingLookups ? "Loading operating models..." : "Select operating model"}
                  </option>
                  {ownershipTypes.map((o) => (
                    <option key={o.mfl_uid || o.id} value={o.mfl_uid || o.id}>
                      {o.name}
                    </option>
                  ))}
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">Managing Organisation<span class="required">*</span></label>
                <select
                  class="form-select"
                  name="authority"
                  value={form.authority}
                  onChange={handleChange}
                  required
                  disabled={submitting || loadingLookups}
                >
                  <option value="">
                    {loadingLookups ? "Loading organisations..." : "Select organisation"}
                  </option>
                  {authorityTypes.map((a) => (
                    <option key={a.mfl_uid || a.id} value={a.mfl_uid || a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">Date Opened</label>
                <input
                  type="date"
                  class="form-control"
                  name="opening_date"
                  value={form.opening_date}
                  onChange={handleChange}
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Approved Operational Site</label>
                <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
                  <input
                    type="checkbox"
                    name="licensed"
                    checked={form.licensed}
                    onChange={handleChange}
                    disabled={submitting}
                  />
                  <span class="form-text">Tick if the station is already approved for field operations</span>
                </div>
              </div>
            </div>
          </div>

          <div class="form-card">
            <h3 class="form-section-title">Administrative Location</h3>
            <div class="form-row" style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: "1rem" }}>
              <div class="form-group">
                <label class="form-label">District</label>
                <input
                  type="text"
                  class="form-control"
                  value={district?.name || ""}
                  disabled
                  placeholder="District will be loaded based on your profile"
                />
              </div>
              <div class="form-group">
                <label class="form-label">Sub-County<span class="required">*</span></label>
                <select
                  class="form-select"
                  name="subcounty_id"
                  value={form.subcounty_id}
                  onChange={handleChange}
                  required
                  disabled={submitting || !subcounties.length}
                >
                  <option value="">
                    {!subcounties.length
                      ? "No sub-counties found for your district"
                      : "Select sub-county"}
                  </option>
                  {subcounties.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          <div class="form-card">
            <h3 class="form-section-title">Contact Details</h3>
            <div
              class="form-row"
              style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: "1rem" }}
            >
              <div class="form-group">
                <label class="form-label">Primary Phone<span class="required">*</span></label>
                <input
                  type="tel"
                  class="form-control"
                  name="contact_personmobile"
                  value={form.contact_personmobile}
                  onChange={handleChange}
                  placeholder="+256 XXX XXX XXX"
                  required
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Secondary Phone</label>
                <input
                  type="tel"
                  class="form-control"
                  name="secondary_phone"
                  value={form.secondary_phone || ""}
                  onChange={handleChange}
                  placeholder="+256 XXX XXX XXX"
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Email Address<span class="required">*</span></label>
                <input
                  type="email"
                  class="form-control"
                  name="contact_personemail"
                  value={form.contact_personemail}
                  onChange={handleChange}
                  placeholder="facility@email.com"
                  required
                  disabled={submitting}
                />
              </div>
              <div class="form-group" style={{ gridColumn: "1 / span 3" }}>
                <label class="form-label">Physical Address<span class="required">*</span></label>
                <textarea
                  class="form-control"
                  name="address"
                  value={form.address}
                  onChange={handleChange}
                  placeholder="Enter complete physical address"
                  required
                  disabled={submitting}
                ></textarea>
              </div>
            </div>
          </div>

          <div class="form-card">
            <h3 class="form-section-title">Geographic Location</h3>
            <div class="form-row">
              <div class="form-group">
                <label class="form-label">Latitude</label>
                <input
                  type="text"
                  class="form-control"
                  name="latitude"
                  value={form.latitude}
                  onChange={handleChange}
                  placeholder="e.g., 0.3476 (optional)"
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Longitude</label>
                <input
                  type="text"
                  class="form-control"
                  name="longitude"
                  value={form.longitude}
                  onChange={handleChange}
                  placeholder="e.g., 32.5825 (optional)"
                  disabled={submitting}
                />
              </div>
            </div>
          </div>

          <div class="form-card">
            <h3 class="form-section-title">Operational Capacity</h3>
            <div class="form-row" style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: "1rem" }}>
              <div class="form-group">
                <label class="form-label">Deployment Capacity</label>
                <input
                  type="number"
                  class="form-control"
                  name="bed_capacity"
                  value={form.bed_capacity}
                  onChange={handleChange}
                  placeholder="Enter planned field-team capacity"
                  min="0"
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Number of Staff</label>
                <input
                  type="number"
                  class="form-control"
                  name="staff_count"
                  value={form.staff_count || ""}
                  onChange={handleChange}
                  placeholder="Enter total staff count"
                  min="0"
                  disabled={submitting}
                />
              </div>
              <div class="form-group">
                <label class="form-label">Operating Hours</label>
                <input
                  type="text"
                  class="form-control"
                  name="operating_hours"
                  value={form.operating_hours}
                  onChange={handleChange}
                  placeholder="e.g., Monday–Friday: 8:00 AM – 5:00 PM"
                  disabled={submitting}
                />
              </div>
            </div>
            <div class="form-group" style={{ marginTop: "0.75rem" }}>
              <label class="form-label">Services Offered</label>
              <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: "0.5rem", marginTop: "0.5rem" }}>
                {serviceColumns.map((col, idx) => (
                  <div key={idx}>
                    {col.map((service) => (
                      <label
                        key={service}
                        style={{ display: "flex", alignItems: "center", fontSize: "0.85rem", cursor: "pointer", marginBottom: "0.25rem" }}
                      >
                        <input
                          type="checkbox"
                          style={{ marginRight: "0.5rem" }}
                          checked={form.services.includes(service)}
                          onChange={() => handleServiceToggle(service)}
                          disabled={submitting}
                        />
                        {service}
                      </label>
                    ))}
                  </div>
                ))}
              </div>
            </div>

          </div>

          <div class="form-card">
            <h3 class="form-section-title">Request Information</h3>
            <div class="form-group">
              <label class="form-label">Supporting Documents<span class="required">*</span></label>
              <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <select
                  class="form-select"
                  style={{ maxWidth: "260px" }}
                  value={documentType}
                  onChange={(e) => setDocumentType(e.target.value)}
                  disabled={submitting}
                >
                  <option value="">Select document type</option>
                  <option value="District CAO Letter">District CAO Letter</option>
                  <option value="Operating License">Operating License</option>
                  <option value="District Council Minutes">District Council Minutes</option>
                </select>
                <input
                  type="file"
                  class="form-control"
                  multiple
                  accept=".pdf,.jpg,.jpeg,.png,.gif,.webp,application/pdf,image/jpeg,image/png,image/gif,image/webp"
                  onChange={handleDocumentChange}
                  disabled={submitting}
                />
              </div>
              {documents.length > 0 && (
                <div
                  class="mt-2"
                  style={{
                    border: "1px solid var(--border-light)",
                    borderRadius: "0.25rem",
                    backgroundColor: "var(--bg-subtle)",
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      padding: "0.35rem 0.75rem",
                      borderBottom: "1px solid var(--border-light)",
                      fontSize: "0.8rem",
                      fontWeight: 500,
                      color: "var(--text-muted)",
                    }}
                  >
                    <span>File name</span>
                    <span>Type</span>
                    <span>Action</span>
                  </div>
                  <div>
                    {documents.map((doc, idx) => (
                      <div
                        key={idx}
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          alignItems: "center",
                          padding: "0.4rem 0.75rem",
                          borderTop: idx === 0 ? "none" : "1px solid var(--border-light)",
                          fontSize: "0.85rem",
                        }}
                      >
                        <div style={{ display: "flex", alignItems: "center", gap: "0.4rem", minWidth: 0 }}>
                          <i class="bi bi-file-earmark-text" style={{ fontSize: "0.95rem", color: "var(--text-muted)" }}></i>
                          <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                            {doc.file?.name || ""}
                          </span>
                        </div>
                        <div style={{ fontSize: "0.8rem", color: "var(--text-muted)" }}>
                          {doc.type}
                        </div>
                        <button
                          type="button"
                          class="btn btn-sm btn-outline-danger"
                          onClick={() => removeDocument(idx)}
                          disabled={submitting}
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              <div class="form-text">Upload registration documents, licenses, and certificates — PDF and images (JPG, PNG, GIF, WebP) only (Required)</div>
            </div>
          </div>

          <div class="action-buttons">
            <button type="submit" class="btn btn-primary" disabled={submitting}>
              <i class="bi bi-send"></i> {submitting ? "Submitting..." : "Submit Request"}
            </button>
            <button type="button" class="btn btn-secondary">
              <i class="bi bi-floppy"></i> Save as Draft
            </button>
            <button type="reset" class="btn btn-outline">
              <i class="bi bi-x-circle"></i> Clear Form
            </button>
          </div>
        </form>
    </Fragment>
  );
}
