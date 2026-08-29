import React, { useEffect, useRef, useState } from 'react';

const APPS = [
  ['launcher', 'All Apps', 'SG', 'http://localhost:3006'], ['analytics', 'Analytics', '📊', 'http://localhost:5000'],
  ['registry', 'Registry', '🪪', 'http://localhost:3007'], ['helpdesk', 'Helpdesk', '🎧', 'http://localhost:3005'],
  ['statchat', 'StatChat', '💬', 'http://localhost:3009'], ['pms', 'PMS', '🏗️', 'http://localhost:3010'],
  ['rms', 'RMS', '🔬', 'http://localhost:3011'], ['governance', 'Governance', '⚖️', 'http://localhost:3012'],
  ['spatial', 'Spatial', '🗺️', 'http://localhost:3014'], ['report-builder', 'Report Builder', '🧮', 'http://localhost:8110/builder.html'],
];

export default function AppLauncher({ currentApp }) {
  const [open, setOpen] = useState(false); const root = useRef(null);
  useEffect(() => { const close = e => { if (e.key === 'Escape' || (e.type === 'mousedown' && !root.current?.contains(e.target))) setOpen(false); }; document.addEventListener('mousedown', close); document.addEventListener('keydown', close); return () => { document.removeEventListener('mousedown', close); document.removeEventListener('keydown', close); }; }, []);
  return <div ref={root} style={{ position: 'relative' }}>
    <button type="button" onClick={() => setOpen(v => !v)} aria-label="Open 3 by 3 app launcher" aria-expanded={open} title="StatGate app launcher" style={s.trigger}><span aria-hidden="true" style={s.dots}>{Array.from({ length: 9 }, (_, i) => <i key={i} style={s.dot} />)}</span></button>
    {open && <div role="dialog" aria-label="StatGate applications" style={s.panel}><strong style={s.title}>StatGate applications</strong><div style={s.grid}>{APPS.map(([id, name, icon, url]) => <a key={id} href={url} onClick={() => setOpen(false)} aria-current={id === currentApp ? 'page' : undefined} style={{ ...s.tile, ...(id === currentApp ? s.active : {}) }}><span style={s.icon}>{icon}</span><span style={s.name}>{name}</span></a>)}</div><a href="http://localhost:3006" style={s.footer}>Open the complete app catalogue →</a></div>}
  </div>;
}

const s = {
  trigger: { width: 38, height: 38, display: 'grid', placeItems: 'center', border: '1px solid #cbd5e1', borderRadius: 9, background: '#fff', cursor: 'pointer' }, dots: { width: 18, height: 18, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 3 }, dot: { width: 4, height: 4, borderRadius: '50%', background: '#334155' },
  panel: { position: 'absolute', top: 46, right: 0, zIndex: 1000, width: 318, padding: 16, border: '1px solid #dbe3ec', borderRadius: 14, background: '#fff', boxShadow: '0 18px 48px rgba(15,23,42,.2)' }, title: { display: 'block', marginBottom: 12, color: '#0f172a', fontSize: 13 }, grid: { display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 8 },
  tile: { minHeight: 78, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 7, padding: 8, border: '1px solid #e2e8f0', borderRadius: 10, background: '#f8fafc', color: '#334155', textDecoration: 'none', textAlign: 'center' }, active: { borderColor: '#2563eb', background: '#eff6ff', color: '#1d4ed8' }, icon: { fontSize: 21, lineHeight: 1 }, name: { fontSize: 11, fontWeight: 700 }, footer: { display: 'block', marginTop: 12, color: '#2563eb', fontSize: 12, fontWeight: 700, textAlign: 'center', textDecoration: 'none' },
};
