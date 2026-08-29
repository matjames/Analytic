import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

function bootstrapSharedSignOn() {
  const url = new URL(window.location.href)
  const token = url.searchParams.get('statgate_token') || url.searchParams.get('registry_token')
  const workspaceID = url.searchParams.get('workspace_id')

  if (token) localStorage.setItem('registry_jwt', token)
  if (workspaceID) localStorage.setItem('workspace_id', workspaceID)
  if (!token && !workspaceID) return

  url.searchParams.delete('statgate_token')
  url.searchParams.delete('registry_token')
  url.searchParams.delete('workspace_id')
  window.history.replaceState({}, document.title, url.pathname + url.search + url.hash)
}

bootstrapSharedSignOn()

const nativeFetch = window.fetch.bind(window)
window.fetch = (input, init = {}) => {
  const headers = new Headers(init.headers || (input instanceof Request ? input.headers : undefined))
  const workspaceID = localStorage.getItem('workspace_id')
  const token = localStorage.getItem('registry_jwt')
  if (workspaceID) headers.set('X-Workspace-ID', workspaceID)
  if (token) headers.set('Authorization', 'Bearer ' + token)
  return nativeFetch(input, { ...init, headers })
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
