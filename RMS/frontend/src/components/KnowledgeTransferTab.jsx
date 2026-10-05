import React, { useEffect, useState } from 'react';

const EMPTY = { title: '', summary: '', body: '', tags: '', status: 'Draft' };

export default function KnowledgeTransferTab({ apiBase, researchId }) {
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(EMPTY);
  const [editing, setEditing] = useState(null);
  const [open, setOpen] = useState(false);
  const load = () => { if (!researchId) return setItems([]); fetch(`${apiBase}/api/research/${researchId}/knowledge-transfers`).then(r => r.json()).then(d => setItems(Array.isArray(d) ? d : [])).catch(() => setItems([])); };
  useEffect(load, [researchId]);
  const submit = (e) => { e.preventDefault(); const method = editing ? 'PUT' : 'POST'; const url = editing ? `${apiBase}/api/knowledge-transfers/${editing}` : `${apiBase}/api/knowledge-transfers`; fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...form, researchId, tags: form.tags.split(',').map(v => v.trim()).filter(Boolean) }) }).then(r => { if (!r.ok) throw new Error('save failed'); setForm(EMPTY); setEditing(null); setOpen(false); load(); }).catch(() => {}); };
  const edit = (item) => { setEditing(item.id); setForm({ title: item.title || '', summary: item.summary || '', body: item.body || '', tags: (item.tags || []).join(', '), status: item.status || 'Draft' }); setOpen(true); };
  const publish = (id) => { fetch(`${apiBase}/api/knowledge-transfers/${id}/publish`, { method: 'POST' }).then(r => { if (!r.ok) throw new Error('publish failed'); load(); }).catch(() => {}); };
  const remove = (id) => { if (window.confirm('Delete this knowledge transfer?')) fetch(`${apiBase}/api/knowledge-transfers/${id}`, { method: 'DELETE' }).then(load); };
  return <div>
    <div className="section-heading"><div><h2>Knowledge Transfer</h2><p>Prepare findings for the shared Knowledge Portal. Publishing forwards the research identity and workspace context.</p></div><button className="btn btn-primary" onClick={() => { setEditing(null); setForm(EMPTY); setOpen(v => !v); }}>{open ? 'Close' : 'New Knowledge Item'}</button></div>
    {open && <form onSubmit={submit} className="inline-form">
      <label>Title<input value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} required /></label>
      <label>Tags<input value={form.tags} onChange={e => setForm({ ...form, tags: e.target.value })} placeholder="health, policy, evidence" /></label>
      <label>Summary<textarea value={form.summary} onChange={e => setForm({ ...form, summary: e.target.value })} /></label>
      <label>Finding or knowledge body<textarea value={form.body} onChange={e => setForm({ ...form, body: e.target.value })} required /></label>
      <label>Status<select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>{['Draft', 'Ready for Review'].map(s => <option key={s}>{s}</option>)}</select></label>
      <div><button className="btn btn-primary" type="submit">{editing ? 'Update Knowledge Item' : 'Save Knowledge Item'}</button></div>
    </form>}
    <div className="data-list">{items.length ? items.map(item => <article className="data-card" key={item.id}><div><strong>{item.title}</strong><p>{item.status} {item.knowledgeId ? `| Portal ID: ${item.knowledgeId}` : ''}</p><small>{item.summary || 'No summary recorded'}</small></div><div>{item.status !== 'Published' && <button className="btn btn-primary" onClick={() => publish(item.id)}>Publish</button>} <button className="btn btn-secondary" onClick={() => edit(item)}>Edit</button> <button className="btn btn-secondary" onClick={() => remove(item.id)}>Delete</button></div></article>) : <div className="empty-state">No knowledge-transfer items recorded for this study.</div>}</div>
  </div>;
}
