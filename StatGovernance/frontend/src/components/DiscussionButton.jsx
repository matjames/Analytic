import React, { useState } from 'react';

export default function DiscussionButton({ apiBase, token, objectType, objectId, name }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const openDiscussion = async (event) => {
    event.stopPropagation();
    setBusy(true);
    setError('');
    try {
      const response = await fetch(`${apiBase}/api/discussions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ object_type: objectType, object_id: objectId, name }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok || !data.object_ref) {
        throw new Error(data.error || 'Discussion is unavailable');
      }
      const chatBase = import.meta.env.VITE_STATCHAT_URL || 'http://localhost:3009';
      const chatURL = new URL(chatBase);
      chatURL.searchParams.set('objectRef', data.object_ref);
      if (token) chatURL.searchParams.set('statgate_token', token);
      window.open(chatURL.toString(), '_blank', 'noopener,noreferrer');
    } catch (err) {
      setError(err.message || 'Discussion is unavailable');
    } finally {
      setBusy(false);
    }
  };

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <button
        type="button"
        className="btn btn-secondary btn-sm"
        onClick={openDiscussion}
        disabled={busy}
        title={error || 'Open the tenant-scoped StatChat discussion'}
      >
        {busy ? 'Opening...' : 'Discuss'}
      </button>
      {error && <span style={{ color: '#b42318', fontSize: 11 }}>{error}</span>}
    </span>
  );
}
