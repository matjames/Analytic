import React, { useEffect, useState, Fragment } from "react";
import { useHistory } from "react-router-dom";
import {
  FacilitiesApi,
  FacilityLevelsApi,
  OwnershipTypesApi,
  AuthorityTypesApi,
  UnitsApi,
  LevelsApi,
} from "../../helpers/api";

const formatValue = (value) => {
  if (value === null || value === undefined) return "—";
  if (typeof value === "string" || typeof value === "number") return value;
  if (typeof value === "object") return value.name || value.title || value.label || value.mfl_uid || "—";
  return "—";
};

const formatIdentifier = (identifier) => {
  if (!identifier || typeof identifier !== "string") return identifier || "—";
  // Remove first 6 characters (800802) if identifier is longer than 6 characters
  return identifier.length > 6 ? identifier.slice(6) : identifier;
};

const getFacilityDisplayName = (facility) => {
  if (!facility) return "—";
  // Display the name field from mfl_details view
  return facility.name || facility.identifier || "—";
};

export default function FacilityList() {
  const history = useHistory();
  const [facilities, setFacilities] = useState([]);
  const [facilityLevels, setFacilityLevels] = useState([]);
  const [ownershipTypes, setOwnershipTypes] = useState([]);
  const [authorityTypes, setAuthorityTypes] = useState([]);
  const [regions, setRegions] = useState([]);
  const [districts, setDistricts] = useState([]);
  const [subcounties, setSubcounties] = useState([]);
  const [treeData, setTreeData] = useState([]);
  const [levelsMap, setLevelsMap] = useState(new Map());
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [filterLevel, setFilterLevel] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [filterOwnership, setFilterOwnership] = useState("");
  const [filterAuthority, setFilterAuthority] = useState("");
  const [filterRegion, setFilterRegion] = useState("");
  const [filterDistrict, setFilterDistrict] = useState("");
  const [filterSubcounty, setFilterSubcounty] = useState("");

  useEffect(() => {
    loadLookups();
  }, []);

  useEffect(() => {
    // Initial load and whenever filters change: always go back to page 1
    loadFacilities(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    filterLevel,
    searchQuery,
    filterOwnership,
    filterAuthority,
    filterRegion,
    filterDistrict,
    filterSubcounty,
  ]);

  // Load districts when region changes
  useEffect(() => {
    if (!treeData.length || !levelsMap.size) return;
    
    if (filterRegion) {
      loadDistrictsForRegion(filterRegion);
      // Clear district and subcounty when region changes
      setFilterDistrict("");
      setFilterSubcounty("");
    } else {
      // If no region selected, load all districts from API
      UnitsApi.list().then(adminUnits => {
        const districtsList = (adminUnits || []).filter(unit => unit.admin_level?.level_number === 3);
        setDistricts(districtsList);
      }).catch(() => {
        // Ignore errors, keep current districts
      });
      setFilterDistrict("");
      setFilterSubcounty("");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterRegion, treeData, levelsMap]);

  // Load subcounties when district changes
  useEffect(() => {
    if (!treeData.length || !levelsMap.size) return;
    
    if (filterDistrict) {
      loadSubcountiesForDistrict(filterDistrict);
      // Clear subcounty when district changes
      setFilterSubcounty("");
    } else {
      // If no district selected, clear subcounties
      setSubcounties([]);
      setFilterSubcounty("");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterDistrict, treeData, levelsMap]);

  async function loadLookups() {
    try {
      const [levels, ownerships, authorities, adminUnits, levelsData, treeRes] = await Promise.all([
        FacilityLevelsApi.list(),
        OwnershipTypesApi.list(),
        AuthorityTypesApi.list(),
        UnitsApi.list(),
        LevelsApi.list().catch(() => []), // Fallback to empty array if fails
        UnitsApi.tree().catch(() => ({ tree: [] })), // Fallback if fails
      ]);
      
      setFacilityLevels(levels || []);
      setOwnershipTypes(ownerships || []);
      setAuthorityTypes(authorities || []);
      
      // Set levels map for hierarchy filtering
      const map = new Map((levelsData || []).map(l => [l.id, l]));
      setLevelsMap(map);
      setTreeData(treeRes?.tree || []);
      
      // Filter regions (level_number = 2)
      const regionsList = (adminUnits || []).filter(unit => unit.admin_level?.level_number === 2);
      setRegions(regionsList);
      
      // Initially load all districts, will be filtered by region selection
      const districtsList = (adminUnits || []).filter(unit => unit.admin_level?.level_number === 3);
      setDistricts(districtsList);
    } catch (e) {
      console.error("Error loading lookups:", e);
      setError(e?.response?.data?.error || e.message || "Failed to load lookup data");
    }
  }

  const loadDistrictsForRegion = (regionName) => {
    // Find the region in the tree
    const findRegionInTree = (nodes) => {
      for (const node of nodes) {
        const level = levelsMap.get(node.levelId);
        if (level && level.level_number === 2 && node.name === regionName) {
          return node;
        }
        if (node.children && node.children.length > 0) {
          const found = findRegionInTree(node.children);
          if (found) return found;
        }
      }
      return null;
    };
    
    const regionNode = findRegionInTree(treeData);
    if (regionNode && regionNode.children) {
      // Get districts (level 3) from region's children
      const districtsList = regionNode.children
        .filter(node => {
          const level = levelsMap.get(node.levelId);
          return level && level.level_number === 3;
        })
        .map(node => ({
          id: node.id,
          name: node.name,
          mfl_uid: node.mfl_uid,
          admin_level: { level_number: 3 }
        }));
      setDistricts(districtsList);
    } else {
      // If not found in tree, try to load from API as fallback
      setDistricts([]);
    }
  };

  const loadSubcountiesForDistrict = (districtName) => {
    // Find the district in the tree
    const findDistrictInTree = (nodes) => {
      for (const node of nodes) {
        const level = levelsMap.get(node.levelId);
        if (level && level.level_number === 3 && node.name === districtName) {
          return node;
        }
        if (node.children && node.children.length > 0) {
          const found = findDistrictInTree(node.children);
          if (found) return found;
        }
      }
      return null;
    };
    
    const districtNode = findDistrictInTree(treeData);
    if (districtNode && districtNode.children) {
      // Get subcounties (level 4) from district's children
      const subcountiesList = districtNode.children
        .filter(node => {
          const level = levelsMap.get(node.levelId);
          return level && level.level_number === 4;
        })
        .map(node => ({
          id: node.id,
          name: node.name,
          mfl_uid: node.mfl_uid,
        }));
      setSubcounties(subcountiesList);
    } else {
      // If not found in tree, clear subcounties
      setSubcounties([]);
    }
  };

  const buildQueryParams = (pageOverride) => {
    const nextPage = pageOverride ?? page;
    const params = {
      page: nextPage,
      pageSize,
      ...(filterLevel && { level: filterLevel }),
      ...(searchQuery && { q: searchQuery }),
      ...(filterOwnership && { ownership: filterOwnership }),
      ...(filterAuthority && { authority: filterAuthority }),
      ...(filterRegion && { region: filterRegion }),
      ...(filterDistrict && { district: filterDistrict }),
      ...(filterSubcounty && { subcounty: filterSubcounty }),
    };
    return params;
  };

  async function loadFacilities(nextPage = 1) {
    setLoading(true);
    setError("");
    try {
      const res = await FacilitiesApi.listPaged({
        ...buildQueryParams(nextPage),
      });
      setFacilities(res.rows || []);
      setTotal(res.total || 0);
      setPage(res.page || nextPage);
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load facilities");
    } finally {
      setLoading(false);
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleSearchChange = (e) => {
    setSearchQuery(e.target.value);
  };

  const handleLevelChange = (e) => {
    setFilterLevel(e.target.value || "");
  };

  const handleOwnershipChange = (e) => {
    setFilterOwnership(e.target.value || "");
  };

  const handleAuthorityChange = (e) => {
    setFilterAuthority(e.target.value || "");
  };

  const handleRegionChange = (e) => {
    setFilterRegion(e.target.value || "");
  };

  const handleDistrictChange = (e) => {
    setFilterDistrict(e.target.value || "");
  };

  const handleSubcountyChange = (e) => {
    setFilterSubcounty(e.target.value || "");
  };

  const handleClearFilters = () => {
    setSearchQuery("");
    setFilterLevel("");
    setFilterOwnership("");
    setFilterAuthority("");
    setFilterRegion("");
    setFilterDistrict("");
    setFilterSubcounty("");
  };

  const handlePageChange = (newPage) => {
    if (newPage < 1 || newPage > totalPages || newPage === page) return;
    loadFacilities(newPage);
  };

  const handleViewFacility = (facility) => {
    // Detail endpoint expects mfl_uid; fall back to error if missing
    if (facility?.mfl_uid) {
      history.push(`/initiator/facilities/${facility.mfl_uid}`);
      return;
    }
    setError("Facility mfl_uid is missing; cannot open details.");
  };

  const handleExport = async () => {
    try {
      setExporting(true);
      // Use export endpoint which returns all facilities (filtered or unfiltered)
      const params = buildQueryParams(1);
      // Remove pagination params from export query
      delete params.page;
      delete params.pageSize;
      
      const rows = await FacilitiesApi.export(params);

      const header = [
        "identifier",
        "name",
        "mfl_uid",
        "short_name",
        "historical_id",
        "region",
        "district",
        "subcounty",
        "parish",
        "village",
        "level",
        "ownership",
        "authority",
        "status",
        "reporting",
        "licensed",
        "address",
        "contact_personemail",
        "contact_personmobile",
        "contact_personname",
        "contact_persontitle",
        "longitude",
        "latitude",
        "opening_date",
        "closing_date",
        "bed_capacity",
        "services",
        "createdAt",
        "updatedAt",
      ];

      const valueForCsv = (val) => {
        if (val === null || val === undefined) return "";
        if (typeof val === "string" || typeof val === "number") return String(val);
        if (typeof val === "object") {
          return (
            val.name ||
            val.title ||
            val.label ||
            val.mfl_uid ||
            ""
          );
        }
        return "";
      };

      // Prevent Excel from converting long numeric identifiers to scientific notation
      const formatIdentifierForCsv = (val) => {
        const raw = valueForCsv(val);
        if (!raw) return "";
        // Prefix with a tab so Excel treats it as text
        return `\t${raw}`;
      };

      // Format ISO datetime strings to YYYY-MM-DD for readability
      const formatDateForCsv = (val) => {
        const raw = valueForCsv(val);
        if (!raw) return "";
        // If ISO-like (e.g. 2025-12-15T21:00:00.000Z) take only the date part
        if (/^\d{4}-\d{2}-\d{2}T/.test(raw)) {
          return raw.slice(0, 10);
        }
        return raw;
      };

      const lines = rows.map((f) => {
        const cols = [
          formatIdentifierForCsv(f.identifier),
          valueForCsv(f.name),
          valueForCsv(f.mfl_uid),
          valueForCsv(f.short_name),
          valueForCsv(f.historical_id),
          valueForCsv(f.region),
          valueForCsv(f.district),
          valueForCsv(f.subcounty),
          valueForCsv(f.parish),
          valueForCsv(f.village),
          valueForCsv(f.level),
          valueForCsv(f.ownership),
          valueForCsv(f.authority),
          valueForCsv(f.status),
          f.reporting ? "true" : "false",
          f.licensed ? "true" : "false",
          valueForCsv(f.address),
          valueForCsv(f.contact_personemail),
          valueForCsv(f.contact_personmobile),
          valueForCsv(f.contact_personname),
          valueForCsv(f.contact_persontitle),
          valueForCsv(f.longitude),
          valueForCsv(f.latitude),
          valueForCsv(f.opening_date),
          valueForCsv(f.closing_date),
          valueForCsv(f.bed_capacity),
          // Services might be an array or JSON – stringify for export
          Array.isArray(f.services)
            ? f.services.join("; ")
            : f.services != null
            ? JSON.stringify(f.services)
            : "",
          formatDateForCsv(f.createdAt),
          formatDateForCsv(f.updatedAt),
        ];
        // Escape double quotes and wrap in quotes to be safe
        return cols
          .map((c) => `"${String(c).replace(/"/g, '""')}"`)
          .join(",");
      });

      const csvContent = [header.join(","), ...lines].join("\r\n");
      const blob = new Blob([csvContent], {
        type: "text/csv;charset=utf-8;",
      });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", "facilities_export.csv");
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (e) {
      setError(
        e?.response?.data?.error ||
          e.message ||
          "Failed to export facilities"
      );
    } finally {
      setExporting(false);
    }
  };

  const handleExportExcel = () => {
    handleExport();
  };

  // Get unique regions and districts from facilities data as fallback
  const uniqueRegions = [...new Set(facilities.map(f => formatValue(f.region)).filter(v => v !== "—"))];
  const uniqueDistricts = [...new Set(facilities.map(f => formatValue(f.district)).filter(v => v !== "—"))];

  return (
    <Fragment>
      <div className="page-header">
        <div>
          <h2>My Field Stations</h2>
          <p>Browse field survey stations assigned to your operational area.</p>
        </div>
      </div>

      <div className="filter-section">
        <div className="filter-action-bar">
          <span className="filter-title">
            <i className="bi bi-building"></i> Filter Options
          </span>
          <div className="action-buttons-group">
            <button
              className="btn-action btn-filter"
              onClick={handleClearFilters}
              disabled={loading}
            >
              <i className="bi bi-x-circle"></i> Clear Filters
            </button>
            <button
              className="btn-action btn-excel"
              onClick={handleExportExcel}
              disabled={loading || exporting}
            >
              <i className="bi bi-file-earmark-excel"></i> Excel
            </button>
          </div>
        </div>

        <div className="row">
            <div className="col-3 filter-group">
              <label>Search Field Station</label>
              <input
                type="text"
                placeholder="Enter facility name or code..."
                value={searchQuery}
                onChange={handleSearchChange}
              />
            </div>
            <div className="col-3 filter-group">
              <label>Level</label>
              <select value={filterLevel} onChange={handleLevelChange}>
                <option value="">All Levels</option>
                {facilityLevels.map((lvl) => (
                  <option key={lvl.mfl_uid || lvl.id} value={lvl.name}>
                    {lvl.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="col-3 filter-group">
              <label>Operating Model</label>
              <select value={filterOwnership} onChange={handleOwnershipChange}>
                <option value="">All Operating Models</option>
                {ownershipTypes.map((o) => (
                  <option key={o.mfl_uid || o.id} value={o.name}>
                    {o.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="col-3 filter-group">
              <label>Managing Organisation</label>
              <select value={filterAuthority} onChange={handleAuthorityChange}>
                <option value="">All Managing Organisations</option>
                {authorityTypes.map((a) => (
                  <option key={a.mfl_uid || a.id} value={a.name}>
                    {a.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="col-3 filter-group">
              <label>Region</label>
              <select
                value={filterRegion}
                onChange={handleRegionChange}
              >
                <option value="">All Regions</option>
                {regions.length > 0
                  ? regions.map((region) => (
                      <option key={region.id || region.mfl_uid} value={region.name}>
                        {region.name}
                      </option>
                    ))
                  : uniqueRegions.map((region, idx) => (
                      <option key={idx} value={region}>
                        {region}
                      </option>
                    ))}
              </select>
            </div>
            <div className="col-3 filter-group">
              <label>District</label>
              <select
                value={filterDistrict}
                onChange={handleDistrictChange}
              >
                <option value="">All Districts</option>
                {districts.length > 0
                  ? districts.map((district) => (
                      <option key={district.id || district.mfl_uid} value={district.name}>
                        {district.name}
                      </option>
                    ))
                  : uniqueDistricts.map((district, idx) => (
                      <option key={idx} value={district}>
                        {district}
                      </option>
                    ))}
              </select>
            </div>
            <div className="col-3 filter-group">
              <label>Subcounty</label>
              <select value={filterSubcounty} onChange={handleSubcountyChange} disabled={!filterDistrict}>
                <option value="">All Subcounties</option>
                {subcounties.map((subcounty) => (
                  <option key={subcounty.id || subcounty.mfl_uid} value={subcounty.name}>
                    {subcounty.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
      </div>

      {error && (
        <div className="alert alert-danger" role="alert">
          {error}
        </div>
      )}

      <div className="table-section">
        <div className="table-header">
          <h3>
            <i className="bi bi-geo-alt"></i> Field Survey Stations ({total.toLocaleString()} records)
          </h3>
        </div>
        <table>
          <thead>
            <tr>
              <th>IDENTIFIER</th>
              <th>NAME</th>
              <th>STATION TIER</th>
              <th>OPERATING MODEL</th>
              <th>MANAGING ORGANISATION</th>
              <th>SUBCOUNTY</th>
              <th>DISTRICT</th>
              <th>REGION</th>
              <th>View</th>
            </tr>
          </thead>
          <tbody>
            {loading && facilities.length === 0 && (
              <tr>
                <td colSpan="9" style={{ textAlign: "center", padding: "2rem" }}>
                  Loading facilities...
                </td>
              </tr>
            )}
            {!loading && facilities.length === 0 && (
              <tr>
                <td colSpan="9" style={{ textAlign: "center", padding: "2rem" }}>
                  No facilities found.
                </td>
              </tr>
            )}
            {facilities.map((facility, index) => (
              <tr key={facility.id || facility.identifier || index}>
                <td>{formatIdentifier(facility.identifier)}</td>
                <td>{getFacilityDisplayName(facility)}</td>
                <td>
                  {facility.level?.name ||
                    facility.level_name ||
                    (typeof facility.level === "string"
                      ? facility.level
                      : "—")}
                </td>
                <td>
                  {facility.ownership?.name ||
                    facility.ownership_name ||
                    (typeof facility.ownership === "string"
                      ? facility.ownership
                      : "—")}
                </td>
                <td>
                  {facility.authority?.name ||
                    facility.authority_name ||
                    (typeof facility.authority === "string"
                      ? facility.authority
                      : "—")}
                </td>
                <td>{formatValue(facility.subcounty)}</td>
                <td>{formatValue(facility.district)}</td>
                <td>{formatValue(facility.region)}</td>
                <td style={{ padding: '0.4rem 0.75rem' }}>
                  <button
                    className="action-btn action-btn-view"
                    title="View Details"
                    type="button"
                    onClick={() => handleViewFacility(facility)}
                  >
                    <i className="bi bi-eye"></i>
                    <span>View</span>
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="pagination-section">
          <div>
            Showing {(page - 1) * pageSize + 1} to {Math.min(page * pageSize, total)} of{" "}
            {total.toLocaleString()} records
          </div>
          <div className="pagination">
            <button
              type="button"
              onClick={() => handlePageChange(page - 1)}
              disabled={page <= 1 || loading}
            >
              Previous
            </button>
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              let pageNum;
              if (totalPages <= 5) {
                pageNum = i + 1;
              } else if (page <= 3) {
                pageNum = i + 1;
              } else if (page >= totalPages - 2) {
                pageNum = totalPages - 4 + i;
              } else {
                pageNum = page - 2 + i;
              }
              return (
                <button
                  key={pageNum}
                  className={pageNum === page ? "active" : ""}
                  onClick={() => handlePageChange(pageNum)}
                  disabled={loading}
                >
                  {pageNum}
                </button>
              );
            })}
            {totalPages > 5 && page < totalPages - 2 && (
              <>
                <button disabled>...</button>
                <button onClick={() => handlePageChange(totalPages)} disabled={loading}>
                  {totalPages}
                </button>
              </>
            )}
            <button
              type="button"
              onClick={() => handlePageChange(page + 1)}
              disabled={page >= totalPages || loading}
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </Fragment>
  );
}
