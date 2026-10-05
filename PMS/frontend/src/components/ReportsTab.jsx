import React, { useEffect, useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function ReportsTab({ reports = [], projectId, onRefresh }) {
  const [isGenerating, setIsGenerating] = useState(false);
  const [reportTitle, setReportTitle] = useState('');
  const [reportType, setReportType] = useState('Progress');
  const [reportFormat, setReportFormat] = useState('PDF');
  const [generatedBy, setGeneratedBy] = useState('Dr. Sarah Jenkins');
  const [templateId, setTemplateId] = useState('');
  const [templates, setTemplates] = useState([]);
  const [donors, setDonors] = useState([]);
  const [donorId, setDonorId] = useState('');
  const [reportingPeriodStart, setReportingPeriodStart] = useState('');
  const [reportingPeriodEnd, setReportingPeriodEnd] = useState('');
  const [error, setError] = useState('');
  const [viewingReport, setViewingReport] = useState(null);

  useEffect(() => {
    if (!projectId) return;
    Promise.all([
      fetch(`${API_BASE}/api/projects/${projectId}/report-templates`).then(res => res.ok ? res.json() : []),
      fetch(`${API_BASE}/api/donors`).then(res => res.ok ? res.json() : []),
    ])
      .then(([availableTemplates, availableDonors]) => {
        setTemplates(Array.isArray(availableTemplates) ? availableTemplates : []);
        setDonors(Array.isArray(availableDonors) ? availableDonors : []);
      })
      .catch(() => setError('Reporting templates could not be loaded.'));
  }, [projectId]);

  const handleGenerate = async (e) => {
    e.preventDefault();
    setError('');
    const isDonorReport = Boolean(templateId);
    const endpoint = isDonorReport
      ? `${API_BASE}/api/projects/${projectId}/donor-reports`
      : `${API_BASE}/api/projects/${projectId}/reports`;
    const body = isDonorReport
      ? { title: reportTitle, templateId, donorId, format: reportFormat, generatedBy, reportingPeriodStart, reportingPeriodEnd }
      : { title: reportTitle, type: reportType, format: reportFormat, generatedBy };
    try {
      const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Report generation failed.');
      onRefresh();
      setIsGenerating(false);
      setReportTitle('');
      setViewingReport(data);
    } catch (err) {
      setError(err.message || 'Report generation failed.');
    }
  };

  const getTypeIcon = (type) => {
    const icons = {
      'Progress': '📊',
      'Financial': '💰',
      'Executive': '🏛️',
      'Activity': '📋',
      'Risk': '⚠️',
      'Survey': '📝',
      'Final': '📄',
      'Donor': '🤝',
    };
    return icons[type] || '📄';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📊 Reports & Analytics</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Generate and view project reports from live data
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }} onClick={() => setIsGenerating(true)}>
          + Generate Report
        </button>
      </div>

      {/* Generate Report Modal */}
      {isGenerating && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleGenerate}>
            <h3>Generate Project Report</h3>
            {error && <div className="form-error" role="alert">{error}</div>}
            <div className="form-group">
              <label>Report Title</label>
              <input type="text" value={reportTitle} onChange={e => setReportTitle(e.target.value)} placeholder="e.g. Q3 Progress Report" required />
            </div>
            <div className="form-group">
              <label>Reporting Template</label>
              <select value={templateId} onChange={e => setTemplateId(e.target.value)}>
                <option value="">Standard project report</option>
                {templates.map(template => <option key={template.id} value={template.id}>{template.name}</option>)}
              </select>
              {templateId && <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{templates.find(template => template.id === templateId)?.description}</span>}
            </div>
            {templateId && (
              <>
                <div className="form-group">
                  <label>Donor (optional)</label>
                  <select value={donorId} onChange={e => setDonorId(e.target.value)}>
                    <option value="">All project funding / donor not specified</option>
                    {donors.map(donor => <option key={donor.id} value={donor.id}>{donor.name}</option>)}
                  </select>
                </div>
                <div className="form-row">
                  <div className="form-group">
                    <label>Period start (optional)</label>
                    <input type="date" value={reportingPeriodStart} onChange={e => setReportingPeriodStart(e.target.value)} />
                  </div>
                  <div className="form-group">
                    <label>Period end (optional)</label>
                    <input type="date" value={reportingPeriodEnd} onChange={e => setReportingPeriodEnd(e.target.value)} />
                  </div>
                </div>
              </>
            )}
            <div className="form-row">
              <div className="form-group">
                <label>Report Type</label>
                <select value={reportType} onChange={e => setReportType(e.target.value)}>
                  <option value="Progress">Progress</option>
                  <option value="Financial">Financial</option>
                  <option value="Executive">Executive</option>
                  <option value="Activity">Activity</option>
                  <option value="Risk">Risk</option>
                  <option value="Survey">Survey</option>
                  <option value="Donor">Donor</option>
                  <option value="Final">Final</option>
                </select>
              </div>
              <div className="form-group">
                <label>Format</label>
                <select value={reportFormat} onChange={e => setReportFormat(e.target.value)}>
                  <option value="PDF">PDF</option>
                  <option value="HTML">HTML</option>
                  <option value="CSV">CSV</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Generated By</label>
              <input type="text" value={generatedBy} onChange={e => setGeneratedBy(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => { setIsGenerating(false); setError(''); }}>Cancel</button>
              <button type="submit" className="btn btn-primary">Generate</button>
            </div>
          </form>
        </div>
      )}

      {/* View Report Modal */}
      {viewingReport && (
        <div className="modal-overlay">
          <div className="modal-content glass-panel" style={{ width: '700px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <h3 style={{ fontSize: '16px', fontWeight: '700' }}>{viewingReport.title || 'Generated Report'}</h3>
              <button className="btn btn-secondary" style={{ padding: '4px 10px', fontSize: '11px' }} onClick={() => setViewingReport(null)}>✕ Close</button>
            </div>
            <pre style={{ fontSize: '12px', lineHeight: '1.6', whiteSpace: 'pre-wrap', background: 'rgba(0,0,0,0.03)', padding: '16px', borderRadius: '8px', maxHeight: '400px', overflowY: 'auto' }}>
              {viewingReport.content}
            </pre>
          </div>
        </div>
      )}

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Total Reports', value: reports.length, icon: '📄', color: 'var(--accent-primary)' },
          { label: 'Progress', value: reports.filter(r => r.type === 'Progress').length, icon: '📊', color: 'var(--accent-secondary)' },
          { label: 'Financial', value: reports.filter(r => r.type === 'Financial').length, icon: '💰', color: 'var(--accent-success)' },
          { label: 'Final', value: reports.filter(r => r.type === 'Final').length, icon: '🏁', color: 'var(--accent-warning)' },
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

      {/* Reports list */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Generated Reports</h3>
        {reports.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No reports generated yet.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {reports.map(report => (
              <div key={report.id} style={{ display: 'flex', gap: '14px', padding: '12px', background: 'rgba(255,255,255,0.02)', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.06)' }}>
                <div style={{ fontSize: '24px', flexShrink: 0 }}>{getTypeIcon(report.type)}</div>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{report.title}</span>
                    <div style={{ display: 'flex', gap: '6px' }}>
                      <span className="badge" style={{ fontSize: '9px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{report.type}</span>
                      <span className="badge" style={{ fontSize: '9px', background: 'rgba(255,255,255,0.05)', color: 'var(--text-secondary)' }}>{report.format}</span>
                    </div>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{report.content}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    <span>By: {report.generatedBy}</span>
                    <span>{new Date(report.generatedAt).toLocaleString()}</span>
                  </div>
                </div>
                <button className="btn btn-secondary" style={{ padding: '4px 10px', fontSize: '11px', alignSelf: 'center' }} onClick={() => setViewingReport(report)}>
                  View
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
