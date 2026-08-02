import { Conversation } from '../types';

interface Props {
  conversations: Conversation[];
  currentConversation: string;
  onSelect: (conversationId: string) => void;
  theme: 'light' | 'dark';
  isMobile: boolean;
}

export default function ConversationList({ conversations, currentConversation, onSelect, theme, isMobile }: Props) {
  return (
    <aside
      style={{
        width: isMobile ? '100%' : 280,
        minWidth: isMobile ? 'auto' : 240,
        maxHeight: 'calc(100vh - 190px)',
        overflowY: 'auto',
        background: theme === 'dark' ? '#0f3f5f' : '#ffffff',
        border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
        borderRadius: 16,
        padding: 18,
      }}
    >
      <h2 style={{ marginTop: 0 }}>Conversations</h2>
      <ul style={{ padding: 0, listStyle: 'none' }}>
        {conversations.map((conversation) => (
          <li key={conversation.id} style={{ marginBottom: 12 }}>
            <button
              type="button"
              onClick={() => onSelect(conversation.id)}
              style={{
                width: '100%',
                padding: '12px 14px',
                borderRadius: 12,
                border: 'none',
                background: conversation.id === currentConversation ? '#165c92' : theme === 'dark' ? '#0f3f5f' : '#f9fafb',
                color: conversation.id === currentConversation ? '#fff' : theme === 'dark' ? '#e8eef4' : '#1a1a1a',
                textAlign: 'left',
                cursor: 'pointer',
              }}
            >
              <div style={{ fontWeight: 700 }}>{conversation.name}</div>
              <div style={{ fontSize: 12, marginTop: 4, color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>
                {conversation.type === 'direct' ? 'Direct message' : 'Channel'}
              </div>
            </button>
          </li>
        ))}
      </ul>
    </aside>
  );
}
