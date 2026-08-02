import { useEffect, useMemo, useRef, useState } from 'react';
import { fetchConversations, fetchCurrentUser, fetchMessages, WS_URL } from './api/client';
import { Conversation, Message, User } from './types';
import ChatSidebarList from './components/ChatSidebarList';
import ChatWorkspace from './components/ChatWorkspace';
import SidebarRail from './components/SidebarRail';
import IntegrationPanel from './components/IntegrationPanel';
import SettingsPanel from './components/SettingsPanel';
import CollaborationPanel from './components/CollaborationPanel';
import CalendarPanel from './components/CalendarPanel';
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
  const [isMobile, setIsMobile] = useState(false);
  const [mobileView, setMobileView] = useState<'list' | 'chat'>('list');
  const [activeSidebarView, setActiveSidebarView] = useState<SidebarView>('directMessages');
  const [activeSubView, setActiveSubView] = useState<SubHeaderView>('home');
  const [activeCalendarView, setActiveCalendarView] = useState<CalendarSubView>('overview');
  const [showAppLauncher, setShowAppLauncher] = useState(false);
  const [legalPanelOpen, setLegalPanelOpen] = useState(false);
  const [legalPanelTab, setLegalPanelTab] = useState<LegalTab>('privacy');
  const currentConversationRef = useRef(currentConversation);

  const activeConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === currentConversation),
    [conversations, currentConversation]
  );

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
        const [currentUser, conversationList, initialMessages] = await Promise.all([
          fetchCurrentUser(),
          fetchConversations(),
          fetchMessages(currentConversation),
        ]);
        setUser(currentUser);
        setConversations(conversationList);
        if (!conversationList.some((conversation: Conversation) => conversation.id === currentConversation)) {
          setCurrentConversation(conversationList[0]?.id ?? currentConversation);
        }
        setMessages(initialMessages);
      } catch (error) {
        console.error('Failed to load initial data', error);
      }
    }

    loadInitialData();

    const ws = new WebSocket(WS_URL);
    const joinConversation = () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ action: 'join-conversation', conversationId: currentConversationRef.current }));
      }
    };

    ws.addEventListener('open', () => {
      setConnected(true);
      joinConversation();
    });
    ws.addEventListener('close', () => setConnected(false));
    ws.addEventListener('message', (event) => {
      const message = JSON.parse(event.data) as Message;
      setMessages((prev) => {
        if (message.conversationId !== currentConversationRef.current) {
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

  useEffect(() => {
    const activeConversationId = currentConversation || 'general';
    currentConversationRef.current = activeConversationId;
    if (!socket) return;

    if (socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ action: 'join-conversation', conversationId: activeConversationId }));
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

  const sendMessage = (textOverride?: string) => {
    const messageText = (textOverride ?? draft).trim();
    if (!messageText || !socket || socket.readyState !== WebSocket.OPEN) return;

    const channelId = activeConversation?.type === 'channel' ? currentConversation : undefined;

    socket.send(
      JSON.stringify({
        action: 'send-message',
        conversationId: currentConversation,
        channelId,
        sender: user?.name ?? 'StatChat User',
        text: messageText,
      })
    );

    setDraft('');
  };

  const activeButtonStyle = (isActive: boolean): React.CSSProperties => ({
    ...(isActive ? { background: '#ffffff', color: '#165c92', borderColor: '#ffffff' } : {}),
  });

  return (
    <div className={styles.appShell} style={{ color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}>
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
          />
        </div>

        <div className={styles.headerActions}>
          <button type="button" className={styles.headerActionButton} title="App launcher" onClick={() => setShowAppLauncher(true)}>
            ◧
          </button>
          <button type="button" className={styles.headerActionButton} title="Notifications">
            🔔
          </button>
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
                />
              ) : (
                <ChatWorkspace
                  conversation={activeConversation}
                  messages={messages}
                  draft={draft}
                  theme={theme}
                  isMobile={isMobile}
                  currentUserName={user?.name}
                  onDraftChange={setDraft}
                  onSend={sendMessage}
                  onCloseMobile={closeMobileChat}
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
                />
                <ChatWorkspace
                  conversation={activeConversation}
                  messages={messages}
                  draft={draft}
                  theme={theme}
                  isMobile={isMobile}
                  currentUserName={user?.name}
                  onDraftChange={setDraft}
                  onSend={sendMessage}
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
              activeSubView={activeSubView}
              conversation={activeConversation}
              conversations={conversations}
              theme={theme}
              isMobile={isMobile}
              onSelectConversation={setCurrentConversation}
            />
          )}
        </div>
      </main>

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