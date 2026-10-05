import React, { useEffect, useState } from 'react';

const emptyForm = { activityId: '', label: '', worker: '', status: 'Recorded', latitude: '', longitude: '', recordedAt: '' };

export default function FieldActivityMap({ projectId, activities = [], apiBase }) {
  const [locations, setLocations] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [isOpen, setIsOpen] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    if (!projectId) return;
    fetch(`${apiBase}/api/projects/${projectId}/field-activities`)
      .then(response => response.ok ? response.json() : Promise.reject(new Error('Field activity locations could not be loaded.')))
      .then(data => setLocations(Array.isArray(data) ? data : []))
      .catch(err => setError(err.message));
  };

  useEffect(load, [projectId, apiBase]);

  const update = (key, value) => setForm(current => ({ ...current, [key]: value }));

  const save = async event => {
    event.preventDefault();
    setError('');
    try {
      const latitude = Number(form.latitude);
      const longitude = Number(form.longitude);
      const body = {
        ...form,
        projectId,
        latitude,
        longitude,
        recordedAt: form.recordedAt ? new Date(form.recordedAt).toISOString() : '',
      };
      const response = await fetch(`${apiBase}/api/field-activities`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Field activity location could not be saved.');
      setForm(emptyForm);
      setIsOpen(false);
      load();
    } catch (err) {
      setError(err.message || 'Field activity location could not be saved.');
    }
  };

  const remove = async id => {
    const response = await fetch(`${apiBase}/api/field-activities/${id}`, { method: 'DELETE' });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      setError(data.error || 'Field activity location could not be deleted.');
      return;
    }
    load();
  };

  const first = locations[0];
  const mapUrl = first && `https://www.openstreetmap.org/export/embed.html?bbox=${first.longitude - 0.04}%2C${first.latitude - 0.04}%2C${first.longitude + 0.04}%2C${first.latitude + 0.04}&layer=mapnik&marker=${first.latitude}%2C${first.longitude}`;

  return (
    <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px' }}>
        <div>
          <h3 style={{ fontSize: '16px', fontWeight: '800', marginBottom: '4px' }}>Field Activity Map</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>Workspace-owned field observations and GPS points linked to project activities.</p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 12px' }} onClick={() => { setError(''); setIsOpen(true); }}>+ Record point</button>
      </div>

      {error && <div className="form-error" role="alert" style={{ marginTop: '12px' }}>{error}</div>}

      {isOpen && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={save}>
            <h3>Record Field Activity Point</h3>
            <div className="form-group"><label>Activity</label><select value={form.activityId} onChange={event => update('activityId', event.target.value)}><option value="">Unlinked observation</option>{activities.map(activity => <option key={activity.id} value={activity.id}>{activity.name}</option>)}</select></div>
            <div className="form-group"><label>Point label</label><input value={form.label} onChange={event => update('label', event.target.value)} placeholder="e.g. Household cluster A" required /></div>
            <div className="form-row"><div className="form-group"><label>Latitude</label><input type="number" step="any" value={form.latitude} onChange={event => update('latitude', event.target.value)} required /></div><div className="form-group"><label>Longitude</label><input type="number" step="any" value={form.longitude} onChange={event => update('longitude', event.target.value)} required /></div></div>
            <div className="form-row"><div className="form-group"><label>Worker / team</label><input value={form.worker} onChange={event => update('worker', event.target.value)} /></div><div className="form-group"><label>Status</label><select value={form.status} onChange={event => update('status', event.target.value)}><option value="Recorded">Recorded</option><option value="In Progress">In Progress</option><option value="Verified">Verified</option><option value="Flagged">Flagged</option></select></div><div className="form-group"><label>Recorded at</label><input type="datetime-local" value={form.recordedAt} onChange={event => update('recordedAt', event.target.value)} /></div></div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}><button type="button" className="btn btn-secondary" onClick={() => setIsOpen(false)}>Cancel</button><button type="submit" className="btn btn-primary">Save point</button></div>
          </form>
        </div>
      )}

      {first ? <iframe title="Field activity map" src={mapUrl} loading="lazy" style={{ width: '100%', height: '220px', border: '1px solid var(--border-light)', borderRadius: '8px', marginTop: '16px' }} /> : <div style={{ padding: '28px 0 10px', textAlign: 'center', fontSize: '12px', color: 'var(--text-muted)' }}>No field points recorded yet.</div>}

      {locations.length > 0 && <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '14px' }}>{locations.map(location => <div key={location.id} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', gap: '12px', alignItems: 'center', padding: '9px 10px', border: '1px solid var(--border-light)', borderRadius: '6px' }}><div><div style={{ fontSize: '12px', fontWeight: '700' }}>{location.label}</div><div style={{ fontSize: '10px', color: 'var(--text-muted)' }}>{location.latitude}, {location.longitude} {location.worker ? `| ${location.worker}` : ''}</div></div><span className="badge">{location.status}</span><button className="btn btn-secondary" style={{ padding: '4px 8px', fontSize: '10px' }} onClick={() => remove(location.id)}>Delete</button></div>)}</div>}
    </div>
  );
}
