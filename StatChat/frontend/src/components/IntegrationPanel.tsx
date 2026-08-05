import { useState } from 'react';
import type { SidebarView } from '../App';
import { Conversation } from '../types';
import GoogleMeetProvider from './meeting-providers/GoogleMeetProvider';
import TeamsProvider from './meeting-providers/TeamsProvider';
import ZoomProvider from './meeting-providers/ZoomProvider';

interface Props {
  activeView: SidebarView;
  conversation?: Conversation;
  conversations: Conversation[];
  theme: 'light' | 'dark';
  isMobile: boolean;
  onSelectConversation: (conversationId: string) => void;
}

const contentByView: Record<SidebarView, { title: string; description: string; actionLabel?: string; url?: string }> = {
  chat: {
    title: 'Chat Workspace',
    description: 'A dedicated WhatsApp-style workspace for direct conversations and channels.',
    actionLabel: 'Open Chat',
  },
  directMessages: {
    title: 'Direct Messages',
    description: 'Start or continue private conversations with your team members.',
    actionLabel: 'Open DMs',
  },
  teams: {
    title: 'Team Spaces',
    description: 'Centralize team collaboration and shared channels inside StatGate.',
    actionLabel: 'Open Teams',
  },
  channels: {
    title: 'Channels',
    description: 'Public and private channels for topic-based discussion.',
    actionLabel: 'Browse Channels',
  },
  meetings: {
    title: 'Meetings',
    description: 'Schedule and launch instant meetings from within StatChat.',
    actionLabel: 'View Meetings',
  },
  wellness: {
    title: 'Wellness',
    description: 'Mental health awareness and developmental ideas — inspired by X and TikTok.',
    actionLabel: 'Open Wellness',
  },
  knowledge: {
    title: 'Knowledge',
    description: 'Where knowledge meets professionals — talented people teach, share ideas, and provide knowledge in statistics, analysis, research, and development.',
    actionLabel: 'Open Knowledge',
  },
  projects: {
    title: 'Projects',
    description: 'Create project collaboration spaces and link chat threads to milestones.',
    actionLabel: 'Open Projects',
  },
  research: {
    title: 'Research Spaces',
    description: 'Discuss datasets and reports inside dedicated research rooms.',
    actionLabel: 'Open Research',
  },
  calendar: {
    title: 'Calendar',
    description: 'View upcoming meetings, events, and task deadlines.',
    actionLabel: 'Open Calendar',
  },
  tasks: {
    title: 'Tasks',
    description: 'Create tasks directly from chat threads and assign priorities.',
    actionLabel: 'Open Tasks',
  },
  contacts: {
    title: 'Contacts',
    description: 'Browse team members, direct message contacts, and guest users.',
    actionLabel: 'Open Contacts',
  },
  communities: {
    title: 'Communities',
    description: 'Public and private communities for the organization.',
    actionLabel: 'Open Communities',
  },
  settings: {
    title: 'Settings',
    description: 'Manage your profile, notifications, appearance, and privacy.',
    actionLabel: 'Open Settings',
  },
};

export default function IntegrationPanel({ activeView, conversation, conversations, theme, isMobile, onSelectConversation }: Props) {
  const content = contentByView[activeView];
  const panelBackground = theme === 'dark' ? '#0f3f5f' : '#ffffff';
  const borderColor = theme === 'dark' ? '#6b7280' : '#e5e7eb';
  const textColor = theme === 'dark' ? '#e8eef4' : '#1a1a1a';
  const subViewLabel = content.actionLabel ? content.actionLabel.replace(/^(Open|View)\s+/i, '') : content.title;
  const [connectedProviders, setConnectedProviders] = useState({
    meet: false,
    teams: false,
    zoom: false,
  });

  const handleProviderToggle = (provider: 'meet' | 'teams' | 'zoom') => {
    setConnectedProviders((prev) => ({ ...prev, [provider]: !prev[provider] }));
  };

  const providerPanel = (
    <div style={{ display: 'grid', gap: 14, marginTop: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
        <div>
          <h3 style={{ margin: '0 0 8px', color: textColor }}>Connect meeting services</h3>
          <p style={{ margin: 0, color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>
            Enable one-click provider access for Google Meet, Teams, and Zoom.
          </p>
        </div>
      </div>
      <div style={{ display: 'grid', gap: 12, gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
        <GoogleMeetProvider
          connected={connectedProviders.meet}
          onConnect={() => handleProviderToggle('meet')}
          theme={theme}
        />
        <TeamsProvider
          connected={connectedProviders.teams}
          onConnect={() => handleProviderToggle('teams')}
          theme={theme}
        />
        <ZoomProvider
          connected={connectedProviders.zoom}
          onConnect={() => handleProviderToggle('zoom')}
          theme={theme}
        />
      </div>
    </div>
  );

  if (['directMessages', 'teams', 'channels'].includes(activeView)) {
    return (
      <section
        style={{
          flex: 1,
          minHeight: isMobile ? 'auto' : 'calc(100vh - 190px)',
          background: panelBackground,
          border: `1px solid ${borderColor}`,
          borderRadius: 24,
          padding: 24,
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'center', marginBottom: 8 }}>
            <h2 style={{ margin: 0, color: textColor }}>{content.title}</h2>
            <span style={{ padding: '6px 10px', borderRadius: 999, background: theme === 'dark' ? '#0a2b45' : '#f0f7fb', color: theme === 'dark' ? '#6ba3c3' : '#165c92', fontSize: 12, fontWeight: 700 }}>
            {subViewLabel}
          </span>
        </div>
        <p style={{ margin: '10px 0 0', color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>{content.description}</p>
      </div>
      <div style={{ display: 'grid', gap: 12 }}>
        {conversations.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => onSelectConversation(item.id)}
              style={{
                width: '100%',
                padding: 18,
                borderRadius: 18,
                border: `1px solid ${borderColor}`,
                background: theme === 'dark' ? '#0a2b45' : '#f9fafb',
                color: textColor,
                textAlign: 'left',
                cursor: 'pointer',
              }}
            >
              <strong>{item.name}</strong>
              <div style={{ marginTop: 4, color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>
                {item.type === 'direct' ? 'Direct message' : 'Channel'}
              </div>
            </button>
          ))}
        </div>
      </section>
    );
  }

  return (
    <section
      style={{
        flex: 1,
        minHeight: isMobile ? 'auto' : 'calc(100vh - 190px)',
        background: panelBackground,
        border: `1px solid ${borderColor}`,
        borderRadius: 24,
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 20,
      }}
    >
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'center', marginBottom: 8 }}>
          <h2 style={{ margin: 0, color: textColor }}>{content.title}</h2>
          <span style={{ padding: '6px 10px', borderRadius: 999, background: theme === 'dark' ? '#0a2b45' : '#f0f7fb', color: theme === 'dark' ? '#6ba3c3' : '#165c92', fontSize: 12, fontWeight: 700 }}>
            {subViewLabel}
          </span>
        </div>
        <p style={{ margin: '10px 0 0', color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>{content.description}</p>
      </div>

      {content.url ? (
        <div style={{ display: 'grid', gap: 18 }}>
          <a
            href={content.url}
            target="_blank"
            rel="noreferrer"
            style={{
              display: 'inline-block',
              padding: '16px 20px',
              borderRadius: 16,
              background: theme === 'dark' ? '#165c92' : '#165c92',
              color: '#ffffff',
              textDecoration: 'none',
              fontWeight: 700,
              width: 'fit-content',
            }}
          >
            {content.actionLabel}
          </a>
          <div style={{ display: 'grid', gap: 12 }}>
            {Object.entries(contentByView)
              .filter(([key]) => ['linkedin', 'x', 'zoom', 'meet', 'msteams'].includes(key))
              .map(([key, card]) => (
                <a
                  key={key}
                  href={card.url}
                  target="_blank"
                  rel="noreferrer"
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: 16,
                    borderRadius: 16,
                    border: `1px solid ${borderColor}`,
                    background: theme === 'dark' ? '#0a2b45' : '#f9fafb',
                    color: textColor,
                    textDecoration: 'none',
                  }}
                >
                  <span>{card.title}</span>
                  <span style={{ opacity: 0.7 }}>↗</span>
                </a>
              ))}
          </div>
          {providerPanel}
        </div>
      ) : (
        <div style={{ display: 'grid', gap: 14 }}>
          <div style={{ display: 'grid', gap: 10 }}>
            <span style={{ color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>This workspace is under active development to support native enterprise collaboration features.</span>
            <button
              type="button"
              style={{
                padding: '14px 18px',
                borderRadius: 16,
                border: 'none',
                background: theme === 'dark' ? '#165c92' : '#165c92',
                color: '#fff',
                cursor: 'pointer',
                width: 'fit-content',
              }}
            >
              {content.actionLabel}
            </button>
          </div>
          <div style={{ display: 'grid', gap: 12 }}>
            <h3 style={{ margin: 0, color: textColor }}>Workspace Highlights</h3>
            <div style={{ display: 'grid', gap: 10 }}>
              <div style={{ padding: 16, borderRadius: 16, background: theme === 'dark' ? '#0a2b45' : '#f9fafb', border: `1px solid ${borderColor}` }}>
                <strong>Action items</strong>
                <p style={{ margin: '8px 0 0', color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>Create tasks, allocate teams, and manage project milestones from every chat.</p>
              </div>
              <div style={{ padding: 16, borderRadius: 16, background: theme === 'dark' ? '#0a2b45' : '#f9fafb', border: `1px solid ${borderColor}` }}>
                <strong>Collaboration</strong>
                <p style={{ margin: '8px 0 0', color: theme === 'dark' ? '#9bc4d8' : '#6b7280' }}>Bring meetings, documents, and research spaces together in a single enterprise workspace.</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
