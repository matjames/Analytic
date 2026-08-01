import React, { Fragment, useEffect, useState } from "react";
import { useHistory, useParams } from "react-router-dom";
import { FacilitiesApi } from "../../../helpers/api";

export default function FacilityDetail() {
  const { id } = useParams();
  const history = useHistory();
  const [facility, setFacility] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    async function load() {
      setLoading(true);
      setError("");
      try {
        // Uses /facilities/:id endpoint where :id is the mfl_uid passed from FacilityList
        // Backend queries mfl_details view using WHERE mfl_uid = $1
        const data = await FacilitiesApi.get(id);
        setFacility(data);
      } catch (e) {
        setError(
          e?.response?.data?.error || e.message || "Failed to load facility"
        );
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  const formatIdentifier = (identifier) => {
    if (!identifier || typeof identifier !== "string") return identifier || "—";
    // Remove first 6 characters if identifier is longer than 6 characters
    return identifier.length > 6 ? identifier.slice(6) : identifier;
  };

  const renderServices = () => {
    if (!facility) return null;
    const services = facility.services;
    if (Array.isArray(services) && services.length > 0) {
      return services.map((svc) => (
        <div key={svc} className="service-item">
          <i className="bi bi-check-circle-fill"></i>
          <span>{svc}</span>
        </div>
      ));
    }
    if (typeof services === "string" && services.trim()) {
      return (
        <div className="service-item">
          <i className="bi bi-check-circle-fill"></i>
          <span>{services}</span>
        </div>
      );
    }
    return <div className="text-muted small">No services listed.</div>;
  };

  if (loading) {
    return (
      <Fragment>
        <div className="page-header">
          <div className="page-header-left">
            <h2>Loading field station...</h2>
          </div>
        </div>
      </Fragment>
    );
  }

  if (error) {
    return (
      <Fragment>
        <div className="page-header">
          <div className="page-header-left">
            <h2>Field Station Details</h2>
          </div>
        </div>
        <div className="alert alert-danger" role="alert">
          {error}
        </div>
      </Fragment>
    );
  }

  if (!facility) {
    return null;
  }

  return (
    <Fragment>
      <div className="page-header">
        <div className="page-header-left">
          <h2>{facility.name || facility.identifier || "Field Station Details"}</h2>
          <div className="subtitle">
            View field station information, operational profile, and current status
          </div>
        </div>
        <div className="page-header-actions">
          <button
            className="btn btn-outline-secondary"
            type="button"
            onClick={() => history.push("/facilities")}
          >
            <i className="bi bi-arrow-left"></i> Back
          </button>
        </div>
      </div>

      <div className="details-grid">
        <div className="details-section">
          <div className="section-header">
            <i className="bi bi-info-circle"></i>
            <h3>Basic Information</h3>
          </div>
          <div className="detail-row">
            <span className="detail-label">Field Station Name</span>
            <span className="detail-value">
              {facility.name || facility.identifier || "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Identifier</span>
            <span className="detail-value">{formatIdentifier(facility.identifier)}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Historical ID</span>
            <span className="detail-value">
              {facility.historical_id || "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Station Tier</span>
            <span className="detail-value">
              {facility.level?.name || facility.level_name || (typeof facility.level === 'string' ? facility.level : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Ownership Type</span>
            <span className="detail-value">
              {facility.ownership?.name || facility.ownership_name || (typeof facility.ownership === 'string' ? facility.ownership : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Authority Type</span>
            <span className="detail-value">
              {facility.authority?.name || facility.authority_name || (typeof facility.authority === 'string' ? facility.authority : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Opening Date</span>
            <span className="detail-value">
              {facility.opening_date || "—"}
            </span>
          </div>
        </div>

        <div className="details-section">
          <div className="section-header">
            <i className="bi bi-geo-alt"></i>
            <h3>Location Details</h3>
          </div>
          <div className="detail-row">
            <span className="detail-label">Region</span>
            <span className="detail-value">
              {facility.region?.name || (typeof facility.region === 'string' ? facility.region : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">District</span>
            <span className="detail-value">
              {facility.district?.name || (typeof facility.district === 'string' ? facility.district : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Sub-County</span>
            <span className="detail-value">
              {facility.subcounty?.name || (typeof facility.subcounty === 'string' ? facility.subcounty : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Parish</span>
            <span className="detail-value">
              {facility.parish?.name || (typeof facility.parish === 'string' ? facility.parish : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Village</span>
            <span className="detail-value">
              {facility.village?.name || (typeof facility.village === 'string' ? facility.village : "—")}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Physical Address</span>
            <span className="detail-value">
              {facility.address || "—"}
            </span>
          </div>
        </div>

        <div className="details-section">
          <div className="section-header">
            <i className="bi bi-telephone"></i>
            <h3>Contact Information</h3>
          </div>
          <div className="detail-row">
            <span className="detail-label">Contact Person</span>
            <span className="detail-value">
              {facility.contact_personname || "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Title</span>
            <span className="detail-value">
              {facility.contact_persontitle || "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Phone Number</span>
            <span className="detail-value">
              {facility.contact_personmobile || "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Email Address</span>
            <span className="detail-value">
              {facility.contact_personemail || "—"}
            </span>
          </div>
        </div>

        <div className="details-section">
          <div className="section-header">
            <i className="bi bi-building"></i>
            <h3>Field Operations Profile</h3>
          </div>
          <div className="detail-row">
            <span className="detail-label">Agent Capacity</span>
            <span className="detail-value">
              {facility.bed_capacity ?? "—"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Reporting</span>
            <span className="detail-value">
              {facility.reporting ? "Yes" : "No"}
            </span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Operationally Verified</span>
            <span className="detail-value">
              {facility.licensed ? "Yes" : "No"}
            </span>
          </div>
        </div>
      </div>

      <div className="services-section">
        <div className="section-header">
          <i className="bi bi-heart-pulse"></i>
          <h3>Available Services</h3>
        </div>
        <div className="services-grid">{renderServices()}</div>
      </div>

      <div className="details-section" style={{ marginBottom: "1rem" }}>
        <div className="section-header">
          <i className="bi bi-pin-map"></i>
          <h3>Geographic Coordinates</h3>
        </div>
        <div className="detail-row">
          <span className="detail-label">Latitude</span>
          <span className="detail-value">{facility.latitude ?? "—"}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Longitude</span>
          <span className="detail-value">{facility.longitude ?? "—"}</span>
        </div>
      </div>
    </Fragment>
  );
}
