import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'

// StatGate single-sign-on bootstrap: accept a Registry-issued token handed over
// by the launcher (?statgate_token canonical, ?registry_token legacy alias),
// persist it under the platform key, and strip it from the address bar so it is
// not retained in history or shared links.
(function bootstrapStatGateSSO() {
  if (typeof window === 'undefined') return
  const sp = new URLSearchParams(window.location.search)
  const handed = sp.get('statgate_token') || sp.get('registry_token')
  if (!handed) return
  window.localStorage.setItem('registry_jwt', handed)
  const u = new URL(window.location.href)
  u.searchParams.delete('statgate_token')
  u.searchParams.delete('registry_token')
  window.history.replaceState({}, document.title, `${u.pathname}${u.search}${u.hash}`)
})()

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
