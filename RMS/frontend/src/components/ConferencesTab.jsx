import React, { useEffect, useState } from 'react';
import ConferenceEventsPanel from './ConferenceEventsPanel';

const EMPTY = { conferenceName: '', manuscriptTitle: '', submissionDate: '', status: 'Draft', presentationType: 'Oral', abstractUrl: '', reviewerComments: '', nextAction: '' };

export default function ConferencesTab({ apiBase, researchId }) {
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(EMPTY);
  const [editing, setEditing] = useState(null);
  const [open, setOpen] = useState(false);
  const load = () => { if (!researchId) return setItems([]); fetch(`${apiBase}/api/research/${researchId}/conferences`).then(r => r.json()).then(d => setItems(Array.isArray(d) ? d : [])).catch(() => setItems([])); };
  useEffect(load, [researchId]);
  const submit = (e) => { e.preventDefault(); const method = editing ? 'PUT' : 'POST'; const url = editing ? `${apiBase}/api/conferences/${editing}` : `${apiBase}/api/conferences`; fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...form, researchId }) }).then(r => { if (!r.ok) throw new Error('save failed'); setForm(EMPTY); setEditing(null); setOpen(false); load(); }).catch(() => {}); };
  const edit = (item) => { setEditing(item.id); setForm({ conferenceName: item.conferenceName || '', manuscriptTitle: item.manuscriptTitle || '', submissionDate: item.submissionDate || '', status: item.status || 'Draft', presentationType: item.presentationType || 'Oral', abstractUrl: item.abstractUrl || '', reviewerComments: item.reviewerComments || '', nextAction: item.nextAction || '' }); setOpen(true); };
  const remove = (id) => { if (window.confirm('Delete this conference submission?')) fetch(`${apiBase}/api/conferences/${id}`, { method: 'DELETE' }).then(load); };
  return <div>
    <div className="section-heading"><div><h2>Conference Submissions</h2><p>Track abstracts, presentation formats, peer review, and conference decisions.</p></div><button className="btn btn-primary" onClick={() => { setEditing(null); setForm(EMPTY); setOpen(v => !v); }}>{open ? 'Close' : 'New Submission'}</button></div>
    {open && <form onSubmit={submit} className="inline-form">
      {['conferenceName', 'manuscriptTitle', 'submissionDate', 'abstractUrl', 'nextAction'].map(key => <label key={key}>{key.replace(/[A-Z]/g, m => ` ${m}`).replace(/^./, m => m.toUpperCase())}<input type={key === 'submissionDate' ? 'date' : 'text'} value={form[key]} onChange={e => setForm({ ...form, [key]: e.target.value })} required={key === 'conferenceName' || key === 'manuscriptTitle'} /></label>)}
      <label>Presentation type<select value={form.presentationType} onChange={e => setForm({ ...form, presentationType: e.target.value })}>{['Oral', 'Poster', 'Workshop', 'Panel', 'Lightning Talk'].map(s => <option key={s}>{s}</option>)}</select></label>
      <label>Status<select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>{['Draft', 'Submitted', 'Under Review', 'Accepted', 'Rejected', 'Withdrawn'].map(s => <option key={s}>{s}</option>)}</select></label>
      <label>Reviewer comments<textarea value={form.reviewerComments} onChange={e => setForm({ ...form, reviewerComments: e.target.value })} /></label>
      <div><button className="btn btn-primary" type="submit">{editing ? 'Update Submission' : 'Save Submission'}</button></div>
    </form>}
    <div className="data-list">{items.length ? items.map(item => <article className="data-card" key={item.id}><div><strong>{item.manuscriptTitle}</strong><p>{item.conferenceName} | {item.presentationType} | {item.status}</p><small>{item.nextAction || 'No next action recorded'}</small></div><div><button className="btn btn-secondary" onClick={() => edit(item)}>Edit</button> <button className="btn btn-secondary" onClick={() => remove(item.id)}>Delete</button></div></article>) : <div className="empty-state">No conference submissions recorded for this study.</div>}</div>
    <ConferenceEventsPanel apiBase={apiBase} researchId={researchId} />
  </div>;
}
