import React, { useEffect, useState } from 'react';

const EMPTY = { journal: '', manuscriptTitle: '', submissionDate: '', status: 'Draft', manuscriptUrl: '', correspondingAuthor: '', reviewerComments: '', nextAction: '' };

export default function SubmissionsTab({ apiBase, researchId }) {
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(EMPTY);
  const [editing, setEditing] = useState(null);
  const [open, setOpen] = useState(false);

  const load = () => {
    if (!researchId) return setItems([]);
    fetch(`${apiBase}/api/research/${researchId}/submissions`).then(r => r.json()).then(d => setItems(Array.isArray(d) ? d : [])).catch(() => setItems([]));
  };
  useEffect(load, [researchId]);

  const submit = (event) => {
    event.preventDefault();
    const method = editing ? 'PUT' : 'POST';
    const url = editing ? `${apiBase}/api/submissions/${editing}` : `${apiBase}/api/submissions`;
    fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...form, researchId }) })
      .then(r => { if (!r.ok) throw new Error('save failed'); setForm(EMPTY); setEditing(null); setOpen(false); load(); }).catch(() => {});
  };
  const edit = (item) => { setEditing(item.id); setForm({ journal: item.journal || '', manuscriptTitle: item.manuscriptTitle || '', submissionDate: item.submissionDate || '', status: item.status || 'Draft', manuscriptUrl: item.manuscriptUrl || '', correspondingAuthor: item.correspondingAuthor || '', reviewerComments: item.reviewerComments || '', nextAction: item.nextAction || '' }); setOpen(true); };
  const remove = (id) => { if (window.confirm('Delete this journal submission?')) fetch(`${apiBase}/api/submissions/${id}`, { method: 'DELETE' }).then(load); };

  return <div>
    <div className="section-heading"><div><h2>Journal Submissions</h2><p>Track manuscript submission, review, revision, and acceptance milestones.</p></div><button className="btn btn-primary" onClick={() => { setEditing(null); setForm(EMPTY); setOpen(v => !v); }}>{open ? 'Close' : 'New Submission'}</button></div>
    {open && <form onSubmit={submit} className="inline-form">
      {['journal', 'manuscriptTitle', 'submissionDate', 'manuscriptUrl', 'correspondingAuthor', 'nextAction'].map(key => <label key={key}>{key.replace(/[A-Z]/g, m => ` ${m}`).replace(/^./, m => m.toUpperCase())}<input type={key === 'submissionDate' ? 'date' : 'text'} value={form[key]} onChange={e => setForm({ ...form, [key]: e.target.value })} required={key === 'journal' || key === 'manuscriptTitle'} /></label>)}
      <label>Status<select value={form.status} onChange={e => setForm({ ...form, status: e.target.value })}>{['Draft', 'Submitted', 'Under Review', 'Revision Requested', 'Accepted', 'Rejected', 'Withdrawn'].map(s => <option key={s}>{s}</option>)}</select></label>
      <label>Reviewer comments<textarea value={form.reviewerComments} onChange={e => setForm({ ...form, reviewerComments: e.target.value })} /></label>
      <div><button className="btn btn-primary" type="submit">{editing ? 'Update Submission' : 'Save Submission'}</button></div>
    </form>}
    <div className="data-list">{items.length ? items.map(item => <article className="data-card" key={item.id}><div><strong>{item.manuscriptTitle}</strong><p>{item.journal} | {item.status} | {item.submissionDate || 'No submission date'}</p><small>{item.nextAction || 'No next action recorded'}</small></div><div><button className="btn btn-secondary" onClick={() => edit(item)}>Edit</button> <button className="btn btn-secondary" onClick={() => remove(item.id)}>Delete</button></div></article>) : <div className="empty-state">No journal submissions recorded for this study.</div>}</div>
  </div>;
}
