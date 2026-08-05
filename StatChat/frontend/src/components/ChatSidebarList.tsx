import { useEffect, useMemo, useState } from 'react';
import type { Conversation, Message, User } from '../types';
import { fetchAllUsers, createDM, createGroup, fetchGroupTemplates, fetchPresence, fetchFavouriteIds, type GroupCategory } from '../api/client';
import styles from './ChatSidebarList.module.css';

interface Props {
  conversations: Conversation[];
  activeConversationId: string;
  onSelectConversation: (conversationId: string) => void;
  theme: 'light' | 'dark';
  messages: Message[];
  isMobile: boolean;
  currentUserId?: string;
onConversationCreated?: (conversation: Conversation) => void;
  viewType: 'direct' | 'group' | 'channel';
  externalSearch?: string;
}

const filters = ['All', 'Unread', 'Favourites', 'Users'] as const;
type Filter = typeof filters[number];

function formatTime(timestamp: string) {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays === 0) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) {
    return date.toLocaleDateString([], { weekday: 'short' });
  }
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
}

function getPreviewText(conversation: Conversation, activeMessage?: Message) {
  if (activeMessage) {
    return activeMessage.text.length > 45 ? `${activeMessage.text.slice(0, 45)}...` : activeMessage.text;
  }

  if (conversation.latestPreview) {
    return conversation.latestPreview.length > 45 ? `${conversation.latestPreview.slice(0, 45)}...` : conversation.latestPreview;
  }

  if (conversation.type === 'channel') {
    return 'Tap to view channel messages';
  }
  if (conversation.type === 'group') {
    return 'Tap to view group messages';
  }

  return 'Tap to start chatting';
}

function getAvatarText(name: string) {
  const cleaned = name.replace('#', '').trim();
  const parts = cleaned.split(' ');
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return cleaned.slice(0, 2).toUpperCase();
}

function getUnreadCount(conversation: Conversation, isActive: boolean) {
  if (isActive) return 0;
  return conversation.unreadCount ?? 0;
}

export default function ChatSidebarList({
  conversations,
  activeConversationId,
  onSelectConversation,
  theme,
  messages,
  isMobile,
  currentUserId,
  onConversationCreated,
  viewType,
  externalSearch,
}: Props) {
  const [activeFilter, setActiveFilter] = useState<Filter>('All');
  const [search, setSearch] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [showNewGroup, setShowNewGroup] = useState(false);
  const [groupName, setGroupName] = useState('');
  const [selectedMembers, setSelectedMembers] = useState<string[]>([]);
const [groupCategories, setGroupCategories] = useState<GroupCategory[]>([]);
  const [activeCategory, setActiveCategory] = useState<string>('all');
  const [onlineUserIds, setOnlineUserIds] = useState<Set<string>>(new Set());
  const [favouriteIds, setFavouriteIds] = useState<Set<string>>(new Set());

  // Reset filter when view type changes
  useEffect(() => {
    setActiveFilter('All');
    setShowNewGroup(false);
    setActiveCategory('all');
  }, [viewType]);

  // Fetch real presence for all users
  useEffect(() => {
    let cancelled = false;
    fetchPresence()
      .then((data) => {
        if (cancelled) return;
        const items = Array.isArray(data) ? data : [data];
        const onlineIds = new Set(items.filter((p) => p.status === 'online').map((p) => p.userId));
        setOnlineUserIds(onlineIds);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  // Fetch real favourites from the backend — refetch whenever the
  // user opens the Favourites filter so toggles appear immediately.
  useEffect(() => {
    let cancelled = false;
    fetchFavouriteIds()
      .then((ids) => {
        if (!cancelled) setFavouriteIds(new Set(ids));
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [activeFilter]);

  // Fetch group templates when in groups view
  useEffect(() => {
    if (viewType === 'group' && groupCategories.length === 0) {
      fetchGroupTemplates()
        .then((cats) => setGroupCategories(cats))
        .catch(() => {});
    }
  }, [viewType, groupCategories.length]);

  useEffect(() => {
    if (activeFilter === 'Users' && users.length === 0 && !usersLoading) {
      setUsersLoading(true);
      fetchAllUsers()
        .then((fetchedUsers) => {
          setUsers(fetchedUsers.filter((u) => u.id !== currentUserId));
        })
        .catch(() => {})
        .finally(() => setUsersLoading(false));
    }
  }, [activeFilter, users.length, usersLoading, currentUserId]);

const filteredConversations = useMemo(() => {
    if (activeFilter === 'Users') return [];
    const lowercaseSearch = search.trim().toLowerCase();
    const externalLower = (externalSearch ?? '').trim().toLowerCase();

    return conversations
      .filter((conversation) => conversation.type === viewType)
      .filter((conversation) => {
        // Category filter for groups
        if (viewType === 'group' && activeCategory !== 'all' && conversation.category !== activeCategory) {
          return false;
        }
        if (activeFilter === 'Unread') {
          if (getUnreadCount(conversation, false) === 0) return false;
        }
        if (activeFilter === 'Favourites') {
          return favouriteIds.has(conversation.id);
        }
        return true;
      })
      .filter((conversation) => {
        if (!lowercaseSearch) return true;
        return conversation.name.toLowerCase().includes(lowercaseSearch);
      })
      .filter((conversation) => {
        if (!externalLower) return true;
        return (
          conversation.name.toLowerCase().includes(externalLower) ||
          (conversation.latestPreview ?? '').toLowerCase().includes(externalLower)
        );
      })
      .sort((a, b) => {
        const aUnread = getUnreadCount(a, a.id === activeConversationId);
        const bUnread = getUnreadCount(b, b.id === activeConversationId);
        if (aUnread > 0 && bUnread === 0) return -1;
        if (bUnread > 0 && aUnread === 0) return 1;
        if (a.id === activeConversationId) return -1;
        if (b.id === activeConversationId) return 1;
        return a.name.localeCompare(b.name);
      });
  }, [conversations, activeConversationId, activeFilter, search, viewType, activeCategory, externalSearch, favouriteIds]);

  const filteredUsers = useMemo(() => {
    const lowercaseSearch = search.trim().toLowerCase();
    const externalLower = (externalSearch ?? '').trim().toLowerCase();
    const combined = lowercaseSearch || externalLower;
    if (!combined) return users;
    return users.filter(
      (u) =>
        u.name.toLowerCase().includes(combined) ||
        (u.organizationId ?? '').toLowerCase().includes(combined)
    );
  }, [users, search, externalSearch]);

  useEffect(() => {
    if (activeFilter === 'Users') return;
    if (!filteredConversations.length) return;

    const conversationStillVisible = filteredConversations.some(
      (conversation) => conversation.id === activeConversationId
    );
    if (!conversationStillVisible) {
      onSelectConversation(filteredConversations[0].id);
    }
  }, [activeConversationId, filteredConversations, onSelectConversation, activeFilter]);

  const handleUserClick = async (user: User) => {
    try {
      const conv = await createDM(user.id, user.name);
      if (onConversationCreated) {
        onConversationCreated(conv);
      }
      onSelectConversation(conv.id);
      setActiveFilter('All');
    } catch {
      // ignore
    }
  };

  const toggleMember = (userId: string) => {
    setSelectedMembers((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId]
    );
  };

  const handleCreateGroup = async () => {
    const name = groupName.trim();
    if (!name || selectedMembers.length === 0) return;
    const groupId = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
    try {
      const conv = await createGroup(groupId, name, [...selectedMembers, currentUserId ?? 'user-001']);
      if (onConversationCreated) {
        onConversationCreated(conv);
      }
      onSelectConversation(conv.id);
      setShowNewGroup(false);
      setGroupName('');
      setSelectedMembers([]);
    } catch {
      // ignore
    }
  };

  const activePreview = messages.length > 0 ? messages[messages.length - 1] : undefined;
  const title = viewType === 'direct' ? 'Chats' : viewType === 'group' ? 'Groups' : 'Channels';

  return (
    <aside
      className={styles.sidebarList}
      style={{ background: 'transparent', color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}
    >
      <div className={styles.searchWrapper} style={{ background: theme === 'dark' ? '#0a2b45' : '#ffffff' }}>
        <div className={styles.searchBox} style={{ background: theme === 'dark' ? '#0f3f5f' : '#f0f7fb', borderColor: 'transparent' }}>
          <span className={styles.searchIcon}>🔍</span>
          <input
            type="text"
            className={styles.searchInput}
            placeholder={activeFilter === 'Users' ? 'Search users...' : `Search ${title.toLowerCase()}...`}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </div>
      </div>

      {viewType === 'direct' && (
        <div className={styles.filterRow}>
          {filters.map((filter) => (
            <button
              key={filter}
              type="button"
              className={`${styles.filterPill} ${activeFilter === filter ? styles.activeFilter : ''}`}
              onClick={() => setActiveFilter(filter)}
            >
              {filter}
            </button>
          ))}
        </div>
      )}

      {viewType === 'group' && (
        <div className={styles.filterRow}>
          <button
            type="button"
            className={`${styles.filterPill} ${showNewGroup ? styles.activeFilter : ''}`}
            onClick={() => setShowNewGroup((prev) => !prev)}
          >
            + New Group
          </button>
        </div>
      )}

      {viewType === 'group' && groupCategories.length > 0 && (
        <div style={{ padding: '0 16px 8px' }}>
          <select
            className={styles.select}
            value={activeCategory}
            onChange={(event) => setActiveCategory(event.target.value)}
            style={{
              width: '100%',
              padding: '8px 12px',
              borderRadius: 10,
              border: `1px solid ${theme === 'dark' ? '#6b7280' : '#d1d5db'}`,
              background: theme === 'dark' ? '#0f3f5f' : '#f0f7fb',
              color: theme === 'dark' ? '#e8eef4' : '#1a1a1a',
              fontSize: 13,
              fontWeight: 600,
              cursor: 'pointer',
              boxSizing: 'border-box',
            }}
          >
            <option value="all">All Categories</option>
            {groupCategories.map((cat) => (
              <option key={cat.id} value={cat.id}>
                {cat.icon} {cat.name} ({cat.groups?.length ?? 0})
              </option>
            ))}
          </select>
        </div>
      )}

      {showNewGroup && viewType === 'group' && (
        <div className={styles.newGroupCard} style={{ background: theme === 'dark' ? '#0f3f5f' : '#f0f7fb', borderColor: 'transparent' }}>
          <input
            type="text"
            className={styles.groupInput}
            placeholder="Group name (e.g. Statistics Team)"
            value={groupName}
            onChange={(event) => setGroupName(event.target.value)}
            style={{ color: theme === 'dark' ? '#e8eef4' : '#1a1a1a' }}
          />
          <div className={styles.groupMemberList}>
            {users.map((user) => (
              <label key={user.id} className={styles.memberRow}>
                <input
                  type="checkbox"
                  checked={selectedMembers.includes(user.id)}
                  onChange={() => toggleMember(user.id)}
                />
                <span>{user.name} <small style={{ opacity: 0.6 }}>· {user.organizationId}</small></span>
              </label>
            ))}
          </div>
          <button
            type="button"
            className={styles.createGroupButton}
            onClick={handleCreateGroup}
            disabled={!groupName.trim() || selectedMembers.length === 0}
          >
            Create Group
          </button>
        </div>
      )}

      <ul className={styles.conversationList}>
        {activeFilter === 'Users' ? (
          usersLoading ? (
            <div className={styles.emptyState}>Loading users...</div>
          ) : filteredUsers.length === 0 ? (
            <div className={styles.emptyState}>No users found</div>
          ) : (
            filteredUsers.map((user) => (
              <li key={user.id} className={styles.conversationItem}>
                <button
                  type="button"
                  className={styles.conversationButton}
                  onClick={() => handleUserClick(user)}
                >
                  <div className={styles.avatarWrapper}>
                    <div className={styles.avatarPlaceholder}>
                      {getAvatarText(user.name)}
                    </div>
                    {user.presence === 'online' && <span className={styles.onlineDot} />}
                  </div>
                  <div className={styles.conversationContent}>
                    <div className={styles.conversationTopRow}>
                      <span className={styles.conversationName}>{user.name}</span>
                    </div>
                    <div className={styles.conversationBottomRow}>
                      <span className={styles.conversationSnippet}>
                        {user.about || user.organizationId || 'Tap to start chatting'}
                      </span>
                    </div>
                  </div>
                </button>
              </li>
            ))
          )
        ) : filteredConversations.length === 0 ? (
          <div className={styles.emptyState}>No {title.toLowerCase()} yet</div>
        ) : (
          filteredConversations.map((conversation) => {
            const active = conversation.id === activeConversationId;
            const previewText = conversation.id === activeConversationId
              ? getPreviewText(conversation, activePreview)
              : getPreviewText(conversation);
            const unread = getUnreadCount(conversation, active);
            const timestamp = conversation.latestMessageAt
              ? formatTime(conversation.latestMessageAt)
              : conversation.id === activeConversationId && activePreview
              ? formatTime(activePreview.createdAt)
              : '';
const online = conversation.type === 'direct' &&
              (conversation.memberIds?.some((id) => id !== currentUserId && onlineUserIds.has(id)) ?? false);
            const isChannel = conversation.type === 'channel';
            const isGroup = conversation.type === 'group';

            return (
              <li key={conversation.id} className={styles.conversationItem}>
                <button
                  type="button"
                  className={`${styles.conversationButton} ${active ? styles.activeConversation : ''}`}
                  onClick={() => onSelectConversation(conversation.id)}
                >
                  <div className={styles.avatarWrapper}>
                    {isChannel ? (
                      <div className={styles.avatarPlaceholder} style={{ borderRadius: '12px' }}>
                        #
                      </div>
                    ) : isGroup ? (
                      <div className={styles.avatarPlaceholder} style={{ borderRadius: '12px' }}>
                        {getAvatarText(conversation.name).slice(0, 1)}
                      </div>
                    ) : (
                      <div className={styles.avatarPlaceholder}>
                        {getAvatarText(conversation.name)}
                      </div>
                    )}
                    {online && <span className={styles.onlineDot} />}
                  </div>

                  <div className={styles.conversationContent}>
                    <div className={styles.conversationTopRow}>
                      <span className={styles.conversationName}>{conversation.name}</span>
                      <span className={`${styles.timestamp} ${unread > 0 ? styles.timestampUnread : ''}`}>
                        {timestamp}
                      </span>
                    </div>
                    <div className={styles.conversationBottomRow}>
                      <span className={styles.conversationSnippet}>
                        {activePreview && conversation.id === activeConversationId && activePreview.sender && (
                          <span className={styles.snippetSender}>{activePreview.sender.split(' ')[0]}: </span>
                        )}
                        {previewText}
                      </span>
                      <div className={styles.badgeGroup}>
                        {isGroup && conversation.memberIds && (
                          <span className={styles.memberCount}>{conversation.memberIds.length}</span>
                        )}
                        {unread > 0 && <span className={styles.unreadBadge}>{unread}</span>}
                      </div>
                    </div>
                  </div>
                </button>
              </li>
            );
          })
        )}
      </ul>
    </aside>
  );
}