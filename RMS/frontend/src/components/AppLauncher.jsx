import React, { useEffect, useRef, useState } from 'react';

const APPS = [
  { id: 'launcher', name: 'All Apps', icon: 'SG', url: 'http://localhost:3006' },
  { id: 'analytics', name: 'Analytics', icon: 'AN', url: 'http://localhost:5000' },
  { id: 'registry', name: 'Registry', icon: 'RG', url: 'http://localhost:3007' },
  { id: 'helpdesk', name: 'Helpdesk', icon: 'HD', url: 'http://localhost:3005' },
  { id: 'statchat', name: 'StatChat', icon: 'SC', url: 'http://localhost:3009' },
  { id: 'pms', name: 'PMS', icon: 'PM', url: 'http://localhost:3010' },
  { id: 'rms', name: 'RMS', icon: 'RM', url: 'http://localhost:3011' },
  { id: 'governance', name: 'Governance', icon: 'GV', url: 'http://localhost:3012' },
  { id: 'spatial', name: 'Spatial', icon: 'SP', url: 'http://localhost:3014' },
  { id: 'report-builder', name: 'Report Builder', icon: 'RB', url: 'http://localhost:8110/builder.html' },
];

export default function AppLauncher({ currentApp }) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);

  useEffect(() => {
    const close = (event) => {
      if (event.key === 'Escape' || (event.type === 'mousedown' && !rootRef.current?.contains(event.target))) setOpen(false);
    };
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', close);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('keydown', close);
    };
  }, []);

  return (
    <div ref={rootRef} style={{ position: 'relative' }}>
      <button type="button" onClick={() => setOpen((value) => !value)} aria-label="Open app launcher" aria-expanded={open} title="StatGate app launcher" style={styles.trigger}>
        <span aria-hidden="true" style={styles.dots}>{Array.from({ length: 9 }, (_, index) => <i key={index} style={styles.dot} />)}</span>
      </button>
      {open && (
        <div role="dialog" aria-label="StatGate applications" style={styles.panel}>
          <strong style={styles.title}>StatGate applications</strong>
          <div style={styles.grid}>
            {APPS.map((app) => {
              const active = app.id === currentApp;
              return <a key={app.id} href={app.url} onClick={() => setOpen(false)} aria-current={active ? 'page' : undefined} style={{ ...styles.tile, ...(active ? styles.active : {}) }}>
                <span style={styles.icon}>{app.icon}</span>
                <span style={styles.name}>{app.name}</span>
              </a>;
            })}
          </div>
          <a href="http://localhost:3006" style={styles.footer}>Open the complete app catalogue</a>
        </div>
      )}
    </div>
  );
}

const styles = {
  trigger: { width: 38, height: 38, display: 'grid', placeItems: 'center', border: '1px solid #cbd5e1', borderRadius: 9, background: '#fff', cursor: 'pointer' },
  dots: { width: 18, height: 18, display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 3 },
  dot: { width: 4, height: 4, borderRadius: '50%', background: '#334155' },
  panel: { position: 'absolute', top: 46, right: 0, zIndex: 1000, width: 318, padding: 16, border: '1px solid #dbe3ec', borderRadius: 14, background: '#fff', boxShadow: '0 18px 48px rgba(15,23,42,.2)' },
  title: { display: 'block', marginBottom: 12, color: '#0f172a', fontSize: 13 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 8 },
  tile: { minHeight: 78, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 7, padding: 8, border: '1px solid #e2e8f0', borderRadius: 10, background: '#f8fafc', color: '#334155', textDecoration: 'none', textAlign: 'center' },
  active: { borderColor: '#2563eb', background: '#eff6ff', color: '#1d4ed8' },
  icon: { fontSize: 16, lineHeight: 1, fontWeight: 700 },
  name: { fontSize: 11, fontWeight: 700 },
  footer: { display: 'block', marginTop: 12, color: '#2563eb', fontSize: 12, fontWeight: 700, textAlign: 'center', textDecoration: 'none' },
};
