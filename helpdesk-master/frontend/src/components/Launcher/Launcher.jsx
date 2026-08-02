import React, { useRef, useEffect, useState } from 'react'

const launcherApps = [
  { name: 'Analytics', url: 'http://localhost:5000', icon: 'fa fa-chart-line' },
  { name: 'Registry', url: 'http://localhost:3007', icon: 'fa fa-archive' },
  { name: 'Helpdesk', url: 'http://localhost:3005', icon: 'fa fa-life-ring' },
  { name: 'StatChat', url: 'http://localhost:3009', icon: 'fa fa-comments' },
  { name: 'System Launcher', url: 'http://localhost:3002', icon: 'bi bi-grid-3x3-gap-fill' },
  { name: 'Register Portal', url: 'http://localhost:3000', icon: 'fa fa-edit' },
]

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
        <div className="launcher-menu">
          {launcherApps.map((app) => (
            <a key={app.name} href={app.url} className="launcher-card" onClick={() => setOpen(false)}>
              <i className={app.icon}></i>
              <span>{app.name}</span>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
