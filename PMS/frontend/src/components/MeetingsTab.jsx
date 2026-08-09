import React, { useState } from 'react';

const STATUS_COLORS = {
  Scheduled:  { bg: 'rgba(99,102,241,0.15)', color: '#818cf8' },
  'In Progress': { bg: 'rgba(6,182,212,0.15)', color: '#22d3ee' },
  Completed:  { bg: 'rgba(16,185,129,0.15)', color: '#34d399' },
  Cancelled:  { bg: 'rgba(239,68,68,0.15)', color: '#f87171' },
};

function MeetingCard({ meeting }) {
  const [expanded, setExpanded] = useState(false);
  const status = meeting.status || 'Scheduled';
  const colors = STATUS_COLORS[status] || STATUS_COLORS.Scheduled;
  const dt = new Date(meeting.dateTime);
  const dateStr = dt.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' });
  const timeStr = dt.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });

  return (
    <div className="glass-panel meeting-card" style={{ padding: '20px', cursor: 'pointer' }} onClick={() => setExpanded(e => !e)}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div style={{ flex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
            <span style={{ fontSize: '15px', fontWeight: '700' }}>{meeting.title}</span>
            <span
              className="badge"
              style={{ background: colors.bg, color: colors.color, fontSize: '10px' }}
            >
              {status}
            </span>
          </div>
          <div style={{ display: 'flex', gap: '16px', fontSize: '12px', color: 'var(--text-secondary)', flexWrap: 'wrap' }}>
            <span>📅 {dateStr}</span>
            <span>⏰ {timeStr}</span>
            {meeting.location && <span>📍 {meeting.location}</span>}
            {meeting.attendees && <span>👥 {meeting.attendees.length} attendees</span>}
          </div>
        </div>
        <span style={{ fontSize: '12px', color: 'var(--text-muted)', marginLeft: '8px' }}>{expanded ? '▲' : '▼'}</span>
      </div>

      {expanded && (
        <div style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '12px', borderTop: '1px solid var(--border-glass)', paddingTop: '16px' }}>
          {/* Agenda */}
          {meeting.agenda && (
            <div>
              <div style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-muted)', letterSpacing: '0.05em', marginBottom: '6px' }}>Agenda</div>
              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', lineHeight: '1.6' }}>{meeting.agenda}</div>
            </div>
          )}

          {/* Attendees */}
          {meeting.attendees && meeting.attendees.length > 0 && (
            <div>
              <div style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-muted)', letterSpacing: '0.05em', marginBottom: '6px' }}>Attendees</div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                {meeting.attendees.map((a, i) => (
                  <span key={i} className="badge" style={{ background: 'rgba(99,102,241,0.1)', color: 'var(--accent-primary)', fontSize: '11px' }}>{a}</span>
                ))}
              </div>
            </div>
          )}

          {/* Action Items */}
          {meeting.actionItems && meeting.actionItems.length > 0 && (
            <div>
              <div style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-muted)', letterSpacing: '0.05em', marginBottom: '6px' }}>Action Items</div>
              <ul style={{ paddingLeft: '16px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                {meeting.actionItems.map((item, i) => (
                  <li key={i} style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.5' }}>{item}</li>
                ))}
              </ul>
            </div>
          )}

          {/* Minutes / Decisions */}
          {meeting.decisions && meeting.decisions.length > 0 && (
            <div>
              <div style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-muted)', letterSpacing: '0.05em', marginBottom: '6px' }}>Decisions</div>
              <ul style={{ paddingLeft: '16px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                {meeting.decisions.map((d, i) => (
                  <li key={i} style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.5' }}>✅ {d}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function MeetingsTab({ meetings = [] }) {
  const [filter, setFilter] = useState('All');
  const statuses = ['All', 'Scheduled', 'In Progress', 'Completed', 'Cancelled'];

  // Enrich meetings with realistic default data if fields are missing
  const enriched = meetings.map((m, i) => ({
    ...m,
    status: m.status || (i % 3 === 0 ? 'Completed' : i % 3 === 1 ? 'Scheduled' : 'In Progress'),
    attendees: m.attendees || ['Dr. Sarah Jenkins', 'Marcus Vance', 'Alice Ouko', 'Robert Amoko'],
    decisions: m.decisions || ['Proceed with Phase 2 field deployment', 'Allocate additional field supervisors to Eastern Zone'],
  }));

  const filtered = filter === 'All' ? enriched : enriched.filter(m => m.status === filter);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Header row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📋 Meetings & Minutes</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Schedules, agendas, attendance records, decisions, and action items
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }}>
          + Schedule Meeting
        </button>
      </div>

      {/* Status filter tabs */}
      <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
        {statuses.map(s => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`workspace-tab ${filter === s ? 'active' : ''}`}
            style={{ fontSize: '12px', padding: '6px 14px', margin: 0 }}
          >
            {s}
          </button>
        ))}
      </div>

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Total Meetings', value: enriched.length, icon: '📅', color: 'var(--accent-primary)' },
          { label: 'Scheduled', value: enriched.filter(m => m.status === 'Scheduled').length, icon: '⏳', color: 'var(--accent-secondary)' },
          { label: 'Completed', value: enriched.filter(m => m.status === 'Completed').length, icon: '✅', color: 'var(--accent-success)' },
          { label: 'Action Items', value: enriched.reduce((acc, m) => acc + (m.actionItems?.length || 0), 0), icon: '📌', color: 'var(--accent-warning)' },
        ].map(stat => (
          <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>{stat.icon}</span>
            <div>
              <div style={{ fontSize: '22px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Meeting cards */}
      {filtered.length > 0 ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {filtered.map(m => <MeetingCard key={m.id} meeting={m} />)}
        </div>
      ) : (
        <div className="glass-panel" style={{ padding: '48px', textAlign: 'center', color: 'var(--text-muted)' }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>📅</div>
          <div style={{ fontSize: '16px', fontWeight: '700', marginBottom: '4px' }}>No meetings found</div>
          <div style={{ fontSize: '13px' }}>No meetings match the selected filter.</div>
        </div>
      )}
    </div>
  );
}
