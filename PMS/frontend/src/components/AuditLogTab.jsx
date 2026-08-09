import React, { useState } from 'react';

export default function AuditLogTab({ logs = [] }) {
  const [filter, setFilter] = useState('All');

  const actions = ['All', ...new Set(logs.map(l => l.action))];

  const filtered = filter === 'All' ? logs : logs.filter(l => l.action === filter);

  const getActionColor = (action) => {
    const colors = {
      'CREATE': 'var(--accent-success)',
      'UPDATE': 'var(--accent-secondary)',
      'DELETE': 'var(--accent-danger)',
      'STAGE_CHANGE': 'var(--accent-warning)',
      'APPROVE': 'var(--accent-primary)',
    };
    return colors[action] || 'var(--text-secondary)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div>
        <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📜 Audit Log</h2>
        <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
          Complete audit trail of all actions taken on this project
        </p>
      </div>

      {/* Filter tabs */}
      <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
        {actions.map(a => (
          <button
            key={a}
            onClick={() => setFilter(a)}
            className={`workspace-tab ${filter === a ? 'active' : ''}`}
            style={{ fontSize: '11px', padding: '5px 12px', margin: 0 }}
          >
            {a}
          </button>
        ))}
      </div>

      {/* Log entries */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        {filtered.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No audit log entries found.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {filtered.map(log => (
              <div key={log.id} style={{ display: 'flex', gap: '12px', padding: '10px', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                <span className="badge" style={{ fontSize: '9px', background: `${getActionColor(log.action)}20`, color: getActionColor(log.action), flexShrink: 0 }}>
                  {log.action}
                </span>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '12px', fontWeight: '600' }}>{log.details}</div>
                  <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '2px' }}>
                    {log.user} | {log.entity} | {new Date(log.timestamp).toLocaleString()}
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