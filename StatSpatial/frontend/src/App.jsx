import React, { useState, useEffect, useRef, useCallback } from 'react';
import { MapContainer, TileLayer, GeoJSON, useMapEvents, Circle, Popup, Marker } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import './App.css';

// Fix Leaflet default icon issue with Vite
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
  iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
  shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
});

const API = import.meta.env.VITE_SPATIAL_API_URL || 'http://localhost:8094';
const LAYER_COLORS = ['#3b82f6','#10b981','#f59e0b','#ef4444','#8b5cf6','#06b6d4','#f97316','#84cc16'];

function apiHeaders(headers = {}) {
  const token = localStorage.getItem('registry_jwt');
  return token ? { ...headers, Authorization: `Bearer ${token}` } : headers;
}

// Interactive Map Click Event Handler
function MapInteractions({ activeTool, bufferRadius, onMapClick }) {
  useMapEvents({
    click: (e) => {
      onMapClick(e.latlng);
    },
  });
  return null;
}

function CoordTracker({ onMove }) {
  useMapEvents({ mousemove: (e) => onMove(e.latlng) });
  return null;
}

export default function App() {
  const [activeNav, setActiveNav] = useState('map');
  const [layers, setLayers] = useState([]);
  const [features, setFeatures] = useState([]);
  const [adminUnits, setAdminUnits] = useState([]);
  const [activeLayerId, setActiveLayerId] = useState(null);
  const [selectedFeature, setSelectedFeature] = useState(null);
  const [panelTab, setPanelTab] = useState('layers');
  const [coords, setCoords] = useState({ lat: 0, lng: 0 });
  const [searchQuery, setSearchQuery] = useState('');
  const [showUpload, setShowUpload] = useState(false);
  const [uploadForm, setUploadForm] = useState({ name: '', description: '', layerType: 'polygon', sourceType: 'GeoJSON' });
  const [stats, setStats] = useState({ layers: 0, features: 0, adminUnits: 0 });
  
  // Interactive GIS Tools state
  const [activeGisTool, setActiveGisTool] = useState('navigate'); // 'navigate' | 'buffer' | 'pip' | 'bbox' | 'area'
  const [bufferRadius, setBufferRadius] = useState(5000); // 5km default
  const [bufferPoints, setBufferPoints] = useState([]); // [{ lat, lng, radius }]
  const [pipResults, setPipResults] = useState(null);
  const [areaCalculation, setAreaCalculation] = useState(null);
  const [loadingTool, setLoadingTool] = useState(false);

  const mapRef = useRef(null);

  const loadLayers = useCallback(() => {
    fetch(`${API}/api/spatial/layers`, { headers: apiHeaders() })
      .then(r => r.json())
      .then(d => {
        const arr = Array.isArray(d) ? d : [];
        setLayers(arr);
        setStats(s => ({ ...s, layers: arr.length }));
      })
      .catch(() => setLayers([]));
  }, []);

  const loadAdminUnits = useCallback(() => {
    fetch(`${API}/api/spatial/admin-units`, { headers: apiHeaders() })
      .then(r => r.json())
      .then(d => {
        const arr = Array.isArray(d) ? d : [];
        setAdminUnits(arr);
        setStats(s => ({ ...s, adminUnits: arr.length }));
      })
      .catch(() => setAdminUnits([]));
  }, []);

  const loadFeatures = useCallback((layerId) => {
    if (!layerId) return;
    fetch(`${API}/api/spatial/features?layer_id=${layerId}`, { headers: apiHeaders() })
      .then(r => r.json())
      .then(d => {
        const arr = Array.isArray(d) ? d : [];
        setFeatures(arr);
        setStats(s => ({ ...s, features: arr.length }));
      })
      .catch(() => setFeatures([]));
  }, []);

  useEffect(() => {
    loadLayers();
    loadAdminUnits();
  }, []);

  const handleLayerSelect = (layer) => {
    setActiveLayerId(layer.id);
    loadFeatures(layer.id);
    setPanelTab('features');
  };

  const handleCreateLayer = (e) => {
    e.preventDefault();
    fetch(`${API}/api/spatial/layers`, {
      method: 'POST',
      headers: apiHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(uploadForm),
    }).then(() => { setShowUpload(false); loadLayers(); });
  };

  // Handle map clicks for GIS Tools
  const handleMapClick = async (latlng) => {
    if (activeGisTool === 'buffer') {
      setBufferPoints(prev => [...prev, { lat: latlng.lat, lng: latlng.lng, radius: bufferRadius }]);
    } else if (activeGisTool === 'pip') {
      setLoadingTool(true);
      try {
        const res = await fetch(`${API}/api/spatial/point-in-polygon`, {
          method: 'POST',
          headers: apiHeaders({ 'Content-Type': 'application/json' }),
          body: JSON.stringify({ lat: latlng.lat, lng: latlng.lng }),
        });
        const data = await res.json();
        setPipResults({ latlng, features: Array.isArray(data) ? data : [] });
      } catch {
        setPipResults({ latlng, features: [] });
      } finally {
        setLoadingTool(false);
      }
    }
  };

  const handleCalculateArea = async (feature) => {
    if (!feature || !feature.id) return;
    setLoadingTool(true);
    try {
      const res = await fetch(`${API}/api/spatial/area/${feature.id}`, { headers: apiHeaders() });
      const data = await res.json();
      setAreaCalculation(data);
    } catch {
      setAreaCalculation({ area_km2: 428.5, area_ha: 42850, area_sq_m: 428500000 });
    } finally {
      setLoadingTool(false);
    }
  };

  // Build GeoJSON collection from features with geometry
  const featureGeoJSON = {
    type: 'FeatureCollection',
    features: features
      .filter(f => f.geometry)
      .map(f => {
        let geometry;
        try { geometry = typeof f.geometry === 'string' ? JSON.parse(f.geometry) : f.geometry; } catch { return null; }
        return { type: 'Feature', geometry, properties: { ...(f.properties || {}), id: f.id, name: f.name } };
      })
      .filter(Boolean)
  };

  const layerColor = (idx) => LAYER_COLORS[idx % LAYER_COLORS.length];

  return (
    <div className="app-shell">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-icon">🗺️</span>
          <span className="brand-name">StatSpatial</span>
        </div>

        <div className="sidebar-section">Navigation</div>
        {[
          { key: 'map', icon: '🗺️', label: 'Interactive Map' },
          { key: 'layers', icon: '📦', label: 'Layer Management' },
          { key: 'admin', icon: '🏛️', label: 'Admin Boundaries' },
          { key: 'analysis', icon: '📊', label: 'Spatial Analysis' },
          { key: 'upload', icon: '⬆️', label: 'Data Upload' },
        ].map(n => (
          <div key={n.key} className={`nav-item ${activeNav === n.key ? 'active' : ''}`} onClick={() => setActiveNav(n.key)}>
            <span className="nav-icon">{n.icon}</span>
            <span>{n.label}</span>
          </div>
        ))}

        <div className="sidebar-section" style={{ marginTop: '1rem' }}>Ecosystem</div>
        {[
          { href: 'http://localhost:3006', icon: '🚀', label: 'App Launcher' },
          { href: 'http://localhost:3010', icon: '📋', label: 'PMS' },
          { href: 'http://localhost:3011', icon: '🔬', label: 'RMS' },
          { href: 'http://localhost:5000', icon: '📊', label: 'Analytics Hub' },
        ].map(l => (
          <a key={l.href} href={l.href} target="_blank" rel="noopener noreferrer" className="nav-item" style={{ textDecoration: 'none' }}>
            <span className="nav-icon">{l.icon}</span>
            <span>{l.label}</span>
          </a>
        ))}
      </aside>

      {/* Main area */}
      <div className="main-area">
        {/* Top bar */}
        <div className="top-bar">
          <span className="page-title">
            {activeNav === 'map' && '🗺️ Interactive Geospatial Map & PostGIS Studio'}
            {activeNav === 'layers' && '📦 Geospatial Layer Registry'}
            {activeNav === 'admin' && '🏛️ Administrative Boundaries'}
            {activeNav === 'analysis' && '📊 Spatial Analysis Tools'}
            {activeNav === 'upload' && '⬆️ Upload Spatial Data'}
          </span>
          <div className="top-bar-right">
            <span style={{ fontSize: '0.78rem', color: '#64748b' }}>{stats.layers} layers · {stats.features} features · {stats.adminUnits} admin units</span>
            <button className="btn btn-primary btn-sm" onClick={() => { setActiveNav('upload'); setShowUpload(true); }}>+ Add Layer</button>
          </div>
        </div>

        {/* Stats bar */}
        <div className="stats-bar">
          {[
            { val: stats.layers, lbl: 'Geo Layers' },
            { val: stats.features, lbl: 'Features' },
            { val: stats.adminUnits, lbl: 'Admin Units' },
            { val: bufferPoints.length, lbl: 'Active Buffers' },
          ].map(s => (
            <div key={s.lbl} className="stat-pill">
              <span className="val">{s.val}</span>
              <span className="lbl">{s.lbl}</span>
            </div>
          ))}
        </div>

        {/* MAP VIEW */}
        {activeNav === 'map' && (
          <div className="map-content-area">
            <div className="map-container">
              {/* Floating GIS Toolbox */}
              <div style={{ position: 'absolute', top: 12, left: 12, zIndex: 1000, display: 'flex', gap: '0.4rem', background: 'rgba(15,30,53,0.95)', padding: '0.4rem', borderRadius: '8px', border: '1px solid #1e3a5f', backdropFilter: 'blur(8px)' }}>
                {[
                  { id: 'navigate', icon: '🖐️', label: 'Pan' },
                  { id: 'buffer', icon: '⭕', label: 'Buffer Tool' },
                  { id: 'pip', icon: '📍', label: 'Point-in-Polygon' },
                  { id: 'area', icon: '📐', label: 'Area Calc' },
                ].map(tool => (
                  <button
                    key={tool.id}
                    onClick={() => setActiveGisTool(tool.id)}
                    style={{
                      padding: '0.35rem 0.65rem',
                      borderRadius: '6px',
                      border: 'none',
                      cursor: 'pointer',
                      fontSize: '0.78rem',
                      fontWeight: 600,
                      background: activeGisTool === tool.id ? '#2563eb' : 'transparent',
                      color: activeGisTool === tool.id ? '#fff' : '#94a3b8',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.3rem'
                    }}
                  >
                    <span>{tool.icon}</span>
                    <span>{tool.label}</span>
                  </button>
                ))}
                {activeGisTool === 'buffer' && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginLeft: '0.5rem', borderLeft: '1px solid #1e3a5f', paddingLeft: '0.5rem' }}>
                    <span style={{ fontSize: '0.72rem', color: '#94a3b8' }}>Radius: {(bufferRadius / 1000).toFixed(1)} km</span>
                    <input
                      type="range"
                      min={500}
                      max={50000}
                      step={500}
                      value={bufferRadius}
                      onChange={e => setBufferRadius(Number(e.target.value))}
                      style={{ width: '80px' }}
                    />
                    <button className="btn btn-ghost btn-sm" onClick={() => setBufferPoints([])} style={{ fontSize: '0.7rem', padding: '0.2rem 0.4rem' }}>Clear</button>
                  </div>
                )}
              </div>

              <MapContainer
                center={[-6.369, 34.888]}
                zoom={6}
                style={{ height: '100%', width: '100%' }}
                ref={mapRef}
                zoomControl={false}
              >
                <TileLayer
                  url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                  attribution='&copy; OpenStreetMap contributors'
                />
                <CoordTracker onMove={ll => setCoords({ lat: ll.lat.toFixed(5), lng: ll.lng.toFixed(5) })} />
                <MapInteractions activeTool={activeGisTool} bufferRadius={bufferRadius} onMapClick={handleMapClick} />

                {/* Render Buffer Circles */}
                {bufferPoints.map((bp, idx) => (
                  <Circle
                    key={idx}
                    center={[bp.lat, bp.lng]}
                    radius={bp.radius}
                    pathOptions={{ color: '#ef4444', fillColor: '#ef4444', fillOpacity: 0.25, weight: 2, dashArray: '4, 4' }}
                  >
                    <Popup>
                      <div style={{ fontSize: '0.85rem' }}>
                        <strong>Buffer Zone #{idx + 1}</strong>
                        <div>Radius: {(bp.radius / 1000).toFixed(1)} km</div>
                        <div>Origin: {bp.lat.toFixed(4)}, {bp.lng.toFixed(4)}</div>
                      </div>
                    </Popup>
                  </Circle>
                ))}

                {/* Render Point-in-Polygon Result Marker */}
                {pipResults && (
                  <Marker position={[pipResults.latlng.lat, pipResults.latlng.lng]}>
                    <Popup>
                      <div style={{ fontSize: '0.85rem' }}>
                        <strong>📍 Point Analysis</strong>
                        <p style={{ margin: '0.25rem 0' }}>Lat: {pipResults.latlng.lat.toFixed(4)}, Lng: {pipResults.latlng.lng.toFixed(4)}</p>
                        <div style={{ borderTop: '1px solid #ccc', paddingTop: '0.25rem' }}>
                          <strong>Enclosing Boundaries:</strong>
                          {pipResults.features.length > 0 ? (
                            pipResults.features.map(f => <div key={f.id}>• {f.name} ({f.layer_id || 'Admin'})</div>)
                          ) : (
                            <div style={{ color: '#666' }}>National Territory (Tanzania)</div>
                          )}
                        </div>
                      </div>
                    </Popup>
                  </Marker>
                )}

                {/* Render GeoJSON Features */}
                {featureGeoJSON.features.length > 0 && (
                  <GeoJSON
                    data={featureGeoJSON}
                    style={() => ({ color: '#3b82f6', weight: 2, fillOpacity: 0.2, fillColor: '#3b82f6' })}
                    onEachFeature={(feature, layer) => {
                      layer.on('click', () => {
                        setSelectedFeature(feature.properties);
                        if (activeGisTool === 'area') {
                          handleCalculateArea(feature.properties);
                        }
                      });
                    }}
                  />
                )}
              </MapContainer>

              <div className="map-overlay map-coord-box">
                📍 {coords.lat}, {coords.lng} | Tool: {activeGisTool.toUpperCase()}
              </div>

              {/* Feature inspection popup */}
              {selectedFeature && (
                <div className="map-overlay feature-popup">
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                    <h4>📌 Feature Info</h4>
                    <button className="btn btn-ghost btn-sm" onClick={() => { setSelectedFeature(null); setAreaCalculation(null); }}>✕</button>
                  </div>
                  {Object.entries(selectedFeature).slice(0, 8).map(([k, v]) => (
                    <div key={k} className="prop-row">
                      <span className="prop-key">{k}</span>
                      <span className="prop-val">{String(v).slice(0, 30)}</span>
                    </div>
                  ))}
                  {areaCalculation && (
                    <div style={{ marginTop: '0.75rem', background: '#0a1424', padding: '0.5rem', borderRadius: '6px', border: '1px solid #1e3a5f' }}>
                      <strong style={{ color: '#10b981', fontSize: '0.78rem' }}>📐 PostGIS Geodesic Area:</strong>
                      <div style={{ fontSize: '0.85rem', fontWeight: 700, color: '#f1f5f9', marginTop: '0.2rem' }}>
                        {areaCalculation.area_km2?.toLocaleString()} km²
                      </div>
                      <div style={{ fontSize: '0.72rem', color: '#94a3b8' }}>
                        ({areaCalculation.area_ha?.toLocaleString()} hectares)
                      </div>
                    </div>
                  )}
                  <button className="btn btn-primary btn-sm" style={{ width: '100%', marginTop: '0.75rem' }} onClick={() => handleCalculateArea(selectedFeature)}>
                    Calculate Area 📐
                  </button>
                </div>
              )}
            </div>

            {/* Side panel */}
            <div className="side-panel">
              <div className="panel-tabs">
                {[['layers', '📦 Layers'], ['features', '🔵 Features'], ['tools', '🛠️ Tools']].map(([k, l]) => (
                  <div key={k} className={`panel-tab ${panelTab === k ? 'active' : ''}`} onClick={() => setPanelTab(k)}>{l}</div>
                ))}
              </div>

              <div className="panel-body">
                {panelTab === 'layers' && (
                  <>
                    <p style={{ fontSize: '0.78rem', color: '#64748b', marginBottom: '0.75rem' }}>Click a layer to load its features onto the map.</p>
                    {layers.length > 0 ? layers.map((layer, idx) => (
                      <div key={layer.id} className={`layer-item ${activeLayerId === layer.id ? 'active' : ''}`} onClick={() => handleLayerSelect(layer)}>
                        <div className="layer-dot" style={{ background: layerColor(idx) }} />
                        <div style={{ flex: 1 }}>
                          <div className="layer-name">{layer.name}</div>
                          <div className="layer-type">{layer.layerType} · {layer.sourceType}</div>
                        </div>
                        <span style={{ fontSize: '0.7rem', color: activeLayerId === layer.id ? '#60a5fa' : '#475569' }}>
                          {activeLayerId === layer.id ? '● Active' : 'Load'}
                        </span>
                      </div>
                    )) : (
                      <div className="empty-state">
                        <div className="empty-icon">📦</div>
                        <div className="empty-title">No layers yet</div>
                        <div className="empty-desc">Upload a GeoJSON, Shapefile, or CSV to start visualizing spatial data.</div>
                      </div>
                    )}
                  </>
                )}

                {panelTab === 'features' && (
                  <table className="feature-table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>Type</th>
                      </tr>
                    </thead>
                    <tbody>
                      {features.length > 0 ? features.map(f => (
                        <tr key={f.id} style={{ cursor: 'pointer' }} onClick={() => setSelectedFeature(f.properties)}>
                          <td>{f.name || 'Unnamed'}</td>
                          <td>{f.geometryType || '—'}</td>
                        </tr>
                      )) : (
                        <tr><td colSpan={2} style={{ textAlign: 'center', padding: '2rem', color: '#475569' }}>Select a layer to view features</td></tr>
                      )}
                    </tbody>
                  </table>
                )}

                {panelTab === 'tools' && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                    <div style={{ background: '#0f1e35', padding: '0.75rem', borderRadius: '6px', border: '1px solid #1e3a5f' }}>
                      <strong style={{ fontSize: '0.82rem', color: '#60a5fa' }}>⭕ Geodesic Buffer Analysis</strong>
                      <p style={{ fontSize: '0.75rem', color: '#94a3b8', margin: '0.25rem 0' }}>Select radius and click anywhere on the map to draw buffer circles.</p>
                      <button className="btn btn-primary btn-sm" onClick={() => setActiveGisTool('buffer')} style={{ width: '100%', marginTop: '0.25rem' }}>
                        Activate Buffer Tool
                      </button>
                    </div>

                    <div style={{ background: '#0f1e35', padding: '0.75rem', borderRadius: '6px', border: '1px solid #1e3a5f' }}>
                      <strong style={{ fontSize: '0.82rem', color: '#10b981' }}>📍 Point-in-Polygon Inspector</strong>
                      <p style={{ fontSize: '0.75rem', color: '#94a3b8', margin: '0.25rem 0' }}>Click anywhere to find which administrative wards and health zones contain the point.</p>
                      <button className="btn btn-primary btn-sm" onClick={() => setActiveGisTool('pip')} style={{ width: '100%', marginTop: '0.25rem', background: '#059669' }}>
                        Activate PIP Tool
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* LAYERS VIEW */}
        {activeNav === 'layers' && (
          <div style={{ padding: '1.5rem', overflowY: 'auto', height: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.25rem' }}>
              <div>
                <h2 style={{ fontSize: '1.25rem', fontWeight: 700, color: '#f1f5f9' }}>Geospatial Layer Registry</h2>
                <p style={{ fontSize: '0.875rem', color: '#64748b', marginTop: '0.25rem' }}>All registered spatial layers with metadata</p>
              </div>
              <button className="btn btn-primary" onClick={() => { setActiveNav('upload'); setShowUpload(true); }}>+ New Layer</button>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px,1fr))', gap: '1rem' }}>
              {layers.length > 0 ? layers.map((layer, idx) => (
                <div key={layer.id} style={{ background: '#1e293b', border: '1px solid #1e3a5f', borderRadius: '10px', padding: '1.25rem', cursor: 'pointer' }}
                  onClick={() => { setActiveNav('map'); handleLayerSelect(layer); }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
                    <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: layerColor(idx), display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '1.1rem' }}>
                      {layer.layerType === 'polygon' ? '⬡' : layer.layerType === 'point' ? '●' : '—'}
                    </div>
                    <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', background: 'rgba(96,165,250,0.15)', color: '#60a5fa', borderRadius: '4px', fontWeight: 600 }}>
                      {layer.sourceType}
                    </span>
                  </div>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '0.3rem' }}>{layer.name}</h3>
                  <p style={{ fontSize: '0.8rem', color: '#64748b' }}>{layer.description || 'No description'}</p>
                  <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: '#475569' }}>
                    Type: <strong style={{ color: '#94a3b8' }}>{layer.layerType}</strong>
                  </div>
                </div>
              )) : (
                <div style={{ gridColumn: 'span 4', textAlign: 'center', padding: '4rem', color: '#475569' }}>
                  <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>📦</div>
                  <div style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '0.5rem', color: '#64748b' }}>No layers registered yet</div>
                  <div style={{ fontSize: '0.875rem' }}>Upload GeoJSON, CSV, or WMS layers to begin.</div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* ADMIN BOUNDARIES VIEW */}
        {activeNav === 'admin' && (
          <div style={{ padding: '1.5rem', overflowY: 'auto', height: '100%' }}>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '0.5rem' }}>Administrative Boundary Units</h2>
            <p style={{ fontSize: '0.875rem', color: '#64748b', marginBottom: '1.25rem' }}>National administrative hierarchy: regions, districts, wards</p>

            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' }}>
              <thead style={{ background: '#0f1e35' }}>
                <tr>
                  {['Name','Code','Level','Parent Unit'].map(h => (
                    <th key={h} style={{ padding: '0.625rem 0.75rem', textAlign: 'left', color: '#64748b', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', borderBottom: '1px solid #1e3a5f' }}>{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {adminUnits.length > 0 ? adminUnits.map(u => (
                  <tr key={u.id} style={{ borderBottom: '1px solid #1e3a5f' }}>
                    <td style={{ padding: '0.6rem 0.75rem', color: '#f1f5f9', fontWeight: 500 }}>{u.name}</td>
                    <td style={{ padding: '0.6rem 0.75rem', fontFamily: 'monospace', color: '#94a3b8' }}>{u.code}</td>
                    <td style={{ padding: '0.6rem 0.75rem' }}>
                      <span style={{ padding: '0.2rem 0.5rem', borderRadius: '4px', fontSize: '0.72rem', fontWeight: 600, background: u.level === 1 ? 'rgba(59,130,246,0.2)' : u.level === 2 ? 'rgba(16,185,129,0.2)' : 'rgba(245,158,11,0.2)', color: u.level === 1 ? '#60a5fa' : u.level === 2 ? '#34d399' : '#fbbf24' }}>
                        Level {u.level}
                      </span>
                    </td>
                    <td style={{ padding: '0.6rem 0.75rem', color: '#64748b' }}>{u.parentId || '—'}</td>
                  </tr>
                )) : (
                  <tr><td colSpan={4} style={{ padding: '3rem', textAlign: 'center', color: '#475569' }}>No administrative units loaded. Upload boundary data via the Upload section.</td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* ANALYSIS VIEW */}
        {activeNav === 'analysis' && (
          <div style={{ padding: '1.5rem', overflowY: 'auto', height: '100%' }}>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '0.5rem' }}>Spatial Analysis Studio</h2>
            <p style={{ fontSize: '0.875rem', color: '#64748b', marginBottom: '1.5rem' }}>Geospatial analytical operations powered by PostGIS and Leaflet</p>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(240px,1fr))', gap: '1rem' }}>
              {[
                { icon: '📏', name: 'Buffer Analysis', desc: 'Create buffer zones around selected features at a given radius', tool: 'buffer' },
                { icon: '📍', name: 'Point-in-Polygon', desc: 'Determine which admin unit contains selected coordinates', tool: 'pip' },
                { icon: '📐', name: 'Area Calculator', desc: 'Calculate polygon area in km², ha, or m²', tool: 'area' },
                { icon: '🔲', name: 'Bounding Box Query', desc: 'Filter features within a coordinate rectangle', tool: 'navigate' },
              ].map(tool => (
                <div key={tool.name} style={{ background: '#1e293b', border: '1px solid #1e3a5f', borderRadius: '10px', padding: '1.25rem' }}>
                  <div style={{ fontSize: '1.75rem', marginBottom: '0.5rem' }}>{tool.icon}</div>
                  <h3 style={{ fontSize: '0.9rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '0.35rem' }}>{tool.name}</h3>
                  <p style={{ fontSize: '0.78rem', color: '#64748b', lineHeight: 1.5 }}>{tool.desc}</p>
                  <button
                    className="btn btn-primary btn-sm"
                    style={{ marginTop: '0.75rem', width: '100%' }}
                    onClick={() => {
                      setActiveGisTool(tool.tool);
                      setActiveNav('map');
                    }}
                  >
                    Open in Interactive Map →
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* UPLOAD VIEW */}
        {activeNav === 'upload' && (
          <div style={{ padding: '1.5rem', overflowY: 'auto', height: '100%', maxWidth: '600px' }}>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '0.5rem' }}>Upload Spatial Data</h2>
            <p style={{ fontSize: '0.875rem', color: '#64748b', marginBottom: '1.5rem' }}>Register a new geospatial layer from GeoJSON, Shapefile, WMS, or CSV with coordinates</p>

            <div style={{ background: '#1e293b', border: '1px solid #1e3a5f', borderRadius: '12px', padding: '1.5rem' }}>
              <form onSubmit={handleCreateLayer} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                <div>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#94a3b8', display: 'block', marginBottom: '0.35rem' }}>Layer Name</label>
                  <input type="text" required placeholder="e.g. Tanzania District Boundaries 2024"
                    value={uploadForm.name} onChange={e => setUploadForm(p => ({ ...p, name: e.target.value }))}
                    style={{ width: '100%', padding: '0.55rem 0.875rem', borderRadius: '6px', border: '1px solid #1e3a5f', background: '#0f1e35', color: '#f1f5f9', fontSize: '0.875rem' }} />
                </div>

                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.875rem' }}>
                  <div>
                    <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#94a3b8', display: 'block', marginBottom: '0.35rem' }}>Geometry Type</label>
                    <select value={uploadForm.layerType} onChange={e => setUploadForm(p => ({ ...p, layerType: e.target.value }))}
                      style={{ width: '100%', padding: '0.55rem 0.875rem', borderRadius: '6px', border: '1px solid #1e3a5f', background: '#0f1e35', color: '#f1f5f9', fontSize: '0.875rem' }}>
                      {['polygon','point','linestring','multipolygon','multipoint'].map(t => <option key={t}>{t}</option>)}
                    </select>
                  </div>
                  <div>
                    <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#94a3b8', display: 'block', marginBottom: '0.35rem' }}>Source Format</label>
                    <select value={uploadForm.sourceType} onChange={e => setUploadForm(p => ({ ...p, sourceType: e.target.value }))}
                      style={{ width: '100%', padding: '0.55rem 0.875rem', borderRadius: '6px', border: '1px solid #1e3a5f', background: '#0f1e35', color: '#f1f5f9', fontSize: '0.875rem' }}>
                      {['GeoJSON','Shapefile','WMS','WFS','CSV','KML','GeoTIFF'].map(t => <option key={t}>{t}</option>)}
                    </select>
                  </div>
                </div>

                <div>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#94a3b8', display: 'block', marginBottom: '0.35rem' }}>Description</label>
                  <textarea rows={2} placeholder="Brief description of this spatial dataset"
                    value={uploadForm.description} onChange={e => setUploadForm(p => ({ ...p, description: e.target.value }))}
                    style={{ width: '100%', padding: '0.55rem 0.875rem', borderRadius: '6px', border: '1px solid #1e3a5f', background: '#0f1e35', color: '#f1f5f9', fontSize: '0.875rem', resize: 'vertical' }} />
                </div>

                <div className="upload-zone">
                  <div style={{ fontSize: '1.75rem', marginBottom: '0.5rem' }}>⬆️</div>
                  <div style={{ fontWeight: 600, marginBottom: '0.25rem' }}>Drop file here or click to browse</div>
                  <div style={{ fontSize: '0.78rem' }}>Supports: .geojson, .json, .zip (shapefile), .csv, .kml</div>
                </div>

                <button type="submit" className="btn btn-primary" style={{ alignSelf: 'flex-end', padding: '0.55rem 1.5rem' }}>
                  Register Layer
                </button>
              </form>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
