import React from 'react';

export default function CalendarTab({ events, researchId, onRefresh }) {
  return (
    <div>
      <div style={{ marginBottom: '20px' }}>
        <h3>Research Calendar</h3>
        <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginTop: '4px' }}>
          Milestones, deadlines, meetings, and key dates
        </p>
      </div>

      <div className="glass-panel" style={{ padding: '24px' }}>
        {events.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No calendar events scheduled.</p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {events.map(ev => (
              <div key={ev.id} style={{ display: 'flex', gap: '16px', padding: '16px', background: 'var(--bg-light)', borderRadius: '8px', borderLeft: `4px solid ${ev.eventType === 'Milestone' ? 'var(--primary)' : ev.eventType === 'Deadline' ? 'var(--danger)' : 'var(--accent)'}` }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0', fontSize: '15px' }}>{ev.title}</h4>
                  {ev.description && <p style={{ fontSize: '13px', margin: '4px 0', color: 'var(--text-secondary)' }}>{ev.description}</p>}
                  <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap', fontSize: '12px', color: 'var(--text-secondary)' }}>
                    <span>📅 {ev.startTime}</span>
                    {ev.endTime && <span>→ {ev.endTime}</span>}
                    {ev.location && <span>📍 {ev.location}</span>}
                    <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px', fontSize: '11px' }}>{ev.eventType}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}