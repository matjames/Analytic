import React, { useRef, useEffect, useState } from 'react'
import launcherApps from '../../config/launcherApps'

export default function Launcher() {
  const [open, setOpen] = useState(false)
  const ref = useRef(null)

  useEffect(() => {
    function onClick(e) {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  return (
    <div className="launcher-dropdown" ref={ref}>
      <button type="button" className="launcher-toggle" aria-label="Open app launcher" onClick={() => setOpen(!open)}>
        <i className="bi bi-grid-3x3-gap-fill"></i>
      </button>
      {open && (
        <div className="launcher-menu" role="dialog" aria-label="StatGate applications" style={{ position: 'absolute', right: 0, top: 'calc(100% + 8px)', width: 320, padding: 12, display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 8, background: '#fff', border: '1px solid #dbe3ec', borderRadius: 12, boxShadow: '0 18px 48px rgba(15,23,42,.2)', zIndex: 2000 }}>
          {launcherApps.map((app) => (
            <a key={app.name} href={app.url} className="launcher-card" onClick={() => setOpen(false)} style={{ minHeight: 78, padding: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 7, color: '#334155', background: app.name === 'Helpdesk' ? '#eff6ff' : '#f8fafc', border: '1px solid #e2e8f0', borderRadius: 10, textDecoration: 'none', textAlign: 'center' }}>
              <span style={{fontSize: 20}}>{app.icon}</span>
              <span style={{fontSize: 11, fontWeight: 700}}>{app.name}</span>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
