import React, { useState } from 'react';

const DOC_TYPES = {
  Contract:          { icon: '📜', color: '#818cf8' },
  Proposal:          { icon: '📄', color: '#22d3ee' },
  Budget:            { icon: '💰', color: '#34d399' },
  'Technical Report':{ icon: '📊', color: '#f59e0b' },
  Presentation:      { icon: '📽️', color: '#a78bfa' },
  'Field Manual':    { icon: '📖', color: '#fb923c' },
  'Meeting Minutes': { icon: '📋', color: '#94a3b8' },
  Policy:            { icon: '🏛️', color: '#ef4444' },
  Dataset:           { icon: '🗄️', color: '#06b6d4' },
  Other:             { icon: '📎', color: '#6b7280' },
};

// Synthetic document list when backend data is empty
const FALLBACK_DOCS = [
  { id: 'd1', name: 'Project Charter & Terms of Reference', type: 'Contract', uploadedBy: 'Dr. Sarah Jenkins', uploadedAt: '2026-02-15', size: '2.4 MB', status: 'Approved' },
  { id: 'd2', name: 'Initial Concept Note & Feasibility Study', type: 'Proposal', uploadedBy: 'Marcus Vance', uploadedAt: '2026-01-10', size: '1.1 MB', status: 'Approved' },
  { id: 'd3', name: 'Approved Budget Framework FY2026', type: 'Budget', uploadedBy: 'Finance Office', uploadedAt: '2026-02-20', size: '890 KB', status: 'Approved' },
  { id: 'd4', name: 'Baseline Assessment Technical Report', type: 'Technical Report', uploadedBy: 'Dr. Sarah Jenkins', uploadedAt: '2026-03-05', size: '5.6 MB', status: 'Final' },
  { id: 'd5', name: 'Field Enumerator Training Manual v2.1', type: 'Field Manual', uploadedBy: 'Alice Ouko', uploadedAt: '2026-04-18', size: '3.2 MB', status: 'Active' },
  { id: 'd6', name: 'Mid-Term Review Presentation', type: 'Presentation', uploadedBy: 'Dr. Sarah Jenkins', uploadedAt: '2026-06-30', size: '12.4 MB', status: 'Final' },
  { id: 'd7', name: 'Data Collection Protocol v1.3', type: 'Policy', uploadedBy: 'Robert Amoko', uploadedAt: '2026-03-22', size: '760 KB', status: 'Active' },
  { id: 'd8', name: 'Stakeholder Meeting Minutes – March 2026', type: 'Meeting Minutes', uploadedBy: 'Alice Ouko', uploadedAt: '2026-03-15', size: '320 KB', status: 'Final' },
  { id: 'd9', name: 'Approved Household Survey Dataset', type: 'Dataset', uploadedBy: 'StatCollect System', uploadedAt: '2026-07-01', size: '28.7 MB', status: 'Approved' },
];

const STATUS_BADGE = {
  Approved: { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
  Final:    { bg: 'rgba(6,182,212,0.15)', color: '#22d3ee' },
  Draft:    { bg: 'rgba(245,158,11,0.15)', color: '#fbbf24' },
  Active:   { bg: 'rgba(99,102,241,0.15)', color: '#818cf8' },
};

function DocCard({ doc }) {
  const meta = DOC_TYPES[doc.type] || DOC_TYPES.Other;
  const statusStyle = STATUS_BADGE[doc.status] || STATUS_BADGE.Draft;

  return (
    <div className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '14px', transition: 'all 0.2s', cursor: 'default' }}
      onMouseEnter={e => e.currentTarget.style.borderColor = 'rgba(99,102,241,0.3)'}
      onMouseLeave={e => e.currentTarget.style.borderColor = 'rgba(255,255,255,0.08)'}
    >
      {/* Type icon */}
      <div style={{ width: '44px', height: '44px', borderRadius: '10px', background: `${meta.color}20`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '20px', flexShrink: 0 }}>
        {meta.icon}
      </div>

      {/* Info */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: '13px', fontWeight: '700', marginBottom: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{doc.name}</div>
        <div style={{ display: 'flex', gap: '10px', fontSize: '11px', color: 'var(--text-muted)', flexWrap: 'wrap' }}>
          <span style={{ color: meta.color }}>{doc.type}</span>
          <span>👤 {doc.uploadedBy}</span>
          <span>📅 {doc.uploadedAt}</span>
          <span>📦 {doc.size}</span>
        </div>
      </div>

      {/* Status badge */}
      <span className="badge" style={{ background: statusStyle.bg, color: statusStyle.color, fontSize: '10px', whiteSpace: 'nowrap' }}>{doc.status}</span>

      {/* Actions */}
      <div style={{ display: 'flex', gap: '6px' }}>
        <button className="btn btn-secondary" style={{ padding: '4px 10px', fontSize: '11px' }} title="View document">👁 View</button>
        <button className="btn btn-secondary" style={{ padding: '4px 10px', fontSize: '11px' }} title="Download">⬇ Download</button>
      </div>
    </div>
  );
}

export default function DocumentsTab({ documents = [] }) {
  const docs = documents.length > 0 ? documents : FALLBACK_DOCS;
  const [activeType, setActiveType] = useState('All');
  const [search, setSearch] = useState('');

  const types = ['All', ...Object.keys(DOC_TYPES)].filter(t =>
    t === 'All' || docs.some(d => d.type === t)
  );

  const filtered = docs.filter(d => {
    const matchType = activeType === 'All' || d.type === activeType;
    const matchSearch = d.name.toLowerCase().includes(search.toLowerCase()) || d.type.toLowerCase().includes(search.toLowerCase());
    return matchType && matchSearch;
  });

  // Count by type
  const typeCounts = {};
  docs.forEach(d => { typeCounts[d.type] = (typeCounts[d.type] || 0) + 1; });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📁 Document Management</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Contracts, reports, manuals, budgets, and official project documents
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }}>
          + Upload Document
        </button>
      </div>

      {/* Type summary cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: '12px' }}>
        {Object.entries(typeCounts).map(([type, count]) => {
          const meta = DOC_TYPES[type] || DOC_TYPES.Other;
          return (
            <button
              key={type}
              onClick={() => setActiveType(activeType === type ? 'All' : type)}
              className="glass-panel"
              style={{
                padding: '14px', textAlign: 'left', cursor: 'pointer', border: activeType === type ? `1px solid ${meta.color}` : '1px solid rgba(255,255,255,0.08)',
                transition: 'all 0.2s', background: activeType === type ? `${meta.color}15` : undefined,
              }}
            >
              <div style={{ fontSize: '20px', marginBottom: '6px' }}>{meta.icon}</div>
              <div style={{ fontSize: '18px', fontWeight: '800', color: meta.color }}>{count}</div>
              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase', marginTop: '2px' }}>{type}</div>
            </button>
          );
        })}
      </div>

      {/* Search + filter bar */}
      <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ position: 'relative', flex: 1, minWidth: '200px' }}>
          <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', fontSize: '14px', pointerEvents: 'none' }}>🔍</span>
          <input
            type="text"
            placeholder="Search documents..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ width: '100%', paddingLeft: '36px', paddingRight: '12px', paddingTop: '8px', paddingBottom: '8px' }}
          />
        </div>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {types.slice(0, 5).map(t => (
            <button
              key={t}
              onClick={() => setActiveType(t)}
              className={`workspace-tab ${activeType === t ? 'active' : ''}`}
              style={{ fontSize: '11px', padding: '5px 12px', margin: 0 }}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {/* Document list */}
      {filtered.length > 0 ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginBottom: '4px' }}>
            {filtered.length} document{filtered.length !== 1 ? 's' : ''} {activeType !== 'All' ? `in ${activeType}` : ''}
          </div>
          {filtered.map(doc => <DocCard key={doc.id} doc={doc} />)}
        </div>
      ) : (
        <div className="glass-panel" style={{ padding: '48px', textAlign: 'center', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>📁</div>
          <div style={{ fontSize: '16px', fontWeight: '700', marginBottom: '4px' }}>No documents found</div>
          <div style={{ fontSize: '13px' }}>Try adjusting your search or upload the first document.</div>
        </div>
      )}
    </div>
  );
}
