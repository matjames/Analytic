import React, { useState } from 'react';

export default function CalendarTab({ events = [], meetings = [], milestones = [] }) {
  const [view, setView] = useState('list');

  // Combine all events
  const allEvents = [
    ...events.map(e => ({ ...e, type: e.eventType || 'Event' })),
    ...meetings.map(m => ({ id: m.id, title: m.title, description: m.agenda, type: 'Meeting', startTime: m.dateTime, endTime: m.dateTime, location: m.location, attendees: m.attendees })),
    ...milestones.map(m => ({ id: m.id, title: m.name, description: m.description, type: 'Milestone', startTime: m.dueDate, endTime: m.dueDate, location: '', attendees: [] })),
  ].sort((a, b) => new Date(a.startTime || a.start_date) - new Date(b.startTime || b.start_date));

  const getTypeColor = (type) => {
    const colors = {
      'Meeting': 'var(--accent-primary)',
      'Milestone': 'var(--accent-warning)',
      'Deadline': 'var(--accent-danger)',
      'Event': 'var(--accent-secondary)',
      'Task': 'var(--accent-success)',
    };
    return colors[type] || 'var(--accent-secondary)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📅 Project Calendar</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Meetings, milestones, deadlines, and scheduled events
          </p>
        </div>
        <div style={{ display: 'flex', gap: '6px' }}>
          <button className={`workspace-tab ${view === 'list' ? 'active' : ''}`} style={{ fontSize: '11px', padding: '6px 12px', margin: 0 }} onClick={() => setView('list')}>List</button>
          <button className={`workspace-tab ${view === 'upcoming' ? 'active' : ''}`} style={{ fontSize: '11px', padding: '6px 12px', margin: 0 }} onClick={() => setView('upcoming')}>Upcoming</button>
        </div>
      </div>

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Total Events', value: allEvents.length, icon: '📅', color: 'var(--accent-primary)' },
          { label: 'Meetings', value: meetings.length, icon: '🤝', color: 'var(--accent-secondary)' },
          { label: 'Milestones', value: milestones.length, icon: '🎯', color: 'var(--accent-warning)' },
          { label: 'Upcoming', value: allEvents.filter(e => new Date(e.startTime || e.start_date) > new Date()).length, icon: '⏳', color: 'var(--accent-success)' },
        ].map(stat => (
          <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>{stat.icon}</span>
            <div>
              <div style={{ fontSize: '22px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Events list */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Scheduled Events</h3>
        {allEvents.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No events scheduled.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {allEvents.map((ev, idx) => {
              const date = new Date(ev.startTime || ev.start_date);
              const isUpcoming = date > new Date();
              return (
                <div key={ev.id || idx} style={{ display: 'flex', gap: '14px', padding: '12px', background: 'rgba(255,255,255,0.02)', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.06)' }}>
                  <div style={{ width: '48px', textAlign: 'center', flexShrink: 0 }}>
                    <div style={{ fontSize: '18px', fontWeight: '800', color: getTypeColor(ev.type) }}>{date.getDate()}</div>
                    <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>
                      {date.toLocaleDateString('en-GB', { month: 'short' })}
                    </div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                      <span style={{ fontSize: '13px', fontWeight: '700' }}>{ev.title}</span>
                      <span className="badge" style={{ fontSize: '9px', background: `${getTypeColor(ev.type)}20`, color: getTypeColor(ev.type) }}>{ev.type}</span>
                    </div>
                    {ev.description && <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{ev.description}</div>}
                    <div style={{ display: 'flex', gap: '12px', fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>
                      <span>🕐 {date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                      {ev.location && <span>📍 {ev.location}</span>}
                      {ev.attendees && ev.attendees.length > 0 && <span>👥 {ev.attendees.length} attendees</span>}
                      {isUpcoming && <span style={{ color: 'var(--accent-success)' }}>● Upcoming</span>}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}