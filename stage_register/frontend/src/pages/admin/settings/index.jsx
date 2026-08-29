import React from 'react'
import { useEffect, useState } from 'react'
import { UsersApi } from '../../../helpers/api/users'

const Settings = () => {
  const [user, setUser] = useState(null)
  const [setup, setSetup] = useState(null)
  const [recoveryCodes, setRecoveryCodes] = useState([])
  const [sessions, setSessions] = useState([])
  const [branding, setBranding] = useState({ display_name: '', logo_url: '', primary_color: '#0f766e', secondary_color: '#0f3d3e' })
  const [preferences, setPreferences] = useState({ locale: 'en', timezone: 'UTC', theme: 'system', density: 'comfortable', email_notifications: true })
  const [code, setCode] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    UsersApi.me()
      .then((currentUser) => setUser(currentUser))
      .catch(() => setError('Unable to load your security settings.'))
    UsersApi.sessions()
      .then((items) => setSessions(items.filter((item) => !item.revoked_at && new Date(item.expires_at) > new Date())))
      .catch(() => setError('Unable to load your active sessions.'))
    UsersApi.getBranding()
      .then((settings) => setBranding(settings))
      .catch(() => setError('Unable to load organisation branding.'))
    UsersApi.preferences()
      .then((settings) => setPreferences(settings))
      .catch(() => setError('Unable to load your preferences.'))
  }, [])

  const revokeSession = async (id) => {
    try {
      await UsersApi.revokeSession(id)
      setSessions(sessions.filter((item) => item.id !== id))
      setMessage('Session revoked.')
    } catch (err) {
      setError(err.response?.data?.error || 'Unable to revoke session.')
    }
  }

  const saveBranding = async (event) => {
    event.preventDefault()
    try {
      const saved = await UsersApi.updateBranding(branding)
      setBranding(saved)
      setMessage('Organisation branding saved.')
    } catch (err) {
      setError(err.response?.data?.error || 'Unable to save organisation branding.')
    }
  }

  const savePreferences = async (event) => {
    event.preventDefault()
    try {
      const saved = await UsersApi.updatePreferences(preferences)
      setPreferences(saved)
      setMessage('Preferences saved.')
    } catch (err) {
      setError(err.response?.data?.error || 'Unable to save preferences.')
    }
  }

  const startSetup = async () => {
    setBusy(true)
    setError('')
    setMessage('')
    try {
      setSetup(await UsersApi.mfaSetup())
      setMessage('Scan the QR code with your authenticator app, then enter the six-digit code.')
    } catch (err) {
      setError(err.response?.data?.error || 'Unable to start MFA setup.')
    } finally {
      setBusy(false)
    }
  }

  const enableMfa = async (event) => {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const result = await UsersApi.mfaEnable(code)
      setUser({ ...user, mfa_enabled: true })
      setRecoveryCodes(result.recovery_codes || [])
      setSetup(null)
      setCode('')
      setMessage('Multi-factor authentication is now enabled for your account.')
    } catch (err) {
      setError(err.response?.data?.error || 'The authenticator code was not accepted.')
    } finally {
      setBusy(false)
    }
  }

  const disableMfa = async () => {
    setBusy(true)
    setError('')
    try {
      await UsersApi.mfaDisable(code)
      setUser({ ...user, mfa_enabled: false })
      setCode('')
      setMessage('Multi-factor authentication has been disabled.')
    } catch (err) {
      setError(err.response?.data?.error || 'The authenticator code was not accepted.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="container-fluid py-4">
      <div className="mb-4">
        <h1 className="h3 mb-1">Security settings</h1>
        <p className="text-muted mb-0">Protect your StatGate account with an authenticator app.</p>
      </div>

      {message && <div className="alert alert-success">{message}</div>}
      {error && <div className="alert alert-danger">{error}</div>}

      <section className="card shadow-sm" aria-labelledby="mfa-heading">
        <div className="card-body">
          <div className="d-flex justify-content-between align-items-start gap-3">
            <div>
              <h2 id="mfa-heading" className="h5">Multi-factor authentication</h2>
              <p className="text-muted mb-2">
                Require a time-based code after your password when signing in.
              </p>
            </div>
            <span className={`badge ${user?.mfa_enabled ? 'bg-success' : 'bg-secondary'}`}>
              {user?.mfa_enabled ? 'Enabled' : 'Not enabled'}
            </span>
          </div>

          {!user?.mfa_enabled && !setup && (
            <button type="button" className="btn btn-primary" onClick={startSetup} disabled={busy}>
              {busy ? 'Starting setup...' : 'Set up authenticator'}
            </button>
          )}

          {setup && !user?.mfa_enabled && (
            <div className="mt-3">
              <p className="mb-2"><strong>Manual setup key</strong></p>
              <code className="d-block p-2 bg-light border rounded mb-3">{setup.secret}</code>
              <p className="small text-muted">
                If your authenticator cannot scan a QR code, add the key manually. Issuer: StatGate.
              </p>
              <form onSubmit={enableMfa} className="row g-2 align-items-end">
                <div className="col-sm-5">
                  <label htmlFor="mfa-enable-code" className="form-label">Authenticator code</label>
                  <input
                    id="mfa-enable-code"
                    className="form-control"
                    inputMode="numeric"
                    pattern="[0-9]{6}"
                    maxLength="6"
                    value={code}
                    onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                    required
                  />
                </div>
                <div className="col-auto">
                  <button type="submit" className="btn btn-success" disabled={busy || code.length !== 6}>
                    Verify and enable
                  </button>
                </div>
              </form>
            </div>
          )}

          {recoveryCodes.length > 0 && <div className="alert alert-warning mt-3"><strong>Save these recovery codes now.</strong><div className="row row-cols-2 row-cols-md-5 mt-2">{recoveryCodes.map((item) => <code className="col mb-1" key={item}>{item}</code>)}</div><small>Each code works once and will not be shown again.</small></div>}

          {user?.mfa_enabled && (
            <div className="mt-3">
              <label htmlFor="mfa-disable-code" className="form-label">Enter a current code to disable MFA</label>
              <div className="d-flex gap-2 flex-wrap">
                <input
                  id="mfa-disable-code"
                  className="form-control"
                  style={{ maxWidth: '14rem' }}
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength="6"
                  value={code}
                  onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
                />
                <button type="button" className="btn btn-outline-danger" onClick={disableMfa} disabled={busy || code.length !== 6}>
                  Disable MFA
                </button>
              </div>
            </div>
          )}
        </div>
      </section>

      <section className="card shadow-sm mt-4" aria-labelledby="sessions-heading">
        <div className="card-body">
          <h2 id="sessions-heading" className="h5">Active sessions</h2>
          <p className="text-muted">Revoke sessions you no longer recognize. Device details use the request browser and a privacy-preserving network fingerprint.</p>
          {sessions.length === 0 ? <p className="mb-0 text-muted">No active refresh sessions found.</p> : <div className="table-responsive"><table className="table table-sm align-middle mb-0"><thead><tr><th>Device</th><th>Last seen</th><th>Expires</th><th /></tr></thead><tbody>{sessions.map((item) => <tr key={item.id}><td className="text-truncate" style={{ maxWidth: '22rem' }}>{item.user_agent || 'Unknown browser'}</td><td>{new Date(item.last_seen_at || item.created_at).toLocaleString()}</td><td>{new Date(item.expires_at).toLocaleString()}</td><td className="text-end"><button type="button" className="btn btn-sm btn-outline-danger" onClick={() => revokeSession(item.id)}>Revoke</button></td></tr>)}</tbody></table></div>}
        </div>
      </section>

      <section className="card shadow-sm mt-4" aria-labelledby="branding-heading">
        <div className="card-body">
          <h2 id="branding-heading" className="h5">Organisation branding</h2>
          <p className="text-muted">Branding is scoped to your authenticated organisation and is shared with supported StatGate clients.</p>
          <form onSubmit={saveBranding} className="row g-3">
            <div className="col-md-6"><label className="form-label" htmlFor="branding-name">Display name</label><input id="branding-name" className="form-control" maxLength="120" required value={branding.display_name || ''} onChange={(event) => setBranding({ ...branding, display_name: event.target.value })} /></div>
            <div className="col-md-6"><label className="form-label" htmlFor="branding-logo">Logo URL</label><input id="branding-logo" className="form-control" type="url" placeholder="https://..." value={branding.logo_url || ''} onChange={(event) => setBranding({ ...branding, logo_url: event.target.value })} /></div>
            <div className="col-md-3"><label className="form-label" htmlFor="branding-primary">Primary colour</label><input id="branding-primary" className="form-control form-control-color" type="color" value={branding.primary_color || '#0f766e'} onChange={(event) => setBranding({ ...branding, primary_color: event.target.value })} /></div>
            <div className="col-md-3"><label className="form-label" htmlFor="branding-secondary">Secondary colour</label><input id="branding-secondary" className="form-control form-control-color" type="color" value={branding.secondary_color || '#0f3d3e'} onChange={(event) => setBranding({ ...branding, secondary_color: event.target.value })} /></div>
            <div className="col-12"><button type="submit" className="btn btn-primary">Save branding</button></div>
          </form>
        </div>
      </section>

      <section className="card shadow-sm mt-4" aria-labelledby="preferences-heading">
        <div className="card-body">
          <h2 id="preferences-heading" className="h5">Personal preferences</h2>
          <p className="text-muted">Control how StatGate presents information and sends account notifications.</p>
          <form onSubmit={savePreferences} className="row g-3">
            <div className="col-md-3"><label className="form-label" htmlFor="preference-timezone">Timezone</label><input id="preference-timezone" className="form-control" maxLength="64" value={preferences.timezone} onChange={(event) => setPreferences({ ...preferences, timezone: event.target.value })} /></div>
            <div className="col-md-3"><label className="form-label" htmlFor="preference-theme">Theme</label><select id="preference-theme" className="form-select" value={preferences.theme} onChange={(event) => setPreferences({ ...preferences, theme: event.target.value })}><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></div>
            <div className="col-md-3"><label className="form-label" htmlFor="preference-density">Layout density</label><select id="preference-density" className="form-select" value={preferences.density} onChange={(event) => setPreferences({ ...preferences, density: event.target.value })}><option value="comfortable">Comfortable</option><option value="compact">Compact</option></select></div>
            <div className="col-md-3 d-flex align-items-end"><div className="form-check mb-2"><input id="preference-email" className="form-check-input" type="checkbox" checked={preferences.email_notifications} onChange={(event) => setPreferences({ ...preferences, email_notifications: event.target.checked })} /><label className="form-check-label" htmlFor="preference-email">Email notifications</label></div></div>
            <div className="col-12"><button type="submit" className="btn btn-primary">Save preferences</button></div>
          </form>
        </div>
      </section>
    </div>
  )
}

export default Settings
