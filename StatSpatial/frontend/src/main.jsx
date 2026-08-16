import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'

// Cross-origin UIs cannot share localStorage. Accept the launcher hand-off once,
// persist it for this origin, then immediately remove it from browser history.
function bootstrapSharedSignOn() {
  const url = new URL(window.location.href)
  const token = url.searchParams.get('statgate_token') || url.searchParams.get('registry_token')
  if (!token) return

  localStorage.setItem('registry_jwt', token)
  url.searchParams.delete('statgate_token')
  url.searchParams.delete('registry_token')
  window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`)
}

bootstrapSharedSignOn()

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
