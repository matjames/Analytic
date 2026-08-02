import { Conversation, Message } from '../types';

interface Props {
  conversation?: Conversation;
  messages: Message[];
  draft: string;
  theme: 'light' | 'dark';
  isMobile: boolean;
  onDraftChange: (value: string) => void;
  onSend: () => void;
}

export default function MessagePanel({
  conversation,
  messages,
  draft,
  theme,
  isMobile,
  onDraftChange,
  onSend,
}: Props) {
  return (
    <main
      style={{
        flex: 1,
        background: theme === 'dark' ? '#0f3f5f' : '#ffffff',
        border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
        borderRadius: 16,
        padding: 20,
        minHeight: isMobile ? 'auto' : 'calc(100vh - 190px)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <div>
          <h2 style={{ margin: 0 }}>{conversation?.name ?? `#${conversation?.id ?? 'unknown'}`}</h2>
          <p style={{ margin: '8px 0 0', color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>
            {conversation?.type === 'direct' ? 'Direct message view' : 'Channel chat view'}
          </p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button type="button" disabled style={{ padding: '10px 16px', borderRadius: 10, border: 'none', background: theme === 'dark' ? '#0f3f5f' : '#e5e7eb', color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
            Workspace
          </button>
          <button type="button" disabled style={{ padding: '10px 16px', borderRadius: 10, border: 'none', background: theme === 'dark' ? '#0f3f5f' : '#e5e7eb', color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
            Members
          </button>
        </div>
      </div>

      <section
        style={{
          marginTop: 16,
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          padding: 18,
          borderRadius: 16,
          background: theme === 'dark' ? '#0a2b45' : '#f9fafb',
          border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
        }}
      >
        {messages.map((message) => (
          <article key={message.id} style={{ marginBottom: 18, paddingBottom: 8, borderBottom: `1px solid ${theme === 'dark' ? '#1f2937' : '#e5e7eb'}` }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}> 
              <span style={{ fontWeight: 700 }}>{message.sender}</span>
              <span style={{ fontSize: 12, color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>
                {new Date(message.createdAt).toLocaleTimeString()}
              </span>
            </div>
            <p style={{ margin: '8px 0 0', lineHeight: 1.6 }}>{message.text}</p>
          </article>
        ))}
      </section>

      <footer style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
        <input
          value={draft}
          onChange={(event) => onDraftChange(event.target.value)}
          placeholder="Type a message..."
          onKeyDown={(event) => {
            if (event.key === 'Enter') onSend();
          }}
          style={{
            flex: 1,
            minWidth: 0,
            padding: 14,
            borderRadius: 12,
            border: `1px solid ${theme === 'dark' ? '#6b7280' : '#d1d5db'}`,
            background: theme === 'dark' ? '#0a2b45' : '#ffffff',
            color: theme === 'dark' ? '#e8eef4' : '#1a1a1a',
          }}
        />
        <button type="button" onClick={onSend} style={{ padding: '14px 20px', borderRadius: 12, border: 'none', background: '#165c92', color: '#ffffff', cursor: 'pointer' }}>
          Send
        </button>
      </footer>
    </main>
  );
}
