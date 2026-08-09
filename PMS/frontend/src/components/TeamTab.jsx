import React, { useState } from 'react';

const ROLE_COLORS = {
  'Project Manager':    { bg: 'rgba(99,102,241,0.15)', color: '#818cf8' },
  'Coordinator':        { bg: 'rgba(6,182,212,0.15)',  color: '#22d3ee' },
  'Field Supervisor':   { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
  'Enumerator':         { bg: 'rgba(245,158,11,0.15)', color: '#fbbf24' },
  'Finance Officer':    { bg: 'rgba(239,68,68,0.15)',  color: '#f87171' },
  'M&E Officer':        { bg: 'rgba(168,85,247,0.15)', color: '#c084fc' },
  'Research Lead':      { bg: 'rgba(251,146,60,0.15)', color: '#fb923c' },
  'Consultant':         { bg: 'rgba(148,163,184,0.15)', color: '#94a3b8' },
  'Team Lead':          { bg: 'rgba(6,182,212,0.15)',  color: '#22d3ee' },
};

const FALLBACK_MEMBERS = [
  { id: 'm1', name: 'Dr. Sarah Jenkins', role: 'Project Manager', department: 'Project Leadership', email: 's.jenkins@statgate.ug', phone: '+256 701 234 001', location: 'Kampala HQ', since: '2026-02-15', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=sarah' },
  { id: 'm2', name: 'Marcus Vance', role: 'Coordinator', department: 'Operations', email: 'm.vance@statgate.ug', phone: '+256 701 234 002', location: 'Kampala HQ', since: '2026-02-15', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=marcus' },
  { id: 'm3', name: 'Alice Ouko', role: 'Field Supervisor', department: 'Field Operations', email: 'a.ouko@statgate.ug', phone: '+256 701 234 003', location: 'Eastern Zone', since: '2026-03-01', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=alice' },
  { id: 'm4', name: 'Robert Amoko', role: 'M&E Officer', department: 'Monitoring & Evaluation', email: 'r.amoko@statgate.ug', phone: '+256 701 234 004', location: 'Kampala HQ', since: '2026-02-20', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=robert' },
  { id: 'm5', name: 'Grace Nakato', role: 'Finance Officer', department: 'Finance', email: 'g.nakato@statgate.ug', phone: '+256 701 234 005', location: 'Kampala HQ', since: '2026-02-15', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=grace' },
  { id: 'm6', name: 'Dr. James Okello', role: 'Research Lead', department: 'Data Science', email: 'j.okello@statgate.ug', phone: '+256 701 234 006', location: 'Kampala HQ', since: '2026-03-10', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=james' },
  { id: 'm7', name: 'Faith Aciro', role: 'Field Supervisor', department: 'Field Operations', email: 'f.aciro@statgate.ug', phone: '+256 701 234 007', location: 'Northern Zone', since: '2026-03-01', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=faith' },
  { id: 'm8', name: 'Emmanuel Onen', role: 'Enumerator', department: 'Field Operations', email: 'e.onen@statgate.ug', phone: '+256 701 234 008', location: 'Northern Zone', since: '2026-04-01', avatarUrl: 'https://api.dicebear.com/7.x/adventurer/svg?seed=emmanuel' },
];

function MemberCard({ member }) {
  const roleStyle = ROLE_COLORS[member.role] || { bg: 'rgba(99,102,241,0.1)', color: '#818cf8' };

  return (
    <div
      className="glass-panel"
      style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px', transition: 'all 0.2s' }}
      onMouseEnter={e => e.currentTarget.style.borderColor = 'rgba(99,102,241,0.3)'}
      onMouseLeave={e => e.currentTarget.style.borderColor = 'rgba(255,255,255,0.08)'}
    >
      {/* Avatar + name */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <img
          src={member.avatarUrl || `https://api.dicebear.com/7.x/adventurer/svg?seed=${encodeURIComponent(member.name)}`}
          alt={member.name}
          style={{ width: '48px', height: '48px', borderRadius: '50%', background: 'rgba(255,255,255,0.06)', flexShrink: 0 }}
        />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '14px', fontWeight: '700', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{member.name}</div>
          <span className="badge" style={{ background: roleStyle.bg, color: roleStyle.color, fontSize: '10px', marginTop: '3px', display: 'inline-block' }}>
            {member.role}
          </span>
        </div>
      </div>

      {/* Details */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '5px', fontSize: '11px', color: 'var(--text-secondary)', borderTop: '1px solid var(--border-glass)', paddingTop: '10px' }}>
        {member.department && <span>🏢 {member.department}</span>}
        {member.email && <span>📧 {member.email}</span>}
        {member.phone && <span>📞 {member.phone}</span>}
        {member.location && <span>📍 {member.location}</span>}
        {member.since && <span>📅 Since {new Date(member.since).toLocaleDateString('en-GB', { month: 'short', year: 'numeric' })}</span>}
      </div>

      {/* Actions */}
      <div style={{ display: 'flex', gap: '6px' }}>
        <button className="btn btn-secondary" style={{ flex: 1, padding: '5px 8px', fontSize: '11px' }} title="Send message">💬 Message</button>
        <button className="btn btn-secondary" style={{ flex: 1, padding: '5px 8px', fontSize: '11px' }} title="View profile">👤 Profile</button>
      </div>
    </div>
  );
}

export default function TeamTab({ members = [] }) {
  const teamMembers = members.length > 0 ? members : FALLBACK_MEMBERS;
  const [search, setSearch] = useState('');
  const [activeRole, setActiveRole] = useState('All');

  const roles = ['All', ...new Set(teamMembers.map(m => m.role))];

  const filtered = teamMembers.filter(m => {
    const matchSearch = m.name.toLowerCase().includes(search.toLowerCase()) ||
      m.role.toLowerCase().includes(search.toLowerCase()) ||
      (m.department || '').toLowerCase().includes(search.toLowerCase());
    const matchRole = activeRole === 'All' || m.role === activeRole;
    return matchSearch && matchRole;
  });

  // Role distribution summary
  const roleCounts = {};
  teamMembers.forEach(m => { roleCounts[m.role] = (roleCounts[m.role] || 0) + 1; });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>👥 Team & Staff Register</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            All assigned personnel, roles, departments, and contact details from the Staff Register
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }}>
          + Assign Member
        </button>
      </div>

      {/* Stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))', gap: '12px' }}>
        <div className="glass-panel" style={{ padding: '14px', textAlign: 'center' }}>
          <div style={{ fontSize: '26px', fontWeight: '800', color: 'var(--accent-primary)' }}>{teamMembers.length}</div>
          <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Total Members</div>
        </div>
        {Object.entries(roleCounts).slice(0, 5).map(([role, count]) => {
          const roleStyle = ROLE_COLORS[role] || { bg: 'rgba(99,102,241,0.1)', color: '#818cf8' };
          return (
            <div key={role} className="glass-panel" style={{ padding: '14px', textAlign: 'center', background: roleStyle.bg, border: `1px solid ${roleStyle.color}20` }}>
              <div style={{ fontSize: '22px', fontWeight: '800', color: roleStyle.color }}>{count}</div>
              <div style={{ fontSize: '9px', color: roleStyle.color, textTransform: 'uppercase', marginTop: '2px' }}>{role}</div>
            </div>
          );
        })}
      </div>

      {/* Search + Role filter */}
      <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', alignItems: 'center' }}>
        <div style={{ position: 'relative', flex: 1, minWidth: '200px' }}>
          <span style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', fontSize: '14px', pointerEvents: 'none' }}>🔍</span>
          <input
            type="text"
            placeholder="Search by name, role, department..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ width: '100%', paddingLeft: '36px', paddingRight: '12px', paddingTop: '8px', paddingBottom: '8px' }}
          />
        </div>
        <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
          {roles.map(r => (
            <button
              key={r}
              onClick={() => setActiveRole(r)}
              className={`workspace-tab ${activeRole === r ? 'active' : ''}`}
              style={{ fontSize: '11px', padding: '5px 12px', margin: 0 }}
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      {/* Member grid */}
      {filtered.length > 0 ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(270px, 1fr))', gap: '16px' }}>
          {filtered.map(m => <MemberCard key={m.id || m.name} member={m} />)}
        </div>
      ) : (
        <div className="glass-panel" style={{ padding: '48px', textAlign: 'center', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>👥</div>
          <div style={{ fontSize: '16px', fontWeight: '700', marginBottom: '4px' }}>No team members found</div>
          <div style={{ fontSize: '13px' }}>Try adjusting your search or assign team members to this project.</div>
        </div>
      )}

      {/* Staff Register link */}
      <div className="glass-panel" style={{ padding: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <div style={{ fontSize: '13px', fontWeight: '700' }}>🏥 Staff Register Integration</div>
          <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>Member profiles are synced from the StatGate Field Operations Registry</div>
        </div>
        <a href="http://localhost:3000" target="_blank" rel="noopener noreferrer" className="btn btn-secondary" style={{ fontSize: '11px', padding: '7px 14px' }}>
          Open Staff Register →
        </a>
      </div>
    </div>
  );
}
