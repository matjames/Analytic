import React, { useState } from 'react';

export default function OverviewTab({ workspaceData, onStageTransition }) {
  const [showStageMenu, setShowStageMenu] = useState(false);

  const stages = [
    'Research Idea', 'Concept Note', 'Research Proposal', 'Protocol Development',
    'Internal Review', 'Ethics Submission', 'Ethics Approval', 'Funding Approval',
    'Project Activation', 'Survey Design', 'Field Data Collection', 'Data Validation',
    'Statistical Analysis', 'Interpretation', 'Report Writing', 'Publication',
    'Knowledge Repository', 'Archive'
  ];

  const currentIndex = stages.indexOf(workspaceData.research.stage);
  const progress = workspaceData.research.progress;

  const handleStageChange = (stage) => {
    onStageTransition(stage);
    setShowStageMenu(false);
  };

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '32px' }}>
      {/* Lifecycle Stage Management */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3>Lifecycle Stage Management</h3>
        <p style={{ margin: '8px 0 20px 0', color: 'var(--text-secondary)' }}>
          Advance the research study through the enterprise lifecycle stages.
        </p>

        {/* Progress Bar */}
        <div style={{ marginBottom: '24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px', fontSize: '14px' }}>
            <span>Current Progress</span>
            <span style={{ fontWeight: '600', color: 'var(--primary)' }}>{progress.toFixed(0)}%</span>
          </div>
          <div style={{ width: '100%', height: '8px', background: 'var(--border-light)', borderRadius: '4px', overflow: 'hidden' }}>
            <div style={{ width: `${progress}%`, height: '100%', background: 'linear-gradient(90deg, var(--primary), var(--accent))', borderRadius: '4px', transition: 'width 0.3s' }} />
          </div>
        </div>

        {/* Stage Timeline */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {stages.map((stage, idx) => {
            const isActive = stage === workspaceData.research.stage;
            const isPast = idx < currentIndex;
            const isFuture = idx > currentIndex;

            return (
              <div key={stage} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '50%',
                  background: isActive ? 'var(--primary)' : isPast ? 'var(--success)' : 'var(--border-light)',
                  color: isActive || isPast ? '#fff' : 'var(--text-secondary)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '12px',
                  fontWeight: '600',
                  flexShrink: 0
                }}>
                  {isPast ? '✓' : idx + 1}
                </div>
                <span style={{
                  flex: 1,
                  fontSize: '14px',
                  fontWeight: isActive ? '600' : '400',
                  color: isActive ? 'var(--primary)' : isPast ? 'var(--text-secondary)' : 'var(--text)',
                  textDecoration: isPast ? 'line-through' : 'none'
                }}>
                  {stage}
                </span>
                {isActive && (
                  <span style={{ fontSize: '11px', padding: '2px 8px', background: 'var(--primary)', color: '#fff', borderRadius: '4px' }}>
                    Current
                  </span>
                )}
                {isFuture && idx === currentIndex + 1 && (
                  <button
                    onClick={() => handleStageChange(stage)}
                    style={{
                      fontSize: '12px',
                      padding: '4px 12px',
                      background: 'var(--success)',
                      color: '#fff',
                      border: 'none',
                      borderRadius: '4px',
                      cursor: 'pointer'
                    }}
                  >
                    Advance →
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Metadata Panel */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3>Metadata</h3>
          <div style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div>
              <span style={{ display: 'block', fontSize: '12px', color: 'var(--text-secondary)' }}>Project Code</span>
              <strong>{workspaceData.research.code}</strong>
            </div>
            <div>
              <span style={{ display: 'block', fontSize: '12px', color: 'var(--text-secondary)' }}>Owner</span>
              <strong>{workspaceData.research.owner}</strong>
            </div>
            <div>
              <span style={{ display: 'block', fontSize: '12px', color: 'var(--text-secondary)' }}>Organisation</span>
              <strong>{workspaceData.research.organisation || '—'}</strong>
            </div>
            <div>
              <span style={{ display: 'block', fontSize: '12px', color: 'var(--text-secondary)' }}>Portfolio</span>
              <strong>{workspaceData.research.portfolio || '—'}</strong>
            </div>
            <div>
              <span style={{ display: 'block', fontSize: '12px', color: 'var(--text-secondary)' }}>Programme</span>
              <strong>{workspaceData.research.programme || '—'}</strong>
            </div>
          </div>
        </div>

        {/* Quick Stats */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3>Study Summary</h3>
          <div style={{ marginTop: '16px', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            <div style={{ textAlign: 'center', padding: '12px', background: 'var(--bg-light)', borderRadius: '8px' }}>
              <div style={{ fontSize: '24px', fontWeight: '700', color: 'var(--primary)' }}>
                {workspaceData.members.length}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Team Members</div>
            </div>
            <div style={{ textAlign: 'center', padding: '12px', background: 'var(--bg-light)', borderRadius: '8px' }}>
              <div style={{ fontSize: '24px', fontWeight: '700', color: 'var(--accent)' }}>
                {workspaceData.proposals.length}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Proposals</div>
            </div>
            <div style={{ textAlign: 'center', padding: '12px', background: 'var(--bg-light)', borderRadius: '8px' }}>
              <div style={{ fontSize: '24px', fontWeight: '700', color: 'var(--success)' }}>
                {workspaceData.publications.length}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Publications</div>
            </div>
            <div style={{ textAlign: 'center', padding: '12px', background: 'var(--bg-light)', borderRadius: '8px' }}>
              <div style={{ fontSize: '24px', fontWeight: '700', color: 'var(--warning)' }}>
                {workspaceData.datasets.length}
              </div>
              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Datasets</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}