import { useEffect, useMemo, useRef, useState } from 'react';
import { fetchConversations, fetchCurrentUser, fetchMessages, sendChatMessage, uploadChatAttachment, fetchNotifications, markAllNotificationsRead, updatePresence, fetchUserSettings, fetchMeetingCallSession, WS_URL } from './api/client';
import { Conversation, Message, User, Notification } from './types';
import ChatSidebarList from './components/ChatSidebarList';
import ChatWorkspace from './components/ChatWorkspace';
import SidebarRail from './components/SidebarRail';
import IntegrationPanel from './components/IntegrationPanel';
import SettingsPanel from './components/SettingsPanel';
import CollaborationPanel from './components/CollaborationPanel';
import CalendarPanel from './components/CalendarPanel';
import CallOverlay from './components/CallOverlay';
import { useCall } from './hooks/useCall';
import WellnessPanel from './components/WellnessPanel';
import KnowledgePanel from './components/KnowledgePanel';
import LegalPanel, { LegalTab } from './components/LegalPanel';
import styles from './App.module.css';

export type SidebarView =
  | 'chat'
  | 'directMessages'
  | 'teams'
  | 'channels'
  | 'meetings'
  | 'wellness'
  | 'knowledge'
  | 'projects'
  | 'research'
  | 'calendar'
  | 'tasks'
  | 'contacts'
  | 'communities'
  | 'settings';

export type SubHeaderView = 'home' | 'connect' | 'opportunities' | 'network' | 'jobs';

export type CalendarSubView = 'overview' | 'schedule' | 'rooms' | 'recordings';

const subHeaderTabs: Array<{ id: SubHeaderView; label: string }> = [
  { id: 'home', label: 'Home' },
  { id: 'connect', label: 'Connect' },
  { id: 'opportunities', label: 'Opportunities' },
  { id: 'network', label: 'My Network' },
  { id: 'jobs', label: 'Jobs' },
];

const calendarTabs: Array<{ id: CalendarSubView; label: string }> = [
  { id: 'overview', label: 'Overview' },
  { id: 'schedule', label: 'Schedule' },
  { id: 'rooms', label: 'Meeting Rooms' },
  { id: 'recordings', label: 'Recordings' },
];

export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [currentConversation, setCurrentConversation] = useState('general');
  const [messages, setMessages] = useState<Message[]>([]);
  const [draft, setDraft] = useState('');
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [error, setError] = useState<string | null>(null);
  const [isMobile, setIsMobile] = useState(false);
  const [mobileView, setMobileView] = useState<'list' | 'chat'>('list');
  const [activeSidebarView, setActiveSidebarView] = useState<SidebarView>('directMessages');
  const [activeSubView, setActiveSubView] = useState<SubHeaderView>('home');
  const [activeCalendarView, setActiveCalendarView] = useState<CalendarSubView>('overview');
  const [showAppLauncher, setShowAppLauncher] = useState(false);
  const [legalPanelOpen, setLegalPanelOpen] = useState(false);
  const [legalPanelTab, setLegalPanelTab] = useState<LegalTab>('privacy');
const [notifications, setNotifications] = useState<Notification[]>([]);
  const [showNotifications, setShowNotifications] = useState(false);
  const [globalSearch, setGlobalSearch] = useState('');
  const [meetingCallRemoteStreams, setMeetingCallRemoteStreams] = useState<Record<string, MediaStream>>({});
  const currentConversationRef = useRef(currentConversation);
  const notificationsRef = useRef<HTMLDivElement | null>(null);

  const activeConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === currentConversation),
    [conversations, currentConversation]
  );

  const {
    session: meetingSession,
    participants: meetingParticipants,
    localStream: meetingLocalStream,
    micMuted: meetingMicMuted,
    cameraOff: meetingCameraOff,
    connecting: meetingConnecting,
    error: meetingError,
    startCall: startMeetingCall,
    joinCall: joinMeetingCall,
    toggleMute: toggleMeetingMute,
    toggleCamera: toggleMeetingCamera,
    hangUp: hangUpMeeting,
    endCall: endMeeting,
  } = useCall({
    user: user ? { id: user.id, name: user.name } : null,
    onIncomingRemoteStream: (stream, userId) => {
      setMeetingCallRemoteStreams((prev) => ({ ...prev, [userId]: stream }));
    },
    onRemoteLeave: (userId) => {
      setMeetingCallRemoteStreams((prev) => {
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    },
    onCallEnded: () => {
      setMeetingCallRemoteStreams({});
    },
  });

  const handleSelectConversation = (conversationId: string) => {
    setCurrentConversation(conversationId);
    if (isMobile) {
      setMobileView('chat');
    }
  };

  const closeMobileChat = () => setMobileView('list');

  const handleConversationCreated = (conv: Conversation) => {
    setConversations((prev) => {
      if (prev.some((c) => c.id === conv.id)) return prev;
      return [...prev, conv];
    });
  };

  const handleJoinMeeting = async (meetingId: string) => {
    try {
      const session = await fetchMeetingCallSession(meetingId);
      await joinMeetingCall(session.id);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to join meeting';
      setError(`Meeting join failed: ${message}`);
      throw error;
    }
  };

  const isChatWorkspace = ['directMessages', 'teams'].includes(activeSidebarView);
  const isSettingsView = activeSidebarView === 'settings';
  const isCollaborationView = activeSidebarView === 'channels';
  const isCalendarView = activeSidebarView === 'calendar' || activeSidebarView === 'meetings';
  const isWellnessView = activeSidebarView === 'wellness';
  const isKnowledgeView = activeSidebarView === 'knowledge';
  const isNativeMeetingView = activeSidebarView === 'meetings';
  const chatViewType: 'direct' | 'group' | 'channel' =
    activeSidebarView === 'directMessages' ? 'direct'
    : activeSidebarView === 'teams' ? 'group'
    : 'channel';

  const sidebarWidth = 68;

  useEffect(() => {
    document.body.style.margin = '0';

    async function loadInitialData() {
      try {
        const [currentUser, conversationList, initialMessages, userSettings] = await Promise.all([
          fetchCurrentUser(),
          fetchConversations(),
          fetchMessages(currentConversation),
          fetchUserSettings(),
        ]);
        setUser(currentUser);
        setConversations(conversationList);
        if (userSettings.theme === 'dark' || userSettings.theme === 'light') {
          setTheme(userSettings.theme);
        }
        if (!conversationList.some((conversation: Conversation) => conversation.id === currentConversation)) {
          setCurrentConversation(conversationList[0]?.id ?? currentConversation);
        }
        setMessages(initialMessages);
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        console.error('Failed to load initial data', error);
        setError(`Unable to reach StatChat right now: ${message}`);
      }
    }

    loadInitialData();

    const ws = new WebSocket(WS_URL);
    const tenantId = 'statgate-uganda';
    const joinConversation = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ action: 'join-conversation', conversationId: currentConversationRef.current, tenantId }));
      }
    };

    ws.addEventListener('open', () => {
      setConnected(true);
      joinConversation();
    });
    ws.addEventListener('close', () => setConnected(false));
    ws.addEventListener('error', () => setError('Live connection dropped. The UI will retry when the backend is reachable again.'));
    ws.addEventListener('message', (event) => {
      let data: unknown;
      try {
        data = JSON.parse(event.data);
      } catch {
        return;
      }
      // GatewayEnvelope events (typing, presence, call state, etc.) carry an
      // `event` field and must NOT be treated as chat messages.
      if (data && typeof data === 'object' && 'event' in (data as Record<string, unknown>)) {
        return;
      }
      const message = data as Message;
      if (!message || typeof message.id !== 'string' || typeof message.conversationId !== 'string') {
        return;
      }
      setMessages((prev) => {
        if (message.conversationId !== currentConversationRef.current) {
          return prev;
        }
        if (prev.some((m) => m.id === message.id)) {
          return prev;
        }
        return [...prev, message];
      });
    });

    setSocket(ws);

    const handleResize = () => setIsMobile(window.innerWidth < 900);
    handleResize();
    window.addEventListener('resize', handleResize);

    return () => {
      ws.close();
      window.removeEventListener('resize', handleResize);
      document.body.style.margin = '';
    };
  }, []);

  // Load notifications and set presence on mount
  useEffect(() => {
    fetchNotifications()
      .then((notifs) => setNotifications(notifs))
      .catch(() => {});
    updatePresence('online').catch(() => {});
  }, []);

  // Close notifications dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (notificationsRef.current && !notificationsRef.current.contains(event.target as Node)) {
        setShowNotifications(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const unreadCount = notifications.filter((n) => !n.read).length;

  const handleNotificationsClick = () => {
    setShowNotifications((prev) => !prev);
    if (!showNotifications && unreadCount > 0) {
      markAllNotificationsRead()
        .then(() => {
          setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
        })
        .catch(() => {});
    }
  };

  useEffect(() => {
    const activeConversationId = currentConversation || 'general';
    currentConversationRef.current = activeConversationId;
    if (!socket) return;

    if (socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ action: 'join-conversation', conversationId: activeConversationId, tenantId: 'statgate-uganda' }));
    }

    let isCancelled = false;
    fetchMessages(activeConversationId)
      .then((nextMessages: Message[]) => {
        if (!isCancelled && currentConversationRef.current === activeConversationId) {
          setMessages(nextMessages);
        }
      })
      .catch((error: unknown) => {
        console.error('Failed to load conversation messages', error);
      });

    return () => {
      isCancelled = true;
    };
  }, [currentConversation, socket]);

  const sendMessage = async (
    textOverride?: string,
    replyContext?: { parentMessageId?: string; threadRootId?: string }
  ) => {
    const messageText = (textOverride ?? draft).trim();
    if (!messageText) return;

    const channelId = activeConversation?.type === 'channel' ? currentConversation : undefined;
    const payload = {
      conversationId: currentConversation,
      channelId,
      sender: user?.name ?? 'StatChat User',
      text: messageText,
      tenantId: user?.organizationId ?? 'statgate-uganda',
      parentMessageId: replyContext?.parentMessageId,
      threadRootId: replyContext?.threadRootId,
    };

    setDraft('');

    try {
      const createdMessage = await sendChatMessage(payload);
      setMessages((prev) => {
        if (prev.some((message) => message.id === createdMessage.id)) {
          return prev;
        }
        return [...prev, createdMessage];
      });
      setError(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      console.error('Failed to send message', error);
      setError(`Unable to send message: ${message}`);
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ action: 'send-message', ...payload }));
      }
    }
  };

const handleMessageUpdate = (updated: Message) => {
    setMessages((prev) => prev.map((m) => (m.id === updated.id ? updated : m)));
  };

  const handleMessageDelete = (messageId: string) => {
    setMessages((prev) => prev.filter((m) => m.id !== messageId));
  };

  const uploadAttachment = async (file: File, text?: string) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('conversationId', currentConversation);
    formData.append('sender', user?.name ?? 'StatChat User');
    if (text) {
      formData.append('text', text);
    }

    try {
      const createdMessage = await uploadChatAttachment(formData);
      setMessages((prev) => {
        if (prev.some((message) => message.id === createdMessage.id)) {
          return prev;
        }
        return [...prev, createdMessage];
      });
      setDraft('');
      setError(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      console.error('Failed to upload attachment', error);
      setError(`Unable to upload attachment: ${message}`);
    }
  };

  const activeButtonStyle = (isActive: boolean): React.CSSProperties => ({
    ...(isActive ? { background: '#ffffff', color: '#165c92', borderColor: '#ffffff' } : {}),
  });

  return (
    <div className={styles.appShell} style={{ color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
      {error && (
        <div style={{ position: 'sticky', top: 0, zIndex: 20, background: '#fef2f2', color: '#991b1b', padding: '12px 16px', borderBottom: '1px solid #fecaca' }}>
          {error}
        </div>
      )}

      {showAppLauncher && (
        <div className={styles.launcherBackdrop} onClick={() => setShowAppLauncher(false)}>
          <div className={styles.launcherPanel} onClick={(event) => event.stopPropagation()}>
            <div className={styles.launcherHeader}>
              <div>
                <div className={styles.launcherLabel}>Application Launcher</div>
                <div className={styles.launcherSubtitle}>Jump between StatGate modules and services.</div>
              </div>
              <button
                type="button"
                className={styles.launcherCloseButton}
                onClick={() => setShowAppLauncher(false)}
                aria-label="Close launcher"
              >
                ✕
              </button>
            </div>
            <div className={styles.launcherGrid}>
              {[
                { name: 'Dashboard', url: 'http://localhost:5000', icon: '🏠' },
                { name: 'Dataset Catalog', url: 'http://localhost:5000/datasets', icon: '📊' },
                { name: 'Notebook', url: 'http://localhost:5000/notebook', icon: '📓' },
                { name: 'Semantic Registry', url: 'http://localhost:5000/semantic', icon: '🧠' },
                { name: 'ABAC Security', url: 'http://localhost:5000/abac', icon: '🔐' },
                { name: 'Executive Centre', url: 'http://localhost:5000/executive', icon: '🏛️' },
                { name: 'System Launcher', url: 'http://localhost:3002', icon: 'bi bi-grid-3x3-gap-fill' },
                { name: 'Register Portal', url: 'http://localhost:3000', icon: '📝' },
              ].map((app) => (
                <a
                  key={app.name}
                  href={app.url}
                  className={styles.launcherItem}
                >
                  <div className={styles.launcherIcon}>
                    {app.icon.includes(' ') ? <i className={app.icon} /> : app.icon}
                  </div>
                  <div className={styles.launcherName}>{app.name}</div>
                </a>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ── Header (StatGate brand style) ── */}
      <header className={`${styles.header} ${theme === 'dark' ? styles.headerDark : ''}`}>
        <div className={styles.headerBrand}>
          <span style={{ fontSize: 20 }}>🚀</span>
          <div className={styles.headerBrandText}>
            <span className={styles.headerBrandTitle}>StatGate</span>
            <span className={styles.headerBrandSub}>StatChat</span>
          </div>
        </div>

<div className={styles.headerSearch}>
          <span style={{ marginRight: 10, opacity: 0.65 }}>🔍</span>
          <input
            type="text"
            placeholder="Search StatChat..."
            className={styles.headerSearchInput}
            value={globalSearch}
            onChange={(event) => setGlobalSearch(event.target.value)}
          />
        </div>

        <div className={styles.headerActions}>
          <button type="button" className={styles.headerActionButton} title="App launcher" onClick={() => setShowAppLauncher(true)}>
            ◧
          </button>
          <div ref={notificationsRef} style={{ position: 'relative' }}>
            <button
              type="button"
              className={styles.headerActionButton}
              title="Notifications"
              onClick={handleNotificationsClick}
            >
              🔔
              {unreadCount > 0 && (
                <span style={{
                  position: 'absolute',
                  top: -4,
                  right: -4,
                  background: '#ef4444',
                  color: '#fff',
                  borderRadius: 999,
                  fontSize: 10,
                  fontWeight: 700,
                  minWidth: 16,
                  height: 16,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  padding: '0 4px',
                }}>
                  {unreadCount}
                </span>
              )}
            </button>
            {showNotifications && (
              <div style={{
                position: 'absolute',
                top: 40,
                right: 0,
                width: 320,
                maxHeight: 400,
                overflowY: 'auto',
                background: theme === 'dark' ? '#0a2b45' : '#ffffff',
                border: `1px solid ${theme === 'dark' ? '#6b7280' : '#e5e7eb'}`,
                borderRadius: 16,
                boxShadow: '0 12px 30px rgba(22, 92, 146, 0.15)',
                zIndex: 50,
                padding: 12,
              }}>
                <strong style={{ display: 'block', marginBottom: 10, color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
                  Notifications
                </strong>
                {notifications.length === 0 ? (
                  <div style={{ padding: 20, textAlign: 'center', opacity: 0.6, color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
                    No notifications yet
                  </div>
                ) : (
                  notifications.map((notification) => (
                    <div
                      key={notification.id}
                      style={{
                        padding: '10px 12px',
                        borderRadius: 12,
                        marginBottom: 6,
                        background: notification.read
                          ? (theme === 'dark' ? '#0f3f5f' : '#f9fafb')
                          : (theme === 'dark' ? '#165c92' : '#e8f0fe'),
                        color: theme === 'dark' ? '#e8eef4' : '#1a1a1a',
                      }}
                    >
                      <div style={{ fontWeight: 600, fontSize: 13 }}>{notification.title}</div>
                      <div style={{ fontSize: 12, opacity: 0.8, marginTop: 2 }}>{notification.body}</div>
                      <div style={{ fontSize: 11, opacity: 0.6, marginTop: 4 }}>
                        {new Date(notification.createdAt).toLocaleString()}
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
          <button
            type="button"
            className={`${styles.headerActionButton} ${activeSidebarView === 'projects' ? styles.headerActionButtonActive : ''}`}
            title="Projects"
            onClick={() => setActiveSidebarView('projects')}
          >
            📂
          </button>
          <button
            type="button"
            className={`${styles.headerActionButton} ${activeSidebarView === 'research' ? styles.headerActionButtonActive : ''}`}
            title="Research"
            onClick={() => setActiveSidebarView('research')}
          >
            🔬
          </button>
          <button
            type="button"
            className={`${styles.headerActionButton} ${activeSidebarView === 'tasks' ? styles.headerActionButtonActive : ''}`}
            title="Tasks"
            onClick={() => setActiveSidebarView('tasks')}
          >
            ✅
          </button>
          <button
            type="button"
            className={`${styles.headerActionButton} ${activeSidebarView === 'contacts' ? styles.headerActionButtonActive : ''}`}
            title="Contacts"
            onClick={() => setActiveSidebarView('contacts')}
          >
            🧑
          </button>
          <button
            type="button"
            className={`${styles.headerActionButton} ${activeSidebarView === 'communities' ? styles.headerActionButtonActive : ''}`}
            title="Communities"
            onClick={() => setActiveSidebarView('communities')}
          >
            🏛️
          </button>
          <button
            type="button"
            className={`${styles.headerActionButton} ${isSettingsView ? styles.headerActionButtonActive : ''}`}
            title="Settings"
            onClick={() => setActiveSidebarView('settings')}
          >
            ⚙️
          </button>
        </div>
      </header>

      <div className={styles.sidebarShell}>
        <SidebarRail activeView={activeSidebarView} onSelect={setActiveSidebarView} theme={theme} />
      </div>

      <main className={styles.innerShell}>
        {isCollaborationView ? (
          <div className={styles.subheader}>
            <div className={styles.subheaderTabs}>
              {subHeaderTabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  className={`${styles.subheaderTab} ${activeSubView === tab.id ? styles.subheaderTabActive : ''}`}
                  onClick={() => setActiveSubView(tab.id)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        {isCalendarView ? (
          <div className={styles.subheader}>
            <div className={styles.subheaderTabs}>
              {calendarTabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  className={`${styles.subheaderTab} ${activeCalendarView === tab.id ? styles.subheaderTabActive : ''}`}
                  onClick={() => setActiveCalendarView(tab.id)}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <div className={styles.workspaceContent}>
          {isChatWorkspace ? (
            isMobile ? (
              mobileView === 'list' ? (
                <ChatSidebarList
                  conversations={conversations}
                  activeConversationId={currentConversation}
                  onSelectConversation={handleSelectConversation}
                  theme={theme}
                  messages={messages}
                  isMobile={isMobile}
                  currentUserId={user?.id}
                  onConversationCreated={handleConversationCreated}
                  viewType={chatViewType}
                  externalSearch={globalSearch}
                />
              ) : (
<ChatWorkspace
                  conversation={activeConversation}
                  messages={messages}
                  draft={draft}
                  theme={theme}
                  isMobile={isMobile}
                  currentUserName={user?.name}
                  currentUserId={user?.id}
                  onDraftChange={setDraft}
                  onSend={sendMessage}
                  onUploadAttachment={uploadAttachment}
                  onCloseMobile={closeMobileChat}
                  onMessageUpdate={handleMessageUpdate}
                  onMessageDelete={handleMessageDelete}
                />
              )
            ) : (
              <div className={styles.chatGrid}>
<ChatSidebarList
                  conversations={conversations}
                  activeConversationId={currentConversation}
                  onSelectConversation={handleSelectConversation}
                  theme={theme}
                  messages={messages}
                  isMobile={isMobile}
                  currentUserId={user?.id}
                  onConversationCreated={handleConversationCreated}
                  viewType={chatViewType}
                  externalSearch={globalSearch}
                />
<ChatWorkspace
                  conversation={activeConversation}
                  messages={messages}
                  draft={draft}
                  theme={theme}
                  isMobile={isMobile}
                  currentUserName={user?.name}
                  currentUserId={user?.id}
                  onDraftChange={setDraft}
                  onSend={sendMessage}
                  onUploadAttachment={uploadAttachment}
                  onMessageUpdate={handleMessageUpdate}
                  onMessageDelete={handleMessageDelete}
                />
              </div>
            )
          ) : isSettingsView ? (
            <SettingsPanel
              user={user}
              theme={theme}
              isMobile={isMobile}
              onThemeChange={setTheme}
              onUserUpdate={setUser}
            />
          ) : isCollaborationView ? (
            <CollaborationPanel
              user={user}
              theme={theme}
              isMobile={isMobile}
              activeSubView={activeSubView}
            />
          ) : isCalendarView ? (
            <CalendarPanel
              user={user}
              theme={theme}
              isMobile={isMobile}
              activeCalendarView={activeCalendarView}
              nativeMode={isNativeMeetingView}
              onJoinMeeting={handleJoinMeeting}
            />
          ) : isWellnessView ? (
            <WellnessPanel
              user={user}
              theme={theme}
              isMobile={isMobile}
            />
          ) : isKnowledgeView ? (
            <KnowledgePanel
              user={user}
              theme={theme}
              isMobile={isMobile}
            />
          ) : (
            <IntegrationPanel
              activeView={activeSidebarView}
              conversation={activeConversation}
              conversations={conversations}
              theme={theme}
              isMobile={isMobile}
              onSelectConversation={setCurrentConversation}
            />
          )}
        </div>
      </main>

      {meetingSession && (
        <CallOverlay
          session={meetingSession}
          participants={meetingParticipants}
          localStream={meetingLocalStream}
          remoteStreams={meetingCallRemoteStreams}
          micMuted={meetingMicMuted}
          cameraOff={meetingCameraOff}
          connecting={meetingConnecting}
          currentUserName={user?.name}
          onToggleMute={toggleMeetingMute}
          onToggleCamera={toggleMeetingCamera}
          onHangUp={hangUpMeeting}
          onEndCall={endMeeting}
        />
      )}

      {/* ── Footer (StatGate brand style) ── */}
      <footer className={styles.footer}>
        <div className={styles.footerBrand}>
          <span style={{ fontSize: 16 }}>🚀</span>
          <span className={styles.footerBrandName}>StatGate</span>
        </div>
        <div className={styles.footerText}>
          &copy; {new Date().getFullYear()} StatGate. Enterprise Communication & Collaboration.
        </div>
        <div className={styles.footerLinks}>
          <button
            type="button"
            className={styles.footerLink}
            onClick={() => { setLegalPanelTab('privacy'); setLegalPanelOpen(true); }}
          >
            Privacy
          </button>
          <button
            type="button"
            className={styles.footerLink}
            onClick={() => { setLegalPanelTab('terms'); setLegalPanelOpen(true); }}
          >
            Terms
          </button>
          <button
            type="button"
            className={styles.footerLink}
            onClick={() => { setLegalPanelTab('status'); setLegalPanelOpen(true); }}
          >
            Status
          </button>
        </div>
      </footer>

      <LegalPanel
        open={legalPanelOpen}
        initialTab={legalPanelTab}
        onClose={() => setLegalPanelOpen(false)}
      />
    </div>
  );
}