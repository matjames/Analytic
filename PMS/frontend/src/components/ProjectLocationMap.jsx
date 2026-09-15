import React, { useEffect, useState } from 'react';

const authHeaders = () => {
  const token = localStorage.getItem('registry_jwt') || localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
};

export default function ProjectLocationMap({ project, apiBase }) {
  const [location, setLocation] = useState(null);
  const [lat, setLat] = useState('');
  const [lng, setLng] = useState('');
  const [adminUnit, setAdminUnit] = useState('au-uganda');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    setLoading(true);
    fetch(`${apiBase}/api/spatial/project-locations?project_id=${encodeURIComponent(project.id)}`, { headers: authHeaders() })
      .then(response => {
        if (!response.ok) throw new Error(`Location request failed: ${response.status}`);
        return response.json();
      })
      .then(items => {
        const item = Array.isArray(items) ? items[0] : null;
        setLocation(item);
        setLat(item?.lat ?? '');
        setLng(item?.lng ?? '');
        setAdminUnit(item?.admin_unit_id || 'au-uganda');
      })
      .catch(() => setMessage('Location service is unavailable.'))
      .finally(() => setLoading(false));
  }, [project.id, apiBase]);

  const saveLocation = event => {
    event.preventDefault();
    const numericLat = Number(lat);
    const numericLng = Number(lng);
    if (!Number.isFinite(numericLat) || !Number.isFinite(numericLng) || numericLat < -90 || numericLat > 90 || numericLng < -180 || numericLng > 180) {
      setMessage('Enter a valid latitude (-90 to 90) and longitude (-180 to 180).');
      return;
    }
    setSaving(true);
    setMessage('');
    fetch(`${apiBase}/api/spatial/project-locations`, {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ target_id: project.id, admin_unit_id: adminUnit.trim(), lat: numericLat, lng: numericLng }),
    })
      .then(response => {
        if (!response.ok) throw new Error(`Location save failed: ${response.status}`);
        return response.json();
      })
      .then(saved => {
        setLocation({ target_id: saved.target_id, admin_unit_id: adminUnit, lat: saved.lat, lng: saved.lng });
        setMessage('Project location saved to StatSpatial.');
      })
      .catch(error => setMessage(error.message))
      .finally(() => setSaving(false));
  };

  const mapUrl = location && `https://www.openstreetmap.org/export/embed.html?bbox=${location.lng - 0.04}%2C${location.lat - 0.04}%2C${location.lng + 0.04}%2C${location.lat + 0.04}&layer=mapnik&marker=${location.lat}%2C${location.lng}`;
  return (
    <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px', flexWrap: 'wrap' }}>
        <div>
          <h3 style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>GIS Project Location</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{project.targetGeo || 'Add a geographic target and coordinate for this project.'}</p>
        </div>
        {location && <a href={`https://www.openstreetmap.org/?mlat=${location.lat}&mlon=${location.lng}#map=10/${location.lat}/${location.lng}`} target="_blank" rel="noreferrer" style={{ fontSize: '12px', color: 'var(--primary-color)' }}>Open full map</a>}
      </div>

      <form onSubmit={saveLocation} style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1.2fr auto', gap: '10px', alignItems: 'end', marginTop: '16px' }}>
        <label className="form-group"><span>Latitude</span><input type="number" step="any" value={lat} onChange={event => setLat(event.target.value)} placeholder="0.3476" required /></label>
        <label className="form-group"><span>Longitude</span><input type="number" step="any" value={lng} onChange={event => setLng(event.target.value)} placeholder="32.5825" required /></label>
        <label className="form-group"><span>Admin unit ID</span><input type="text" value={adminUnit} onChange={event => setAdminUnit(event.target.value)} placeholder="au-uganda" required /></label>
        <button className="btn btn-primary" type="submit" disabled={saving || loading}>{saving ? 'Saving...' : 'Save location'}</button>
      </form>

      {message && <p style={{ marginTop: '10px', fontSize: '12px', color: message.includes('saved') ? 'var(--success-color)' : 'var(--accent-danger)' }}>{message}</p>}
      {loading ? <p style={{ marginTop: '16px', fontSize: '12px', color: 'var(--text-muted)' }}>Loading spatial location...</p> : location && (
        <iframe title={`Map location for ${project.name}`} src={mapUrl} loading="lazy" style={{ width: '100%', height: '220px', border: '1px solid var(--border-light)', borderRadius: '8px', marginTop: '16px' }} />
      )}
    </div>
  );
}
