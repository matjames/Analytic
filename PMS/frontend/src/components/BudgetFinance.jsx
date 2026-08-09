import React from 'react';

export default function BudgetFinance({ budgetTotal, spentTotal, budgetLines, costCentres = [], budgetRevisions = [], procurementRefs = [] }) {
  const spentPercent = budgetTotal > 0 ? (spentTotal / budgetTotal) * 100 : 0;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      {/* Financial Overview Cards */}
      <div className="stat-grid" style={{ marginBottom: 0 }}>
        <div className="glass-panel stat-card">
          <span className="stat-label">Total Allocated Budget</span>
          <span className="stat-value" style={{ color: 'var(--accent-secondary)' }}>
            ${budgetTotal ? budgetTotal.toLocaleString() : '0'}
          </span>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Approved funding from all sources</span>
        </div>

        <div className="glass-panel stat-card">
          <span className="stat-label">Total Expenditures (Spent)</span>
          <span className="stat-value" style={{ color: 'var(--accent-primary)' }}>
            ${spentTotal ? spentTotal.toLocaleString() : '0'}
          </span>
          <div style={{ width: '100%', marginTop: '6px' }}>
            <div className="project-progress-bar" style={{ height: '4px' }}>
              <div className="project-progress-fill" style={{ width: `${Math.min(100, spentPercent)}%` }}></div>
            </div>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)', display: 'block', marginTop: '4px' }}>
              {spentPercent.toFixed(1)}% of total budget consumed
            </span>
          </div>
        </div>

        <div className="glass-panel stat-card">
          <span className="stat-label">Remaining Balance</span>
          <span className="stat-value" style={{ color: spentPercent > 90 ? 'var(--accent-danger)' : 'var(--accent-success)' }}>
            ${(budgetTotal - spentTotal) ? (budgetTotal - spentTotal).toLocaleString() : '0'}
          </span>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Available unspent balances</span>
        </div>
      </div>

      {/* Budget Ledger Table */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Category Allocation & Expenditure Ledger</h3>
        
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border-glass)', textAlign: 'left', color: 'var(--text-secondary)' }}>
              <th style={{ padding: '12px 8px' }}>Category</th>
              <th style={{ padding: '12px 8px' }}>Description</th>
              <th style={{ padding: '12px 8px' }}>Source</th>
              <th style={{ padding: '12px 8px', textAlign: 'right' }}>Allocated Amount</th>
              <th style={{ padding: '12px 8px', textAlign: 'right' }}>Spent Amount</th>
              <th style={{ padding: '12px 8px', textAlign: 'right' }}>Utilization</th>
            </tr>
          </thead>
          <tbody>
            {budgetLines && budgetLines.map(line => {
              const util = line.amount > 0 ? (line.spent / line.amount) * 100 : 0;
              return (
                <tr key={line.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
                  <td style={{ padding: '12px 8px', fontWeight: '600' }}>{line.category}</td>
                  <td style={{ padding: '12px 8px', color: 'var(--text-secondary)' }}>{line.description}</td>
                  <td style={{ padding: '12px 8px' }}>
                    <span className="badge" style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-secondary)' }}>{line.source}</span>
                  </td>
                  <td style={{ padding: '12px 8px', textAlign: 'right', fontWeight: '600' }}>${line.amount.toLocaleString()}</td>
                  <td style={{ padding: '12px 8px', textAlign: 'right', color: 'var(--accent-primary)' }}>${line.spent.toLocaleString()}</td>
                  <td style={{ padding: '12px 8px', textAlign: 'right' }}>
                    <span style={{ 
                      fontWeight: '700', 
                      color: util > 85 ? 'var(--accent-warning)' : 'var(--text-primary)'
                    }}>
                      {util.toFixed(0)}%
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Cost Centres */}
      {costCentres.length > 0 && (
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>🏢 Cost Centres</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '12px' }}>
            {costCentres.map(cc => {
              const util = cc.budget > 0 ? (cc.spent / cc.budget) * 100 : 0;
              return (
                <div key={cc.id} style={{ padding: '14px', background: 'rgba(22,92,146,0.03)', borderRadius: '8px', border: '1px solid rgba(22,92,146,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{cc.name}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{cc.code}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{cc.description}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--text-muted)', marginTop: '6px' }}>
                    <span>Budget: ${cc.budget.toLocaleString()}</span>
                    <span>Spent: ${cc.spent.toLocaleString()}</span>
                  </div>
                  <div style={{ marginTop: '6px' }}>
                    <div className="project-progress-bar" style={{ height: '4px' }}>
                      <div className="project-progress-fill" style={{ width: `${Math.min(100, util)}%` }} />
                    </div>
                    <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>{util.toFixed(1)}% utilized</div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Budget Revisions */}
      {budgetRevisions.length > 0 && (
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>📝 Budget Revisions</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {budgetRevisions.map(br => (
              <div key={br.id} style={{ display: 'flex', gap: '12px', padding: '10px', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                <span className="badge" style={{ fontSize: '9px', background: 'rgba(99,102,241,0.1)', color: 'var(--accent-primary)', flexShrink: 0 }}>v{br.version}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '12px', fontWeight: '600' }}>{br.description}</div>
                  <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '2px' }}>
                    Amount: ${br.amount.toLocaleString()} | Status: {br.status} | Approved by: {br.approvedBy}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Procurement References */}
      {procurementRefs.length > 0 && (
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>📦 Procurement References</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {procurementRefs.map(pr => (
              <div key={pr.id} style={{ display: 'flex', gap: '12px', padding: '10px', borderBottom: '1px solid rgba(255,255,255,0.04)' }}>
                <span className="badge" style={{ fontSize: '9px', background: 'rgba(6,182,212,0.1)', color: 'var(--accent-secondary)', flexShrink: 0 }}>{pr.reference}</span>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: '12px', fontWeight: '600' }}>{pr.description}</div>
                  <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '2px' }}>
                    Vendor: {pr.vendor} | Amount: ${pr.amount.toLocaleString()} | Status: {pr.status}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}