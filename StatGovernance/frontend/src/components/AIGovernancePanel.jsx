import React, { useState } from 'react';

export default function AIGovernancePanel({ apiBase, token }) {
  const [question, setQuestion] = useState('');
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState(null);

  const sampleQueries = [
    "What policies and controls govern our data classification and privacy?",
    "Which institutional risks are currently rated Critical or High?",
    "What evidence is required for statutory Data Protection Act compliance?"
  ];

  const handleAskAI = async (queryText) => {
    const q = queryText || question;
    if (!q) return;

    try {
      setLoading(true);
      setResponse(null);
      const res = await fetch(`${apiBase}/api/ai/analyze`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ question: q })
      });
      if (res.ok) {
        setResponse(await res.json());
      }
    } catch (err) {
      console.error('Error invoking AI analysis:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <div className="glass-panel" style={{ padding: 22, background: 'linear-gradient(135deg, #092c4d 0%, #1e293b 100%)', color: 'white' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
          <span style={{ fontSize: 28 }}>🤖</span>
          <div>
            <h2 style={{ fontSize: 20, fontWeight: 800 }}>StatGate Enterprise AI — Evidence-First Governance Intelligence</h2>
            <p style={{ fontSize: 13, color: '#93c5fd' }}>
              Phase VII grounded institutional intelligence. Formulates analytical recommendations directly from persisted policies, risks, controls, and audit findings.
            </p>
          </div>
        </div>

        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', gap: 10 }}>
            <input
              style={{
                flex: 1,
                padding: '12px 16px',
                borderRadius: 8,
                border: '1px solid rgba(255, 255, 255, 0.2)',
                background: 'rgba(255, 255, 255, 0.1)',
                color: 'white',
                fontSize: 14,
                outline: 'none'
              }}
              placeholder="Ask a question about institutional governance, compliance gaps, or risk controls..."
              value={question}
              onChange={e => setQuestion(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleAskAI()}
            />
            <button
              className="btn btn-primary"
              style={{ background: '#38bdf8', color: '#092c4d', fontWeight: 800, padding: '0 24px' }}
              onClick={() => handleAskAI()}
              disabled={loading}
            >
              {loading ? 'Synthesizing...' : 'Analyze Evidence'}
            </button>
          </div>

          <div style={{ display: 'flex', gap: 8, marginTop: 12, alignItems: 'center', flexWrap: 'wrap' }}>
            <span style={{ fontSize: 11, color: '#94a3b8', textTransform: 'uppercase', fontWeight: 700 }}>Prompt Ideas:</span>
            {sampleQueries.map((sample, idx) => (
              <button
                key={idx}
                style={{
                  background: 'rgba(255, 255, 255, 0.08)',
                  border: '1px solid rgba(255, 255, 255, 0.15)',
                  borderRadius: 6,
                  color: '#cbd5e1',
                  fontSize: 12,
                  padding: '4px 10px',
                  cursor: 'pointer'
                }}
                onClick={() => {
                  setQuestion(sample);
                  handleAskAI(sample);
                }}
              >
                {sample}
              </button>
            ))}
          </div>
        </div>
      </div>

      {response && (
        <div className="glass-panel" style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 18 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border-light)', paddingBottom: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span className={`badge ${response.status === 'grounded' ? 'badge-active' : 'badge-draft'}`}>
                {response.status === 'grounded' ? '✓ Evidence Grounded' : '⚠ Insufficient Evidence'}
              </span>
              <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-dark)' }}>
                {response.question}
              </span>
            </div>
            <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
              Generated: {new Date(response.generated_at).toLocaleTimeString()}
            </span>
          </div>

          {/* Sourced Findings */}
          <div>
            <div style={{ fontSize: 11.5, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 8 }}>
              1. Grounded Facts & Evidence Extracts
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {(response.findings || []).map((f, idx) => (
                <div key={idx} style={{ background: '#f8fafc', padding: 12, borderRadius: 8, border: '1px solid var(--border-light)', fontSize: 13, display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ color: 'var(--accent-emerald)', fontWeight: 800 }}>✓</span>
                  <span style={{ flex: 1, color: 'var(--text-dark)' }}>{f.statement}</span>
                  <span className="badge badge-approved" style={{ fontSize: 10 }}>Confidence: {f.confidence}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Traceable Sources */}
          <div>
            <div style={{ fontSize: 11.5, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 8 }}>
              2. Traceable Institutional Sources
            </div>
            <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
              {(response.sources || []).map((s, idx) => (
                <div key={idx} style={{ background: '#ffffff', border: '1px solid var(--border-light)', padding: '8px 12px', borderRadius: 8, fontSize: 12 }}>
                  <span style={{ fontWeight: 700, color: 'var(--primary-color)' }}>[{s.entity?.toUpperCase()}]</span> {s.title}
                </div>
              ))}
            </div>
          </div>

          {/* Synthesis & Recommendation */}
          <div style={{ background: '#f0fdf4', padding: 16, borderRadius: 8, border: '1px solid #bbf7d0' }}>
            <div style={{ fontSize: 12, fontWeight: 700, textTransform: 'uppercase', color: '#166534', marginBottom: 4 }}>
              3. Institutional Analysis & Recommendation
            </div>
            <div style={{ fontSize: 13.5, color: '#14532d', lineHeight: 1.6, marginBottom: 8 }}>
              {response.analysis}
            </div>
            {response.recommendation && (
              <div style={{ fontSize: 13, color: '#166534', fontWeight: 600 }}>
                💡 Recommendation: {response.recommendation}
              </div>
            )}
          </div>

          <div style={{ fontSize: 11.5, color: 'var(--text-muted)', fontStyle: 'italic', borderTop: '1px solid var(--border-light)', paddingTop: 8 }}>
            🔒 {response.human_control_notice || 'Human authority required. AI does not autonomously create binding decisions or policy approvals.'}
          </div>
        </div>
      )}
    </div>
  );
}
