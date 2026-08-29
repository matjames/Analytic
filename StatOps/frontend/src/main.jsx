import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'

const nativeFetch = window.fetch.bind(window)
window.fetch = (input, init = {}) => {
  const headers = new Headers(init.headers || (input instanceof Request ? input.headers : undefined))
  const workspaceID = localStorage.getItem('workspace_id')
  const token = localStorage.getItem('registry_jwt')
  if (workspaceID) headers.set('X-Workspace-ID', workspaceID)
  if (token) headers.set('Authorization', 'Bearer ' + token)
  return nativeFetch(input, { ...init, headers })
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
