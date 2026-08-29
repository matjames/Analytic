import { useState } from 'react';
import { setAuthToken } from '../api/client';

// StatChat sign-in. Authenticates against the StatGate Registry identity
// service (proxied same-origin at /registry/... -> /api/...), stores the issued
// JWT for REST + WebSocket, and mirrors the visual style of the Registry's
// auth pages (light background, sharp-cornered card, icon inputs, #165c92).
const C = {
  primary: '#165c92',
  primaryDark: '#0f4a78',
  background: '#f5f7fa',
  textDark: '#0f172a',
  textMuted: '#6b7280',
  border: '#e5e7eb',
  danger: '#dc2626',
};

const fieldBase: React.CSSProperties = {
  width: '100%',
  fontSize: '0.88rem',
  color: C.textDark,
  borderRadius: 0,
  backgroundColor: '#fff',
  outline: 'none',
  fontFamily: 'inherit',
  transition: 'border-color 0.2s, box-shadow 0.2s',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '0.8rem',
  fontWeight: 600,
  color: C.textDark,
  marginBottom: '0.3rem',
};

const togglerStyle: React.CSSProperties = {
  position: 'absolute',
  right: '0.75rem',
  top: '50%',
  transform: 'translateY(-50%)',
  background: 'none',
  border: 'none',
  color: C.textMuted,
  cursor: 'pointer',
  fontSize: '0.95rem',
  padding: '0.25rem',
};

const iconStyle: React.CSSProperties = {
  position: 'absolute',
  left: '0.85rem',
  top: '50%',
  transform: 'translateY(-50%)',
  color: C.textMuted,
  fontSize: '0.95rem',
};

const borderStyle: React.CSSProperties = {
  border: '1px solid #d1d5db',
};

export default function Login() {
  const [id, setId] = useState('');
  const [password, setPassword] = useState('');
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!id.trim() || !password) {
      setError('Enter your email/username and password.');
      return;
    }
    setBusy(true);
    try {
      const res = await fetch('/registry/users/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ emailOrUsername: id.trim(), password }),
      });
      const rawText = await res.text();
      let data: Record<string, unknown> = {};
      try {
        data = rawText ? JSON.parse(rawText) : {};
      } catch {
        data = {};
      }
      if (!res.ok || !data.token) {
        const detail = (typeof data.error === 'string' && data.error)
          ? data.error
          : (typeof data.message === 'string' && data.message)
            ? data.message
            : (rawText.trim() ? rawText.trim().slice(0, 200) : 'no detail in response');
        setError(`Sign in failed (${res.status}): ${detail}`);
        return;
      }
      if (data.email_verification_required) {
        setError('Your email is not verified. Check your inbox for a verification link.');
        return;
      }
      setAuthToken(data.token as string);
      window.location.reload();
    } catch (err) {
      setError(`Could not reach the identity service (${err instanceof Error ? err.message : 'network error'}).`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '1rem',
        background: C.background,
        fontFamily: "'Inter', sans-serif",
      }}
    >
      <div style={{ width: '100%', maxWidth: 380 }}>
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: '1rem' }}>
          <img
            src="/statgate-logo.png"
            alt="StatGate logo"
            style={{
              width: 60,
              height: 60,
              objectFit: 'contain',
              margin: '0 auto 0.5rem',
              display: 'block',
            }}
          />
          <h1 style={{ fontSize: '1.2rem', fontWeight: 700, color: C.textDark, margin: 0, lineHeight: 1.2 }}>
            StatGate Analytics
          </h1>
          <p style={{ fontSize: '0.8rem', color: C.textMuted, margin: '0.2rem 0 0' }}>
            Enterprise Communication &amp; Collaboration
          </p>
        </div>

        {/* Card */}
        <form
          onSubmit={handleSubmit}
          style={{
            background: '#fff',
            padding: '1.6rem',
            boxShadow: '0 2px 8px rgba(0,0,0,0.10)',
            borderRadius: 0,
            border: '1px solid ' + C.border,
          }}
        >
          <h2 style={{ fontSize: '1.1rem', fontWeight: 700, color: C.textDark, margin: '0 0 1.25rem', textAlign: 'center' }}>
            Account Sign In
          </h2>

          {/* Email or Username */}
          <div style={{ marginBottom: '0.9rem' }}>
            <label style={labelStyle}>Email or Username</label>
            <div style={{ position: 'relative' }}>
              <i className="bi bi-person" style={iconStyle} />
              <input
                style={{ ...fieldBase, ...borderStyle, paddingLeft: '2.45rem' }}
                value={id}
                onChange={(e) => setId(e.target.value)}
                placeholder="Enter your email or username"
                autoComplete="username"
                autoFocus
                aria-label="Email or username"
              />
            </div>
          </div>

          {/* Password */}
          <div style={{ marginBottom: '1rem' }}>
            <label style={labelStyle}>Password</label>
            <div style={{ position: 'relative' }}>
              <i className="bi bi-lock" style={iconStyle} />
              <input
                style={{ ...fieldBase, ...borderStyle, paddingLeft: '2.45rem' }}
                type={showPw ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password"
                autoComplete="current-password"
                aria-label="Password"
              />
              <button type="button" style={togglerStyle} onClick={() => setShowPw((s) => !s)} aria-label="Toggle password visibility">
                <i className={showPw ? 'bi bi-eye-slash' : 'bi bi-eye'} />
              </button>
            </div>
          </div>

          {/* Options */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', fontSize: '0.8rem' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', cursor: 'pointer', color: C.textDark }}>
              <input type="checkbox" style={{ width: 15, height: 15, cursor: 'pointer', accentColor: C.primary }} />
              Remember me
            </label>
            <span style={{ color: C.primary, cursor: 'pointer', fontWeight: 500 }}>Forgot password?</span>
          </div>

          {error && (
            <div
              style={{
                background: '#fee2e2',
                color: C.danger,
                borderLeft: '3px solid ' + C.danger,
                padding: '0.6rem 0.75rem',
                marginBottom: '0.9rem',
                fontSize: '0.8rem',
              }}
            >
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={busy}
            style={{
              width: '100%',
              padding: '0.7rem',
              background: C.primary,
              color: '#fff',
              border: 'none',
              borderRadius: 0,
              fontSize: '0.95rem',
              fontWeight: 600,
              cursor: busy ? 'not-allowed' : 'pointer',
              opacity: busy ? 0.7 : 1,
              fontFamily: 'inherit',
              transition: 'background 0.2s',
            }}
          >
            {busy ? 'Signing in…' : 'Sign In'}
          </button>
        </form>

        <p style={{ textAlign: 'center', color: C.textMuted, fontSize: '0.8rem', marginTop: '1.25rem' }}>
          © {new Date().getFullYear()} StatGate. {''}
          <span style={{ color: C.primary, fontWeight: 600 }}>Field Operations Registry</span>
        </p>
      </div>
    </div>
  );
}
