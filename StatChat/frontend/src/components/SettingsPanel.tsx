import { useEffect, useRef, useState, type CSSProperties, type ChangeEvent } from 'react';
import type { User, UserSettings } from '../types';
import { fetchUserSettings, updateUserSettings, updateProfile } from '../api/client';
import styles from './SettingsPanel.module.css';

export type SettingsSection =
  | 'profile'
  | 'notifications'
  | 'appearance'
  | 'privacy'
  | 'chats'
  | 'storage'
  | 'language'
  | 'help'
  | 'about';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
  onThemeChange: (theme: 'light' | 'dark') => void;
  onUserUpdate: (user: User) => void;
}

interface NavMeta {
  label: string;
  subtitle: string;
  icon: string;
}

const navMeta: Record<SettingsSection, NavMeta> = {
  profile: { label: 'Profile', subtitle: 'Name, avatar, about', icon: '👤' },
  notifications: { label: 'Notifications', subtitle: 'Messages, mentions, sounds', icon: '🔔' },
  appearance: { label: 'Appearance', subtitle: 'Theme and accent colour', icon: '🎨' },
  privacy: { label: 'Privacy', subtitle: 'Last seen, receipts, blocks', icon: '🔒' },
  chats: { label: 'Chats & Calls', subtitle: 'Wallpaper, font size, calls', icon: '💬' },
  storage: { label: 'Storage & Data', subtitle: 'Usage, downloads, clear data', icon: '🗄️' },
  language: { label: 'Language', subtitle: 'Interface language', icon: '🌐' },
  help: { label: 'Help', subtitle: 'FAQ, contact, community', icon: '❓' },
  about: { label: 'About', subtitle: 'Version and licenses', icon: 'ℹ️' },
};

const accentColors = [
  { name: 'Blue', value: '#165c92' },
  { name: 'Teal', value: '#0f766e' },
  { name: 'Green', value: '#16a34a' },
  { name: 'Violet', value: '#7c3aed' },
  { name: 'Rose', value: '#e11d48' },
  { name: 'Amber', value: '#d97706' },
];

const languages = [
  'English',
  'Kiswahili',
  'French',
  'Spanish',
  'Arabic',
  'Portuguese',
  'Dutch',
  'German',
];

const defaultSettings: UserSettings = {
  userId: 'user-001',
  theme: 'light',
  accentColor: '#165c92',
  fontSize: 'medium',
  enterToSend: true,
  language: 'English',
  lastSeen: 'everyone',
  profilePhoto: 'contacts',
  readReceipts: true,
  typingIndicator: true,
  voiceNotes: true,
  readByDefault: false,
  autoDownload: 'never',
  notifMessages: true,
  notifGroups: true,
  notifMentions: true,
  notifMeetings: true,
  notifSound: true,
  notifPreview: false,
  downloadImages: 'wifi',
  downloadVideos: 'wifi',
  downloadDocuments: 'wifi',
};

export default function SettingsPanel({ user, theme, isMobile, onThemeChange, onUserUpdate }: Props) {
  const [activeSection, setActiveSection] = useState<SettingsSection>('profile');
  const [mobileDetail, setMobileDetail] = useState(false);
  const [editingProfile, setEditingProfile] = useState(false);
  const [saveNote, setSaveNote] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const saveNoteTimer = useRef<number | null>(null);
  const avatarInputRef = useRef<HTMLInputElement | null>(null);

  const [draftName, setDraftName] = useState(user?.name ?? '');
  const [draftAbout, setDraftAbout] = useState(user?.about ?? 'Enterprise researcher at StatGate');
  const [draftAvatar, setDraftAvatar] = useState(user?.avatarUrl ?? '');

  const [settings, setSettings] = useState<UserSettings>(defaultSettings);

  const background = theme === 'dark' ? '#0a2b45' : '#ffffff';
  const textColor = theme === 'dark' ? '#e8eef4' : '#1a1a1a';
  const borderColor = theme === 'dark' ? '#6b7280' : '#e5e7eb';

  useEffect(() => {
    let cancelled = false;
    async function loadSettings() {
      try {
        const s = await fetchUserSettings();
        if (!cancelled && s) {
          setSettings({ ...defaultSettings, ...s });
        }
      } catch {
        // use defaults if settings can't be loaded
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    loadSettings();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (user) {
      setDraftName(user.name);
      setDraftAbout(user.about ?? 'Enterprise researcher at StatGate');
      setDraftAvatar(user.avatarUrl ?? '');
    }
  }, [user]);

  useEffect(() => {
    return () => {
      if (saveNoteTimer.current) {
        window.clearTimeout(saveNoteTimer.current);
      }
    };
  }, []);

  const selectSection = (section: SettingsSection) => {
    setActiveSection(section);
    if (isMobile) {
      setMobileDetail(true);
    }
  };

  const backToList = () => {
    setMobileDetail(false);
  };

  const showSaveNote = (message: string) => {
    setSaveNote(message);
    if (saveNoteTimer.current) {
      window.clearTimeout(saveNoteTimer.current);
    }
    saveNoteTimer.current = window.setTimeout(() => setSaveNote(null), 1600);
  };

  const saveSettings = (updated: Partial<UserSettings>) => {
    const next = { ...settings, ...updated };
    setSettings(next);
    setSaving(true);
    updateUserSettings(next)
      .then(() => showSaveNote('Settings saved'))
      .catch(() => showSaveNote('Failed to save — using local state'))
      .finally(() => setSaving(false));
  };

  const handleAvatarChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      setDraftAvatar(dataUrl);
    };
    reader.readAsDataURL(file);
    event.target.value = '';
  };

  const saveProfile = () => {
    const nextName = draftName.trim() || user?.name || 'StatChat User';
    const nextAbout = draftAbout.trim() || 'Enterprise researcher at StatGate';
    setSaving(true);
    updateProfile(nextName, nextAbout, draftAvatar)
      .then((updatedUser) => {
        onUserUpdate(updatedUser);
        setEditingProfile(false);
        showSaveNote('Profile updated');
      })
      .catch(() => showSaveNote('Failed to update profile'))
      .finally(() => setSaving(false));
  };

  const accentStyle = { '--accent': settings.accentColor } as CSSProperties;

  const renderToggle = (on: boolean, onClick: () => void, label: string) => (
    <button
      type="button"
      className={`${styles.toggle} ${on ? styles.toggleOn : ''}`}
      onClick={onClick}
      title={label}
      aria-label={label}
    >
      <span className={styles.toggleKnob} />
    </button>
  );

  const avatarSrc = editingProfile
    ? draftAvatar || `https://api.dicebear.com/6.x/initials/svg?seed=${user?.id ?? 'user-1'}`
    : user?.avatarUrl || `https://api.dicebear.com/6.x/initials/svg?seed=${user?.id ?? 'user-1'}`;

  const renderProfileSection = () => (
    <div style={accentStyle}>
      <input ref={avatarInputRef} type="file" accept="image/*" className={styles.hiddenInput} onChange={handleAvatarChange} />
      <div className={styles.profileHero} style={{ borderColor }}>
        <img className={styles.heroAvatar} src={avatarSrc} alt={user?.name ?? 'Profile'} />
        <div className={styles.profileHeroInfo}>
          <strong>{user?.name ?? 'StatChat User'}</strong>
          <span>{user?.email ?? 'user@statchat.local'}</span>
          <span>{user?.about ?? user?.organizationId ?? 'StatGate'}</span>
        </div>
        <button type="button" className={styles.heroAction} onClick={() => setEditingProfile((prev) => !prev)}>
          {editingProfile ? 'Cancel' : 'Edit'}
        </button>
      </div>

      {editingProfile && (
        <div className={styles.editCard} style={{ borderColor }}>
          <label className={styles.fieldLabel} htmlFor="profile-avatar">Profile photo</label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <img
              src={draftAvatar || `https://api.dicebear.com/6.x/initials/svg?seed=${user?.id ?? 'user-1'}`}
              alt="Preview"
              style={{ width: 56, height: 56, borderRadius: '50%', objectFit: 'cover', border: `2px solid ${borderColor}` }}
            />
            <button type="button" className={styles.cancelButton} onClick={() => avatarInputRef.current?.click()}>
              Choose photo
            </button>
            {draftAvatar && (
              <button type="button" className={styles.cancelButton} onClick={() => setDraftAvatar('')}>
                Remove
              </button>
            )}
          </div>
          <label className={styles.fieldLabel} htmlFor="profile-name">Name</label>
          <input
            id="profile-name"
            className={styles.textInput}
            value={draftName}
            onChange={(event) => setDraftName(event.target.value)}
            placeholder="Your display name"
          />
          <label className={styles.fieldLabel} htmlFor="profile-about">About</label>
          <textarea
            id="profile-about"
            className={styles.textInput}
            rows={3}
            value={draftAbout}
            onChange={(event) => setDraftAbout(event.target.value)}
            placeholder="A short description about you"
          />
          <div className={styles.editActions}>
            <button type="button" className={styles.cancelButton} onClick={() => setEditingProfile(false)}>
              Cancel
            </button>
            <button type="button" className={styles.saveButton} onClick={saveProfile} disabled={saving}>
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      )}

      <p className={styles.groupHeader}>Account</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>📧</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Email</span>
            <span className={styles.rowDescription}>{user?.email ?? 'user@statchat.local'}</span>
          </div>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🏢</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Organization</span>
            <span className={styles.rowDescription}>{user?.organizationId ?? 'StatGate'}</span>
          </div>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🛡️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Roles</span>
            <span className={styles.rowDescription}>{user?.roles?.join(', ') ?? 'member'}</span>
          </div>
        </div>
      </div>

      <p className={styles.groupHeader}>Security</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Two-step verification is managed by the StatGate launcher')}>
          <span className={styles.rowIcon}>🔐</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Two-step verification</span>
            <span className={styles.rowDescription}>Managed by StatGate Identity</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
        <div className={styles.row} onClick={() => showSaveNote('Active sessions are managed by the StatGate launcher')}>
          <span className={styles.rowIcon}>🔑</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Active sessions</span>
            <span className={styles.rowDescription}>1 desktop session</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderNotificationsSection = () => (
    <div>
      <p className={styles.groupHeader}>Chat notifications</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>💬</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Messages</span>
            <span className={styles.rowDescription}>New direct and channel messages</span>
          </div>
          {renderToggle(settings.notifMessages, () => saveSettings({ notifMessages: !settings.notifMessages }), 'Toggle messages')}
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>👥</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Groups</span>
            <span className={styles.rowDescription}>Activity in team spaces and groups</span>
          </div>
          {renderToggle(settings.notifGroups, () => saveSettings({ notifGroups: !settings.notifGroups }), 'Toggle groups')}
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>@</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Mentions</span>
            <span className={styles.rowDescription}>When you are mentioned in a chat</span>
          </div>
          {renderToggle(settings.notifMentions, () => saveSettings({ notifMentions: !settings.notifMentions }), 'Toggle mentions')}
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>📅</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Meetings</span>
            <span className={styles.rowDescription}>Invitations and meeting reminders</span>
          </div>
          {renderToggle(settings.notifMeetings, () => saveSettings({ notifMeetings: !settings.notifMeetings }), 'Toggle meetings')}
        </div>
      </div>

      <p className={styles.groupHeader}>Delivery</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🔊</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Notification sounds</span>
            <span className={styles.rowDescription}>Play a sound for incoming alerts</span>
          </div>
          {renderToggle(settings.notifSound, () => saveSettings({ notifSound: !settings.notifSound }), 'Toggle sounds')}
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🪟</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Message previews</span>
            <span className={styles.rowDescription}>Show message text in notifications</span>
          </div>
          {renderToggle(settings.notifPreview, () => saveSettings({ notifPreview: !settings.notifPreview }), 'Toggle previews')}
        </div>
      </div>

      <p className={styles.groupHeader}>Email</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Email notification hooks are configured in StatGate Notifications')}>
          <span className={styles.rowIcon}>📧</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Email digest</span>
            <span className={styles.rowDescription}>Daily summary of activity</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderAppearanceSection = () => (
    <div style={accentStyle}>
      <p className={styles.groupHeader}>Theme</p>
      <div className={styles.groupCard} style={{ borderColor, padding: 16 }}>
        <div className={styles.themeOptions}>
          <button
            type="button"
            className={`${styles.themeCard} ${theme === 'light' ? styles.themeCardActive : ''}`}
            onClick={() => {
              onThemeChange('light');
              saveSettings({ theme: 'light' });
            }}
          >
            <span className={styles.themePreviewLight} />
            Light
            {theme === 'light' && <span className={styles.themeCheck}>✓</span>}
          </button>
          <button
            type="button"
            className={`${styles.themeCard} ${theme === 'dark' ? styles.themeCardActive : ''}`}
            onClick={() => {
              onThemeChange('dark');
              saveSettings({ theme: 'dark' });
            }}
          >
            <span className={styles.themePreviewDark} />
            Dark
            {theme === 'dark' && <span className={styles.themeCheck}>✓</span>}
          </button>
        </div>
      </div>

      <p className={styles.groupHeader}>Accent colour</p>
      <div className={styles.groupCard} style={{ borderColor, padding: 16 }}>
        <div className={styles.accentOptions}>
          {accentColors.map((color) => (
            <button
              key={color.value}
              type="button"
              title={color.name}
              className={`${styles.accentSwatch} ${settings.accentColor === color.value ? styles.accentSwatchActive : ''}`}
              style={{ background: color.value }}
              onClick={() => {
                saveSettings({ accentColor: color.value });
                showSaveNote(`Accent set to ${color.name}`);
              }}
            />
          ))}
        </div>
      </div>

      <p className={styles.groupHeader}>Chat display</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🔤</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Message font size</span>
            <span className={styles.rowDescription}>Small, medium, or large</span>
          </div>
          <select
            className={styles.select}
            value={settings.fontSize}
            onChange={(event) => saveSettings({ fontSize: event.target.value })}
          >
            <option value="small">Small</option>
            <option value="medium">Medium</option>
            <option value="large">Large</option>
          </select>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>↩️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Enter to send</span>
            <span className={styles.rowDescription}>Use Enter to send the message</span>
          </div>
          {renderToggle(settings.enterToSend, () => saveSettings({ enterToSend: !settings.enterToSend }), 'Toggle enter to send')}
        </div>
      </div>
    </div>
  );

  const renderPrivacySection = () => (
    <div>
      <p className={styles.groupHeader}>Presence</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>👁️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Last seen</span>
            <span className={styles.rowDescription}>Who can see your last seen time</span>
          </div>
          <select
            className={styles.select}
            value={settings.lastSeen}
            onChange={(event) => saveSettings({ lastSeen: event.target.value })}
          >
            <option value="everyone">Everyone</option>
            <option value="contacts">My contacts</option>
            <option value="nobody">Nobody</option>
          </select>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🖼️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Profile photo</span>
            <span className={styles.rowDescription}>Who can see your profile photo</span>
          </div>
          <select
            className={styles.select}
            value={settings.profilePhoto}
            onChange={(event) => saveSettings({ profilePhoto: event.target.value })}
          >
            <option value="everyone">Everyone</option>
            <option value="contacts">My contacts</option>
            <option value="nobody">Nobody</option>
          </select>
        </div>
      </div>

      <p className={styles.groupHeader}>Messaging</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>✓✓</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Read receipts</span>
            <span className={styles.rowDescription}>Let others know when you read their messages</span>
          </div>
          {renderToggle(settings.readReceipts, () => saveSettings({ readReceipts: !settings.readReceipts }), 'Toggle read receipts')}
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>⌨️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Typing indicator</span>
            <span className={styles.rowDescription}>Show when you are typing a reply</span>
          </div>
          {renderToggle(settings.typingIndicator, () => saveSettings({ typingIndicator: !settings.typingIndicator }), 'Toggle typing indicator')}
        </div>
      </div>

      <p className={styles.groupHeader}>Block list</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Blocking will be available with Contacts')}>
          <span className={styles.rowIcon}>🚫</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Blocked contacts</span>
            <span className={styles.rowDescription}>0 blocked contacts</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderChatsSection = () => (
    <div>
      <p className={styles.groupHeader}>Chats</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Wallpaper picker coming with Chats')}>
          <span className={styles.rowIcon}>🖼️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Wallpaper</span>
            <span className={styles.rowDescription}>Customize your chat background</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🗑️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Media auto-download</span>
            <span className={styles.rowDescription}>Downloads depend on your connection</span>
          </div>
          <select
            className={styles.select}
            value={settings.autoDownload}
            onChange={(event) => saveSettings({ autoDownload: event.target.value })}
          >
            <option value="never">Never</option>
            <option value="wifi">Wi-Fi only</option>
            <option value="always">Always</option>
          </select>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🗂️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Read messages by default</span>
            <span className={styles.rowDescription}>Mark new messages as read automatically</span>
          </div>
          {renderToggle(settings.readByDefault, () => saveSettings({ readByDefault: !settings.readByDefault }), 'Toggle read by default')}
        </div>
      </div>

      <p className={styles.groupHeader}>Calls</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🎙️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Voice notes</span>
            <span className={styles.rowDescription}>Allow recording and sending voice notes</span>
          </div>
          {renderToggle(settings.voiceNotes, () => saveSettings({ voiceNotes: !settings.voiceNotes }), 'Toggle voice notes')}
        </div>
        <div className={styles.row} onClick={() => showSaveNote('Call routing is managed by StatGate Calls')}>
          <span className={styles.rowIcon}>📞</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Call settings</span>
            <span className={styles.rowDescription}>Ringtone, data saver, and routing</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderStorageSection = () => (
    <div>
      <div className={styles.storageCard} style={{ borderColor }}>
        <div className={styles.storageHeader}>
          <strong>Storage used</strong>
          <span>2.7 GB of 5 GB</span>
        </div>
        <div className={styles.storageBar}>
          <span className={styles.storageFill} style={{ width: '54%' }} />
        </div>
      </div>

      <p className={styles.groupHeader}>Media downloads</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>📷</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Images</span>
            <span className={styles.rowDescription}>Auto-download recent images</span>
          </div>
          <select
            className={styles.select}
            value={settings.downloadImages}
            onChange={(event) => saveSettings({ downloadImages: event.target.value })}
          >
            <option value="wifi">Wi-Fi only</option>
            <option value="always">Always</option>
            <option value="never">Never</option>
          </select>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🎥</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Videos</span>
            <span className={styles.rowDescription}>Auto-download recent videos</span>
          </div>
          <select
            className={styles.select}
            value={settings.downloadVideos}
            onChange={(event) => saveSettings({ downloadVideos: event.target.value })}
          >
            <option value="wifi">Wi-Fi only</option>
            <option value="always">Always</option>
            <option value="never">Never</option>
          </select>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>📄</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Documents</span>
            <span className={styles.rowDescription}>Auto-download recent documents</span>
          </div>
          <select
            className={styles.select}
            value={settings.downloadDocuments}
            onChange={(event) => saveSettings({ downloadDocuments: event.target.value })}
          >
            <option value="wifi">Wi-Fi only</option>
            <option value="always">Always</option>
            <option value="never">Never</option>
          </select>
        </div>
      </div>

      <p className={styles.groupHeader}>Manage</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Storage breakdown available with Files')}>
          <span className={styles.rowIcon}>📊</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Storage usage by chat</span>
            <span className={styles.rowDescription}>See which conversations use the most space</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
        <div className={`${styles.row} ${styles.dangerRow}`} onClick={() => showSaveNote('Clearing history is disabled in this workspace')}>
          <span className={styles.rowIcon}>🗑️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Clear chat history</span>
            <span className={styles.rowDescription}>Delete local message history</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderLanguageSection = () => (
    <div>
      <p className={styles.groupHeader}>Interface</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🌐</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Display language</span>
            <span className={styles.rowDescription}>StatChat language preference</span>
          </div>
          <select
            className={styles.select}
            value={settings.language}
            onChange={(event) => saveSettings({ language: event.target.value })}
          >
            {languages.map((lang) => (
              <option key={lang} value={lang}>
                {lang}
              </option>
            ))}
          </select>
        </div>
      </div>

      <p className={styles.groupHeader}>Region</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🏳️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Time zone</span>
            <span className={styles.rowDescription}>Automatically detected from your device</span>
          </div>
          <span className={styles.rowValue}>{Intl.DateTimeFormat().resolvedOptions().timeZone ?? 'UTC'}</span>
        </div>
      </div>
    </div>
  );

  const renderHelpSection = () => (
    <div>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Help centre will open in a new window')}>
          <span className={styles.rowIcon}>📖</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Help centre</span>
            <span className={styles.rowDescription}>Guides and answers for StatChat</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
        <div className={styles.row} onClick={() => showSaveNote('Opening a support request with the StatGate team')}>
          <span className={styles.rowIcon}>💬</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Contact support</span>
            <span className={styles.rowDescription}>Open a request with the StatGate team</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
        <div className={styles.row} onClick={() => showSaveNote('Community forums will open in a new window')}>
          <span className={styles.rowIcon}>👥</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Community</span>
            <span className={styles.rowDescription}>Connect with other StatGate users</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderAboutSection = () => (
    <div>
      <div className={styles.profileHero} style={{ borderColor }}>
        <span style={{ fontSize: 40 }}>🚀</span>
        <div className={styles.profileHeroInfo}>
          <strong>StatChat</strong>
          <span>Enterprise Collaboration & Communication</span>
          <span>Version 0.1.0 (Milestone 1)</span>
        </div>
      </div>

      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>⚙️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Built on</span>
            <span className={styles.rowDescription}>Go, React, TypeScript, Vite</span>
          </div>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🔌</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Real-time layer</span>
            <span className={styles.rowDescription}>WebSocket /ws/chat</span>
          </div>
        </div>
        <div className={styles.row} style={{ cursor: 'default' }}>
          <span className={styles.rowIcon}>🗄️</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Data</span>
            <span className={styles.rowDescription}>PostgreSQL persisted store</span>
          </div>
        </div>
      </div>

      <p className={styles.groupHeader}>Licenses</p>
      <div className={styles.groupCard} style={{ borderColor }}>
        <div className={styles.row} onClick={() => showSaveNote('Open-source licences will open in a new window')}>
          <span className={styles.rowIcon}>📄</span>
          <div className={styles.rowText}>
            <span className={styles.rowLabel}>Open-source licences</span>
            <span className={styles.rowDescription}>View third-party software notices</span>
          </div>
          <span className={styles.rowValue}>›</span>
        </div>
      </div>
    </div>
  );

  const renderSection = () => {
    switch (activeSection) {
      case 'profile':
        return renderProfileSection();
      case 'notifications':
        return renderNotificationsSection();
      case 'appearance':
        return renderAppearanceSection();
      case 'privacy':
        return renderPrivacySection();
      case 'chats':
        return renderChatsSection();
      case 'storage':
        return renderStorageSection();
      case 'language':
        return renderLanguageSection();
      case 'help':
        return renderHelpSection();
      case 'about':
        return renderAboutSection();
    }
  };

  const activeMeta = navMeta[activeSection];
  const showList = !isMobile || !mobileDetail;
  const showDetail = !isMobile || mobileDetail;

  if (loading) {
    return (
      <div className={styles.settingsShell} style={{ background, color: textColor }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', opacity: 0.6 }}>
          Loading settings...
        </div>
      </div>
    );
  }

  return (
    <div className={styles.settingsShell} style={{ background, color: textColor }}>
      {showList && (
        <aside className={styles.settingsListPane} style={{ background, borderColor }}>
          <div className={styles.profileCard}>
            <img
              className={styles.profileAvatar}
              src={user?.avatarUrl || `https://api.dicebear.com/6.x/initials/svg?seed=${user?.id ?? 'user-1'}`}
              alt={user?.name ?? 'Profile'}
            />
            <div className={styles.profileInfo}>
              <span className={styles.profileName}>{user?.name ?? 'StatChat User'}</span>
              <span className={styles.profileMeta}>{user?.email ?? 'user@statchat.local'}</span>
            </div>
            <button
              type="button"
              className={styles.deviceBadge}
              title="Devices"
              onClick={() => showSaveNote('Linked devices are managed by the StatGate launcher')}
            >
              📱
            </button>
          </div>

          <nav className={styles.settingsNav}>
            {(Object.keys(navMeta) as SettingsSection[]).map((section) => {
              const meta = navMeta[section];
              const active = section === activeSection;
              return (
                <button
                  key={section}
                  type="button"
                  className={`${styles.navItem} ${active ? styles.navItemActive : ''}`}
                  onClick={() => selectSection(section)}
                >
                  <span className={styles.navIcon}>{meta.icon}</span>
                  <span className={styles.navText}>
                    <span className={styles.navLabel}>{meta.label}</span>
                    <span className={styles.navSubtitle}>{meta.subtitle}</span>
                  </span>
                  <span className={styles.chevron}>›</span>
                </button>
              );
            })}
          </nav>
        </aside>
      )}

      {showDetail && (
        <section className={styles.detailPane}>
          <div className={styles.detailTopbar}>
            {isMobile && (
              <button type="button" className={styles.backButton} onClick={backToList} title="Back to settings">
                ←
              </button>
            )}
            <span className={styles.backLabel}>{activeMeta.label}</span>
          </div>
          <div className={styles.detailScroll}>
            <header className={styles.detailHeader}>
              <h2 className={styles.detailTitle}>{activeMeta.label}</h2>
              <p className={styles.detailSubtitle}>{activeMeta.subtitle}</p>
            </header>
            {renderSection()}
          </div>
        </section>
      )}

      {saveNote && (
        <div
          style={{
            position: 'fixed',
            bottom: 24,
            left: '50%',
            transform: 'translateX(-50%)',
            padding: '12px 20px',
            borderRadius: 12,
            background: '#1a1a1a',
            color: '#ffffff',
            fontSize: 13,
            fontWeight: 600,
            boxShadow: '0 10px 24px rgba(22, 92, 146, 0.35)',
            zIndex: 100,
          }}
        >
          {saveNote}
        </div>
      )}
    </div>
  );
}