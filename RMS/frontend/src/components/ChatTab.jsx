import React, { useState, useEffect, useRef } from 'react';

export default function ChatTab({ messages, researchId, onRefresh }) {
  const [newMessage, setNewMessage] = useState('');
  const [channel, setChannel] = useState('general');
  const messagesEndRef = useRef(null);
  const statChatURL = import.meta.env.VITE_STATCHAT_UI_URL || 'http://localhost:3009';
  const discussionURL = `${statChatURL}/?objectRef=${encodeURIComponent(`obj:rms:research:${researchId}`)}`;

  const handleSend = async (e) => {
    e.preventDefault();
    if (!newMessage.trim()) return;
    await fetch(`${API_URL}/api/chats`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ researchId, sender: 'Admin User', channel, message: newMessage })
    });
    setNewMessage('');
    onRefresh();
  };

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 200px)' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '16px' }}>
        <div>
        <h3>Research Collaboration Chat</h3>
        <p style={{ color: 'var(--text-secondary)', fontSize: '14px', marginTop: '4px' }}>
          Team discussions, announcements, and coordination
        </p>
        </div>
        <a href={discussionURL} target="_blank" rel="noopener noreferrer" className="btn btn-primary">Open Full StatChat</a>
      </div>

      <div className="glass-panel" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Channel selector */}
        <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border-light)', display: 'flex', gap: '8px' }}>
          {['general', 'announcements', 'methodology', 'field-ops'].map(ch => (
            <button
              key={ch}
              onClick={() => setChannel(ch)}
              style={{
                padding: '4px 12px', borderRadius: '4px', border: 'none', cursor: 'pointer',
                background: channel === ch ? 'var(--primary)' : '#f1f3f4',
                color: channel === ch ? '#fff' : 'var(--text)',
                fontSize: '12px', fontWeight: '600'
              }}
            >
              #{ch}
            </button>
          ))}
        </div>

        {/* Messages area */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {messages.length === 0 ? (
            <p style={{ color: 'var(--text-secondary)', textAlign: 'center', marginTop: '40px' }}>
              No messages in #{channel} yet.
            </p>
          ) : (
            messages.filter(m => m.channel === channel).map(msg => (
              <div key={msg.id} style={{ maxWidth: '70%', padding: '10px 14px', borderRadius: '12px', background: '#f1f3f4', alignSelf: 'flex-start' }}>
                <div style={{ fontSize: '12px', fontWeight: '700', marginBottom: '4px', color: 'var(--primary)' }}>{msg.sender}</div>
                <div style={{ fontSize: '14px', lineHeight: '1.4' }}>{msg.message}</div>
                <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px', textAlign: 'right' }}>
                  {new Date(msg.createdTime).toLocaleString()}
                </div>
              </div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Message input */}
        <form onSubmit={handleSend} style={{ padding: '12px 16px', borderTop: '1px solid var(--border-light)', display: 'flex', gap: '8px' }}>
          <input
            type="text"
            placeholder={`Message #${channel}`}
            value={newMessage}
            onChange={e => setNewMessage(e.target.value)}
            style={{ flex: 1, padding: '8px 12px', borderRadius: '20px', border: '1px solid var(--border-light)', outline: 'none' }}
          />
          <button type="submit" className="btn btn-primary" style={{ borderRadius: '20px', padding: '8px 20px' }}>Send</button>
        </form>
      </div>
    </div>
  );
}
