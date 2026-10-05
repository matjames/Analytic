import React, { useEffect, useState } from 'react';

function ScoreCard({ title, review }) {
  if (!review) return <div className="empty-state">Loading {title.toLowerCase()}...</div>;
  return <div className="data-card">
    <div>
      <strong>{title}</strong>
      <p>{review.band} | governed deterministic review</p>
    </div>
    <strong>{review.score}/100</strong>
  </div>;
}

function CheckList({ checks = [] }) {
  return <div className="data-list">{checks.map(check => <article className="data-card" key={check.key}>
    <div><strong>{check.passed ? 'Passed' : 'Needs attention'}: {check.label}</strong><p>{check.detail}</p></div>
    <span className="badge">{check.severity}</span>
  </article>)}</div>;
}

export default function PublicationQualityTab({ apiBase, researchId }) {
  const [readiness, setReadiness] = useState(null);
  const [quality, setQuality] = useState(null);
  const [text, setText] = useState('');
  const [languageMode, setLanguageMode] = useState('deterministic');
  const [language, setLanguage] = useState(null);
  const [languageError, setLanguageError] = useState('');
  const [loading, setLoading] = useState(false);

  const load = () => {
    if (!researchId) {
      setReadiness(null);
      setQuality(null);
      return;
    }
    setLoading(true);
    Promise.all([
      fetch(`${apiBase}/api/research/${researchId}/publication-readiness`).then(r => { if (!r.ok) throw new Error('readiness failed'); return r.json(); }),
      fetch(`${apiBase}/api/research/${researchId}/quality-review`).then(r => { if (!r.ok) throw new Error('quality failed'); return r.json(); }),
    ]).then(([readinessResult, qualityResult]) => {
      setReadiness(readinessResult);
      setQuality(qualityResult);
    }).catch(() => {
      setReadiness(null);
      setQuality(null);
    }).finally(() => setLoading(false));
  };

  useEffect(load, [researchId]);

  const reviewLanguage = (event) => {
    event.preventDefault();
    setLanguageError('');
    fetch(`${apiBase}/api/research/${researchId}/language-review`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text, mode: languageMode }),
    }).then(async r => {
      const body = await r.json();
      if (!r.ok) throw new Error(body.error || 'language review failed');
      return body;
    }).then(setLanguage).catch(error => { setLanguage(null); setLanguageError(error.message); });
  };

  if (!researchId) return <div className="empty-state">Select a research study to review publication readiness and quality.</div>;
  if (loading && !readiness) return <div className="empty-state">Reviewing registered research evidence...</div>;

  return <div>
    <div className="section-heading"><div><h2>Publication Quality and Readiness</h2><p>Transparent checks over registered study records. These reviews support authors and reviewers; they do not replace ethics, peer, or editorial decisions.</p></div><button className="btn btn-primary" onClick={load}>Refresh Reviews</button></div>
    <div className="data-list">
      <ScoreCard title="Publication readiness" review={readiness} />
      <ScoreCard title="Research quality" review={quality} />
    </div>
    {readiness && <><h3>Publication readiness checks</h3><CheckList checks={readiness.checks} /><div className="data-card"><strong>Recommendations</strong>{readiness.recommendations.map(item => <p key={item}>{item}</p>)}</div></>}
    {quality && <><h3>Research quality checks</h3><CheckList checks={quality.checks} /><div className="data-card"><strong>Quality actions</strong>{quality.recommendations.map(item => <p key={item}>{item}</p>)}</div></>}
    <div className="section-heading"><div><h3>Language review and editing</h3><p>Use deterministic guidance by default, or request bounded provider editing when an approved LLM endpoint is configured.</p></div></div>
    <form onSubmit={reviewLanguage} className="inline-form">
      <label>Text to review<textarea value={text} onChange={event => setText(event.target.value)} rows="8" placeholder="Paste text here, or leave blank to review registered study metadata." /></label>
      <label>Review mode<select value={languageMode} onChange={event => setLanguageMode(event.target.value)}><option value="deterministic">Deterministic guidance</option><option value="provider">Provider editing</option></select></label>
      <div><button className="btn btn-primary" type="submit">{languageMode === 'provider' ? 'Edit Text' : 'Review Text'}</button></div>
    </form>
    {languageError && <div className="data-card"><strong>Language review unavailable</strong><p>{languageError}</p></div>}
    {language && <div className="data-list"><article className="data-card"><div><strong>Suggested text</strong><p>{language.suggestedText || 'No text was available for review.'}</p></div><span className="badge">{language.mode}</span></article>{language.issues?.length ? language.issues.map(issue => <article className="data-card" key={issue.key}><div><strong>{issue.key}</strong><p>{issue.detail}</p></div><span className="badge">{issue.severity}</span></article>) : <div className="data-card"><strong>No deterministic language issues detected.</strong></div>}<small>{language.disclaimer}</small></div>}
  </div>;
}
