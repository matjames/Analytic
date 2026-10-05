import React, { useEffect, useState } from 'react';

const EMPTY = { title: '', archiveType: 'Dataset', repository: '', accessUrl: '', checksum: '', status: 'Planned', retentionUntil: '' };

export default function PreservationTab({ apiBase, researchId }) {
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(EMPTY);
  const [editing, setEditing] = useState(null);
  const [open, setOpen] = useState(false);

  const load = () => {
    if (!researchId) return setItems([]);
    fetch(`${apiBase}/api/research/${researchId}/archives`).then(r => r.json()).then(d => setItems(Array.isArray(d) ? d : [])).catch(() => setItems([]));
  };
  useEffect(load, [researchId]);
  const submit = (event) => {
    event.preventDefault();
    const method = editing ? 'PUT' : 'POST';
    const url = editing ? `${apiBase}/api/archives/${editing}` : `${apiBase}/api/archives`;
    fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...form, researchId }) }).then(r => { if (!r.ok) throw new Error('save failed'); setForm(EMPTY); setEditing(null); setOpen(false); load(); }).catch(() => {});
  };
  const edit = (item) => { setEditing(item.id); setForm({ title: item.title || '', archiveType: item.archiveType || 'Dataset', repository: item.repository || '', accessUrl: item.accessUrl || '', checksum: item.checksum || '', status: item.status || 'Planned', retentionUntil: item.retentionUntil || '' }); setOpen(true); };
  const remove = (id) => { if (window.confirm('Delete this preservation record?')) fetch(`${apiBase}/api/archives/${id}`, { method: 'DELETE' }).then(load); };

  return <div>
    <div className="section-heading"><div><h2>Preservation and Archive</h2><p>Record the repository, integrity checksum, retention policy, and access state of each research output.</p></div><button className="btn btn-primary" onClick={() => { setEditing(null); setForm(EMPTY); setOpen(v => !v); }}>{open ? 'Close' : 'Plan Archive'}</button></div>
    {open && <form onSubmit={submit} className="inline-form">
      {['title', 'repository', 'accessUrl', 'checksum', 'retentionUntil'].map(key => <label key={key}>{key.replace(/[A-Z]/g, m => ` ${m}`).replace(/^./, m => m.toUpperCase())}<input type={key === 'retentionUntil' ? 'date' : 'text'} value={form[key]} onChange={e => setForm({ ...form, [key]: e.target.value })} required={key === 'title'} /></label>)}
      <label>Archive type<select value={form.archiveType} onChange={e => setForm({ ...form, archiveType: e.target.value })}>{['Dataset', 'Publication', 'Code', 'Protocol', 'Records'].map(s => <option key={s}>{s}</option>)}</select></label>
      <label>Status<select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>{['Planned', 'Deposited', 'Verified', 'Embargoed', 'Expired'].map(s => <option key={s}>{s}</option>)}</select></label>
      <div><button className="btn btn-primary" type="submit">{editing ? 'Update Archive' : 'Save Archive'}</button></div>
    </form>}
    <div className="data-list">{items.length ? items.map(item => <article className="data-card" key={item.id}><div><strong>{item.title}</strong><p>{item.archiveType} | {item.repository || 'Repository not set'} | {item.status}</p><small>{item.checksum ? `Checksum: ${item.checksum}` : 'Integrity checksum not recorded'}</small></div><div><button className="btn btn-secondary" onClick={() => edit(item)}>Edit</button> <button className="btn btn-secondary" onClick={() => remove(item.id)}>Delete</button></div></article>) : <div className="empty-state">No preservation records recorded for this study.</div>}</div>
  </div>;
}
