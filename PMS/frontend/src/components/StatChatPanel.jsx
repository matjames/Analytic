import React, { useState, useEffect, useRef } from 'react';

export default function StatChatPanel({ chats, onSendMessage, projectId }) {
  const [activeChannel, setActiveChannel] = useState('general');
  const [text, setText] = useState('');
  const chatBottomRef = useRef(null);

  const channels = ['general', 'announcements', 'meetings'];
  const statChatURL = import.meta.env.VITE_STATCHAT_UI_URL || 'http://localhost:3009';
  const discussionURL = `${statChatURL}/?objectRef=${encodeURIComponent(`obj:pms:project:${projectId}`)}`;

  const filteredChats = chats ? chats.filter(c => c.channel === activeChannel) : [];

  useEffect(() => {
    if (chatBottomRef.current) {
      chatBottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [chats, activeChannel]);

  const handleSend = (e) => {
    e.preventDefault();
    if (!text.trim()) return;

    onSendMessage({
      projectId,
      channel: activeChannel,
      sender: "StatGate Operator",
      role: "System Administrator",
      message: text
    });
    setText('');
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Open in StatChat banner */}
      <div className="glass-panel" style={{ padding: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h3 style={{ fontSize: '15px', fontWeight: '700' }}>💬 StatChat Integration</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            Project communication is connected to the enterprise StatChat platform for real-time collaboration.
          </p>
        </div>
        <a href={discussionURL} target="_blank" rel="noopener noreferrer" className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }}>
          Open Full StatChat ↗
        </a>
      </div>

      <div className="glass-panel" style={{ display: 'grid', gridTemplateColumns: '200px 1fr', height: '500px', overflow: 'hidden' }}>
        {/* Channels List */}
        <div style={{ borderRight: '1px solid var(--border-light)', padding: '16px', display: 'flex', flexDirection: 'column', gap: '8px', background: 'var(--bg-light)' }}>
          <h4 style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-muted)', letterSpacing: '1px', marginBottom: '8px' }}>Channels</h4>
          {channels.map(ch => (
            <div
              key={ch}
              onClick={() => setActiveChannel(ch)}
              style={{
                padding: '8px 12px',
                borderRadius: '6px',
                cursor: 'pointer',
                fontSize: '13px',
                fontWeight: activeChannel === ch ? '700' : '500',
                color: activeChannel === ch ? 'var(--primary-color)' : 'var(--text-secondary)',
                background: activeChannel === ch ? 'rgba(22,92,146,0.08)' : 'transparent',
                transition: 'all 0.2s ease'
              }}
            >
              # {ch}
            </div>
          ))}
        </div>

        {/* Chat Window */}
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          {/* Channel Header */}
          <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border-light)', fontSize: '14px', fontWeight: '700', color: 'var(--text-dark)' }}>
            # {activeChannel}
          </div>

          {/* Message Feed */}
          <div style={{ flexGrow: 1, padding: '20px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {filteredChats.length === 0 ? (
              <div style={{ textAlign: 'center', color: 'var(--text-muted)', fontSize: '13px', marginTop: '40px' }}>
                No messages in #{activeChannel} yet. Start the conversation!
              </div>
            ) : (
              filteredChats.map(chat => (
                <div key={chat.id} style={{ display: 'flex', gap: '12px' }}>
                  <div style={{
                    width: '32px',
                    height: '32px',
                    borderRadius: '50%',
                    background: 'rgba(22,92,146,0.1)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '14px'
                  }}>
                    👤
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <span style={{ fontWeight: '700', fontSize: '13px', color: 'var(--text-dark)' }}>{chat.sender}</span>
                      <span className="badge" style={{ fontSize: '8px', padding: '1px 6px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{chat.role}</span>
                      <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
                        {new Date(chat.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </span>
                    </div>
                    <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{chat.message}</span>
                  </div>
                </div>
              ))
            )}
            <div ref={chatBottomRef} />
          </div>

          {/* Text Area */}
          <form onSubmit={handleSend} style={{ padding: '16px', borderTop: '1px solid var(--border-light)', display: 'flex', gap: '12px' }}>
            <input
              type="text"
              placeholder={`Message #${activeChannel}...`}
              value={text}
              onChange={e => setText(e.target.value)}
              style={{
                flexGrow: 1,
                padding: '10px 16px',
                borderRadius: '6px',
                background: 'var(--bg-light)',
                border: '1px solid var(--border-light)',
                color: 'var(--text-dark)',
                fontSize: '13px',
                outline: 'none'
              }}
            />
            <button type="submit" className="btn btn-primary" style={{ padding: '10px 16px' }}>Send</button>
          </form>
        </div>
      </div>
    </div>
  );
}
