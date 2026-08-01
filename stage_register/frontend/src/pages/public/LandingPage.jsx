import React, { Fragment, useEffect, useState } from "react";
import { publicApiClient } from "../../helpers/api";
import Header from './Header.jsx';
import Footer from "./Footer.jsx";

const LandingPage = () => {
  const [summary, setSummary] = useState([]);
  const [totals, setTotals] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [regions, setRegions] = useState([]);
  const [districts, setDistricts] = useState([]);
  const [dlgs, setDlgs] = useState([]);
  const [subcounties, setSubcounties] = useState([]);
  const [selectedRegion, setSelectedRegion] = useState("");
  const [selectedDistrict, setSelectedDistrict] = useState("");
  const [selectedDlg, setSelectedDlg] = useState("");
  const [selectedSubcounty, setSelectedSubcounty] = useState("");
  const [filtersLoading, setFiltersLoading] = useState(true);
  const [hierarchy, setHierarchy] = useState([]);

  // Load filter options once on mount
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        const res = await publicApiClient.get("/facilities/filters");
        const data = res.data || {};
        const hierarchyData = Array.isArray(data.hierarchy) ? data.hierarchy : [];
        setHierarchy(hierarchyData);

        // Derive unique top-level lists from hierarchy
        const regionsSet = new Set();
        const districtsSet = new Set();
        const dlgsSet = new Set();
        const subcountiesSet = new Set();

        hierarchyData.forEach((row) => {
          if (row.region) regionsSet.add(row.region);
          if (row.district) districtsSet.add(row.district);
          if (row.dlg) dlgsSet.add(row.dlg);
          if (row.subcounty) subcountiesSet.add(row.subcounty);
        });

        setRegions(Array.from(regionsSet));
        setDistricts(Array.from(districtsSet));
        setDlgs(Array.from(dlgsSet));
        setSubcounties(Array.from(subcountiesSet));
      } catch (err) {
        console.error("Failed to load filter options", err);
      } finally {
        setFiltersLoading(false);
      }
    };

    fetchFilterOptions();
  }, []);

  // Load statistics whenever region/district filters change
  useEffect(() => {
    const fetchStats = async () => {
      setLoading(true);
      setError(null);

      const params = {};
      if (selectedRegion) params.region = selectedRegion;
      if (selectedDistrict) params.district = selectedDistrict;
      if (selectedSubcounty) params.subcounty = selectedSubcounty;

      try {
        const [byLevelRes, totalsRes] = await Promise.all([
          publicApiClient.get("/facilities/summary/ownership-by-level", { params }),
          publicApiClient.get("/facilities/summary/ownership-totals", { params }),
        ]);

        const byLevelData = byLevelRes.data;
        const totalsData = totalsRes.data;

        setSummary(Array.isArray(byLevelData) ? byLevelData : []);
        setTotals(totalsData || null);
      } catch (err) {
        console.error("Failed to fetch facility summary", err);
        setError("Unable to load facility statistics at the moment.");
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, [selectedRegion, selectedDistrict, selectedDlg, selectedSubcounty]);

  // Cascade: update districts when region changes
  useEffect(() => {
    if (!hierarchy.length) return;

    const districtsSet = new Set();
    hierarchy.forEach((row) => {
      if (!selectedRegion || row.region === selectedRegion) {
        if (row.district) districtsSet.add(row.district);
      }
    });

    const nextDistricts = Array.from(districtsSet);
    setDistricts(nextDistricts);

    // Reset lower-level selections when parent changes
    setSelectedDistrict("");
    setSelectedDlg("");
    setSelectedSubcounty("");
  }, [selectedRegion, hierarchy]);

  // Cascade: update DLG/municipalities when district changes
  useEffect(() => {
    if (!hierarchy.length) return;

    const dlgsSet = new Set();
    hierarchy.forEach((row) => {
      const regionMatch = !selectedRegion || row.region === selectedRegion;
      const districtMatch = !selectedDistrict || row.district === selectedDistrict;
      if (regionMatch && districtMatch && row.dlg) {
        dlgsSet.add(row.dlg);
      }
    });

    const nextDlgs = Array.from(dlgsSet);
    setDlgs(nextDlgs);

    setSelectedDlg("");
    setSelectedSubcounty("");
  }, [selectedDistrict, selectedRegion, hierarchy]);

  // Cascade: update subcounties when DLG changes
  useEffect(() => {
    if (!hierarchy.length) return;

    const subcountiesSet = new Set();
    hierarchy.forEach((row) => {
      const regionMatch = !selectedRegion || row.region === selectedRegion;
      const districtMatch = !selectedDistrict || row.district === selectedDistrict;
      const dlgMatch = !selectedDlg || row.dlg === selectedDlg;
      if (regionMatch && districtMatch && dlgMatch && row.subcounty) {
        subcountiesSet.add(row.subcounty);
      }
    });

    const nextSubcounties = Array.from(subcountiesSet);
    setSubcounties(nextSubcounties);
    setSelectedSubcounty("");
  }, [selectedDlg, selectedDistrict, selectedRegion, hierarchy]);

  const formatNumber = (value) =>
    typeof value === "number" ? value.toLocaleString() : value;

  return (
    <Fragment>
      <Header />
      <section className="hero-section">
        <div className="hero-content mt-3">
          <h2>StatGate – Field Operations & Agent Workforce Registry</h2>
          <p>
            Welcome to the StatGate Field Operations & Agent Workforce Registry. The Registry is a complete
            listing of field agents, enumerators, and field survey stations across the country. Track agent status,
            recruitment pipelines, and operational sampling territories in real time.
          </p>
        </div>
      </section>

      <section class="statistics-wrapper">
        <div class="statistics-content">
          <div class="filters-row">
            <div class="filter-group">
              <label class="filter-label">Region</label>
              <select
                class="filter-select"
                value={selectedRegion}
                onChange={(e) => setSelectedRegion(e.target.value)}
                disabled={filtersLoading}
              >
                <option value="">All Regions</option>
                {regions.map((region) => (
                  <option key={region} value={region}>
                    {region}
                  </option>
                ))}
              </select>
            </div>
            <div class="filter-group">
              <label class="filter-label">District</label>
              <select
                class="filter-select"
                value={selectedDistrict}
                onChange={(e) => setSelectedDistrict(e.target.value)}
                disabled={filtersLoading}
              >
                <option value="">All Districts</option>
                {districts.map((district) => (
                  <option key={district} value={district}>
                    {district}
                  </option>
                ))}
              </select>
            </div>
            <div class="filter-group">
              <label class="filter-label">DLG / Municipality</label>
              <select
                class="filter-select"
                value={selectedDlg}
                onChange={(e) => setSelectedDlg(e.target.value)}
                disabled={filtersLoading || !districts.length}
              >
                <option value="">All DLGs / Municipalities</option>
                {dlgs.map((dlg) => (
                  <option key={dlg} value={dlg}>
                    {dlg}
                  </option>
                ))}
              </select>
            </div>
            <div class="filter-group">
              <label class="filter-label">Subcounty</label>
              <select
                class="filter-select"
                value={selectedSubcounty}
                onChange={(e) => setSelectedSubcounty(e.target.value)}
                disabled={filtersLoading || (!dlgs.length && !selectedDistrict)}
              >
                <option value="">All Subcounties</option>
                {subcounties.map((subcounty) => (
                  <option key={subcounty} value={subcounty}>
                    {subcounty}
                  </option>
                ))}
              </select>
            </div>
            <div class="filter-actions">
              <button
                type="button"
                class="btn-clear-filters"
                onClick={() => {
                  setSelectedRegion("");
                  setSelectedDistrict("");
                  setSelectedDlg("");
                  setSelectedSubcounty("");
                }}
                disabled={
                  filtersLoading ||
                  (!selectedRegion && !selectedDistrict && !selectedDlg && !selectedSubcounty)
                }
              >
                Clear filters
              </button>
            </div>
          </div>

          <div class="stats-grid">
            <div class="stat-card">
              <div class="stat-card-header">
                <i class="bi bi-geo-alt stat-icon"></i>
                <span class="stat-card-title">Total</span>
              </div>
              <div class="stat-items-grid" style={{ gridTemplateColumns: '1fr' }}>
                <div class="stat-item">
                  <i class="bi bi-geo-alt stat-item-icon"></i>
                  <div class="stat-item-value">
                    {loading || !totals ? "…" : formatNumber(totals.total || 0)}
                  </div>
                  <div class="stat-item-label">Field Survey Stations</div>
                </div>
              </div>
            </div>

            <div class="stat-card">
              <div class="stat-card-header">
                <i class="bi bi-diagram-2 stat-icon"></i>
                <span class="stat-card-title">Operating Model</span>
              </div>
              <div class="stat-items-grid">
                <div class="stat-item">
                  <i class="bi bi-building stat-item-icon"></i>
                  <div class="stat-item-value">
                    {loading || !totals ? "…" : formatNumber(totals.government || 0)}
                  </div>
                  <div class="stat-item-label">Public Sector</div>
                </div>
                <div class="stat-item">
                  <i class="bi bi-building stat-item-icon"></i>
                  <div class="stat-item-value">
                    {loading || !totals ? "…" : formatNumber(totals.pnfp || 0)}
                  </div>
                  <div class="stat-item-label">Non-profit Partner</div>
                </div>
                <div class="stat-item">
                  <i class="bi bi-building stat-item-icon"></i>
                  <div class="stat-item-value">
                    {loading || !totals ? "…" : formatNumber(totals.pfp || 0)}
                  </div>
                  <div class="stat-item-label">Private Partner</div>
                </div>
              </div>
            </div>

            {/* <div class="stat-card">
              <div class="stat-card-header">
                <i class="bi bi-check-circle stat-icon"></i>
                <span class="stat-card-title">Functionality</span>
              </div>
              <div class="stat-items-grid two-col">
                <div class="stat-item">
                  <i class="bi bi-check-circle stat-item-icon"></i>
                  <div class="stat-item-value">4888</div>
                  <div class="stat-item-label">Functional</div>
                </div>
                <div class="stat-item">
                  <i class="bi bi-x-circle stat-item-icon"></i>
                  <div class="stat-item-value">1812</div>
                  <div class="stat-item-label">Non Functional</div>
                </div>
              </div>
            </div> */}
          </div>

          <div class="table-section">
            <div class="table-header">
              <h3 class="table-title">
                <i class="bi bi-table table-icon"></i>
                Field Survey Stations by Tier and Operating Model
              </h3>
            </div>

            {/* <div class="filters-row">
              <div class="filter-group">
                <label class="filter-label">Region</label>
                <select class="filter-select">
                  <option>All Regions</option>
                  <option>Central</option>
                  <option>Eastern</option>
                  <option>Northern</option>
                  <option>Western</option>
                </select>
              </div>
              <div class="filter-group">
                <label class="filter-label">District</label>
                <select class="filter-select">
                  <option>All Districts</option>
                  <option>Kampala</option>
                  <option>Wakiso</option>
                  <option>Mukono</option>
                  <option>Jinja</option>
                </select>
              </div>
            </div> */}

            {error && (
              <div class="alert alert-warning" role="alert">
                {error}
              </div>
            )}

            <table class="summary-table">
              <thead>
                <tr>
                  <th>Station Tier</th>
                  <th>Public Sector</th>
                  <th>Non-profit Partner</th>
                  <th>Private Partner</th>
                  <th>Total</th>
                </tr>
              </thead>
              <tbody>
                {loading && summary.length === 0 ? (
                  <tr>
                    <td colSpan="5">Loading summary data…</td>
                  </tr>
                ) : (
                  summary.map((row) => (
                    <tr key={row.level}>
                      <td>
                        <strong>{row.level}</strong>
                      </td>
                      <td>{formatNumber(row.government || 0)}</td>
                      <td>{formatNumber(row.pnfp || 0)}</td>
                      <td>{formatNumber(row.pfp || 0)}</td>
                      <td>
                        <strong>{formatNumber(row.total || 0)}</strong>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </section>
      <Footer />
    </Fragment>
  );
}

export default LandingPage;
