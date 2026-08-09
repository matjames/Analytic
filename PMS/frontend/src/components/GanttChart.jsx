import React from 'react';

export default function GanttChart({ tasks }) {
  if (!tasks || tasks.length === 0) {
    return (
      <div className="glass-panel" style={{ padding: '24px', textAlign: 'center', color: 'var(--text-secondary)' }}>
        No tasks or milestones available to plot schedule.
      </div>
    );
  }

  // Find overall start and end bounds
  const getTimelineBounds = () => {
    let start = new Date();
    let end = new Date();
    end.setMonth(end.getMonth() + 2); // Default 2 months out

    const dates = tasks
      .map(t => [new Date(t.startDate), new Date(t.endDate)])
      .flat()
      .filter(d => !isNaN(d.getTime()));

    if (dates.length > 0) {
      start = new Date(Math.min(...dates));
      end = new Date(Math.max(...dates));
      // Buffer by a few days
      start.setDate(start.getDate() - 5);
      end.setDate(end.getDate() + 10);
    }
    return { start, end };
  };

  const { start: minDate, end: maxDate } = getTimelineBounds();
  const totalDays = Math.ceil((maxDate - minDate) / (1000 * 60 * 60 * 24)) || 30;

  const getPositionPercentage = (dateStr) => {
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return 0;
    const diff = date - minDate;
    const pct = (diff / (1000 * 60 * 60 * 24) / totalDays) * 100;
    return Math.max(0, Math.min(100, pct));
  };

  return (
    <div className="glass-panel" style={{ padding: '24px', overflowX: 'auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <h3 style={{ fontSize: '18px', fontWeight: '700' }}>Work Breakdown Structure & Critical Path</h3>
        <span style={{ fontSize: '12px', color: 'var(--accent-secondary)', fontWeight: '600' }}>
          Timeline Bounds: {minDate.toLocaleDateString()} – {maxDate.toLocaleDateString()}
        </span>
      </div>

      <div style={{ minWidth: '800px' }}>
        {/* Timeline Header Scale */}
        <div style={{ display: 'grid', gridTemplateColumns: '250px 1fr', borderBottom: '1px solid var(--border-glass)', paddingBottom: '8px', marginBottom: '12px' }}>
          <div style={{ fontWeight: '750', fontSize: '12px', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>Activity & WBS</div>
          <div style={{ position: 'relative', height: '20px' }}>
            <div style={{ position: 'absolute', left: '0%', fontSize: '11px', color: 'var(--text-muted)' }}>Start</div>
            <div style={{ position: 'absolute', left: '33%', fontSize: '11px', color: 'var(--text-muted)' }}>Phase I</div>
            <div style={{ position: 'absolute', left: '66%', fontSize: '11px', color: 'var(--text-muted)' }}>Phase II</div>
            <div style={{ position: 'absolute', right: '0%', fontSize: '11px', color: 'var(--text-muted)' }}>End</div>
          </div>
        </div>

        {/* Gantt Rows */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {tasks.map((task) => {
            const startPct = getPositionPercentage(task.startDate);
            const endPct = getPositionPercentage(task.endDate);
            const widthPct = Math.max(2, endPct - startPct);

            return (
              <div key={task.ID || task.id} style={{ display: 'grid', gridTemplateColumns: '250px 1fr', alignItems: 'center' }}>
                <div style={{ display: 'flex', flexDirection: 'column', paddingRight: '12px' }}>
                  <div style={{ fontSize: '13px', fontWeight: '600', display: 'flex', alignItems: 'center', gap: '6px' }}>
                    <span style={{ color: 'var(--accent-primary)', fontSize: '11px', fontFamily: 'monospace' }}>{task.wbsCode}</span>
                    {task.title}
                  </div>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Assignee: {task.assignedTo || 'Unassigned'}</span>
                </div>
                <div style={{ position: 'relative', height: '36px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: '6px', border: '1px solid rgba(255,255,255,0.03)' }}>
                  {/* Gantt Bar */}
                  <div
                    style={{
                      position: 'absolute',
                      left: `${startPct}%`,
                      width: `${widthPct}%`,
                      height: '24px',
                      top: '5px',
                      background: 'linear-gradient(90deg, rgba(99, 102, 241, 0.25) 0%, rgba(6, 182, 212, 0.25) 100%)',
                      border: '1px solid rgba(99, 102, 241, 0.5)',
                      borderRadius: '12px',
                      display: 'flex',
                      alignItems: 'center',
                      padding: '0 8px',
                      boxSizing: 'border-box',
                      cursor: 'pointer'
                    }}
                    title={`${task.title}: ${task.startDate} to ${task.endDate}`}
                  >
                    {/* Inner Progress Bar */}
                    <div
                      style={{
                        position: 'absolute',
                        left: 0,
                        top: 0,
                        height: '100%',
                        width: `${task.progress}%`,
                        background: 'linear-gradient(90deg, var(--accent-primary) 0%, var(--accent-secondary) 100%)',
                        borderRadius: '12px',
                        zIndex: -1,
                        transition: 'width 0.3s ease'
                      }}
                    />
                    <span style={{ fontSize: '10px', fontWeight: '700', color: '#fff', whiteSpace: 'nowrap', textShadow: '0 1px 2px rgba(0,0,0,0.5)' }}>
                      {task.progress}%
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
