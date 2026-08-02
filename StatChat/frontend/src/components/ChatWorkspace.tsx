import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import type { Conversation, Message } from '../types';
import styles from './ChatWorkspace.module.css';

interface Props {
  conversation?: Conversation;
  messages: Message[];
  draft: string;
  theme: 'light' | 'dark';
  isMobile: boolean;
  currentUserName?: string;
  onDraftChange: (value: string) => void;
  onSend: (textOverride?: string) => void;
  onCloseMobile?: () => void;
}

const quickEmojis = ['❤️', '😂', '👍', '😮', '😢', '🙏'];
const emojiPickerEmojis = ['😀', '😃', '😄', '😁', '😆', '😅', '🤣', '😂', '🙂', '🙃', '😉', '😊', '😇', '🥰', '😍', '🤩', '😘', '😗', '😚', '😙', '🥲', '😋', '😛', '😜', '🤪', '😝', '🤑', '🤗', '🤭', '🤫', '🤔', '🤐', '🤨', '😐', '😑', '😶', '😏', '😒', '🙄', '😬', '😮‍💨', '🥥', '🤥', '😌', '😔', '😪', '🤤', '😴', '😷', '🤒', '🤕', '🤢', '🤮', '🤧', '🥵', '🥶', '🥴', '😵', '🤯', '🤠', '🥳', '🥸', '😎', '🤓', '🧐'];

function formatMessageTime(timestamp: string) {
  const date = new Date(timestamp);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function formatDateLabel(timestamp: string) {
  const date = new Date(timestamp);
  const now = new Date();
  const diffDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));
  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  return date.toLocaleDateString([], { weekday: 'long', month: 'short', day: 'numeric' });
}

function getAvatarText(name: string) {
  const cleaned = name.replace('#', '').trim();
  const parts = cleaned.split(' ');
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return cleaned.slice(0, 2).toUpperCase();
}

interface Reaction {
  emoji: string;
  count: number;
  reacted: boolean;
}

export default function ChatWorkspace({
  conversation,
  messages,
  draft,
  theme,
  isMobile,
  currentUserName,
  onDraftChange,
  onSend,
  onCloseMobile,
}: Props) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [recording, setRecording] = useState(false);
  const [attachmentNote, setAttachmentNote] = useState<string | null>(null);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [typing, setTyping] = useState(false);
  const [reactions, setReactions] = useState<Record<string, Reaction[]>>({});
  const [activeQuickReactions, setActiveQuickReactions] = useState<string | null>(null);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const streamRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLTextAreaElement | null>(null);

  const isDark = theme === 'dark';
  const isChannel = conversation?.type === 'channel';

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false);
        setActiveQuickReactions(null);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (streamRef.current) {
      streamRef.current.scrollTop = streamRef.current.scrollHeight;
    }
  }, [messages]);

  // Simulate typing indicator
  useEffect(() => {
    if (messages.length === 0) return;
    const lastMessage = messages[messages.length - 1];
    if (lastMessage.sender !== currentUserName && lastMessage.sender !== 'StatChat User') {
      setTyping(true);
      const timer = setTimeout(() => setTyping(false), 2000);
      return () => clearTimeout(timer);
    }
  }, [messages, currentUserName]);

  // Auto-resize textarea
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto';
      inputRef.current.style.height = `${Math.min(inputRef.current.scrollHeight, 100)}px`;
    }
  }, [draft]);

  const visibleMessages = useMemo(() => {
    const normalized = searchQuery.trim().toLowerCase();
    if (!normalized) return messages;
    return messages.filter((message) =>
      [message.sender, message.text].some((value) => value.toLowerCase().includes(normalized))
    );
  }, [messages, searchQuery]);

  // Group messages by date
  const messageGroups = useMemo(() => {
    const groups: Array<{ date: string; messages: Message[] }> = [];
    let currentDate = '';
    for (const message of visibleMessages) {
      const dateKey = new Date(message.createdAt).toDateString();
      if (dateKey !== currentDate) {
        currentDate = dateKey;
        groups.push({ date: message.createdAt, messages: [] });
      }
      groups[groups.length - 1].messages.push(message);
    }
    return groups;
  }, [visibleMessages]);

  const handleAttachment = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    setAttachmentNote(`📎 ${file.name}`);
    onSend(`📎 ${file.name}`);
    event.target.value = '';
  };

  const toggleVoiceNote = () => {
    const nextValue = !recording;
    setRecording(nextValue);
    if (!nextValue) {
      onSend('🎙️ Voice note sent');
      setAttachmentNote('🎙️ Voice note sent');
    } else {
      setAttachmentNote('🎙️ Recording...');
    }
  };

  const addReaction = (messageId: string, emoji: string) => {
    setReactions((prev) => {
      const existing = prev[messageId] || [];
      const found = existing.find((r) => r.emoji === emoji);
      if (found) {
        return {
          ...prev,
          [messageId]: existing.map((r) =>
            r.emoji === emoji
              ? { ...r, count: r.reacted ? r.count - 1 : r.count + 1, reacted: !r.reacted }
              : r
          ).filter((r) => r.count > 0),
        };
      }
      return {
        ...prev,
        [messageId]: [...existing, { emoji, count: 1, reacted: true }],
      };
    });
    setActiveQuickReactions(null);
  };

  const handleSend = () => {
    if (replyTo) {
      onSend(replyTo ? `↩️ Replying to ${replyTo.sender.split(' ')[0]}: ${draft}` : draft);
      setReplyTo(null);
    } else {
      onSend();
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
  };

  const addEmoji = (emoji: string) => {
    onDraftChange(`${draft}${emoji}`);
    setShowEmojiPicker(false);
    inputRef.current?.focus();
  };

  const headerAvatarText = conversation ? getAvatarText(conversation.name) : '?';

  return (
    <section className={styles.workspace} style={{ background: 'transparent', color: isDark ? '#e8eef4' : '#1a1a1a' }}>
      <header className={styles.chatHeader} style={{ background: isDark ? '#0a2b45' : '#ffffff' }}>
        <div className={styles.chatHeaderLeft}>
          {isMobile && onCloseMobile && (
            <button type="button" className={styles.backButton} onClick={onCloseMobile} title="Back">
              ←
            </button>
          )}
          <div className={`${styles.headerAvatar} ${isChannel ? styles.headerAvatarChannel : ''}`}>
            {isChannel ? '#' : headerAvatarText}
          </div>
          <div className={styles.headerInfo}>
            <h2 className={styles.chatName}>{conversation?.name ?? 'Select a conversation'}</h2>
            <p className={`${styles.chatStatus} ${conversation?.type === 'channel' ? styles.chatStatusOffline : ''}`}>
              {conversation?.type === 'direct' ? '● online' : conversation?.type === 'channel' ? `${conversation.memberIds?.length ?? 0} members` : 'Select a chat'}
            </p>
          </div>
        </div>
        <div className={styles.headerActions}>
          <div className={styles.searchWrapper}>
            <input
              type="search"
              className={styles.searchInput}
              placeholder="Search"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              style={{ background: isDark ? '#0f3f5f' : '#f0f7fb', color: isDark ? '#e8eef4' : '#1a1a1a' }}
            />
          </div>
          <button type="button" className={styles.iconButton} title="Voice call">📞</button>
          <button type="button" className={styles.iconButton} title="Video call">🎥</button>
          <div className={styles.menuWrapper} ref={menuRef}>
            <button
              type="button"
              className={styles.iconButton}
              title="More options"
              onClick={() => setMenuOpen((open) => !open)}
            >
              ⋮
            </button>
            {menuOpen && (
              <div className={styles.menuPopover} style={{ background: isDark ? '#0f3f5f' : '#ffffff' }}>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}>📌 Pin message</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}>⭐ Star messages</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}>🔔 Mute notifications</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}>🖼️ Wallpaper</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}>📊 Export chat</button>
                <button type="button" className={`${styles.menuItem} ${styles.menuItemDanger}`}>🗑️ Clear chat</button>
              </div>
            )}
          </div>
        </div>
      </header>

      <div
        ref={streamRef}
        className={`${styles.messageStream} ${isDark ? styles.messageStreamDark : ''}`}
      >
        <div className={styles.systemBanner}>
          <div className={`${styles.systemBannerInner} ${isDark ? styles.systemBannerInnerDark : ''}`}>
            🔒 Messages are end-to-end encrypted within StatGate
          </div>
        </div>

        {visibleMessages.length === 0 ? (
          <div className={styles.emptyState}>
            No messages yet. Start the conversation!
          </div>
        ) : (
          messageGroups.map((group, groupIndex) => (
            <div key={groupIndex}>
              <div className={styles.dateSeparator}>
                <span className={`${styles.datePill} ${isDark ? styles.datePillDark : ''}`}>
                  {formatDateLabel(group.date)}
                </span>
              </div>
              {group.messages.map((message, msgIndex) => {
                const isOwnMessage = message.sender === currentUserName || message.sender === 'StatChat User';
                const prevMessage = msgIndex > 0 ? group.messages[msgIndex - 1] : null;
                const showSender = !isOwnMessage && (!prevMessage || prevMessage.sender !== message.sender);
                const msgReactions = reactions[message.id] || [];

                return (
                  <div
                    key={message.id}
                    className={`${styles.messageRow} ${isOwnMessage ? styles.messageRowSent : styles.messageRowReceived}`}
                  >
                    <div
                      className={`${styles.messageBubble} ${isOwnMessage ? (isDark ? styles.sentDark : styles.sent) : (isDark ? styles.receivedDark : styles.received)}`}
                      onClick={() => setActiveQuickReactions(activeQuickReactions === message.id ? null : message.id)}
                      onDoubleClick={() => addReaction(message.id, '❤️')}
                    >
                      {showSender && <div className={styles.senderName}>{message.sender}</div>}
                      <div className={styles.messageText}>{message.text}</div>
                      <div className={`${styles.messageMeta} ${isDark ? styles.metaDark : ''}`}>
                        <span>{formatMessageTime(message.createdAt)}</span>
                        {isOwnMessage && <span className={styles.receiptRead}>✓✓</span>}
                      </div>

                      {/* Quick reactions — TikTok style */}
                      {activeQuickReactions === message.id && (
                        <div className={`${styles.quickReactions} ${styles.quickReactionsVisible}`}>
                          {quickEmojis.map((emoji) => (
                            <button
                              key={emoji}
                              type="button"
                              className={styles.quickReactionBtn}
                              onClick={(e) => {
                                e.stopPropagation();
                                addReaction(message.id, emoji);
                              }}
                            >
                              {emoji}
                            </button>
                          ))}
                        </div>
                      )}

                      {/* Reactions bar — X style */}
                      {msgReactions.length > 0 && (
                        <div className={styles.reactionsBar}>
                          {msgReactions.map((r) => (
                            <button
                              key={r.emoji}
                              type="button"
                              className={`${styles.reactionChip} ${r.reacted ? styles.reactionChipActive : ''}`}
                              onClick={(e) => {
                                e.stopPropagation();
                                addReaction(message.id, r.emoji);
                              }}
                              style={{
                                background: isDark ? 'rgba(30,41,59,0.9)' : 'rgba(255,255,255,0.95)',
                              }}
                            >
                              {r.emoji}
                              <span className={styles.reactionCount}>{r.count}</span>
                            </button>
                          ))}
                        </div>
                      )}

                      {/* Thread indicator — X style */}
                      <div
                        className={styles.threadIndicator}
                        onClick={(e) => {
                          e.stopPropagation();
                          setReplyTo(message);
                        }}
                      >
                        <div className={styles.threadLine} />
                        Reply
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          ))
        )}

        {/* Typing indicator — WhatsApp style */}
        {typing && (
          <div className={styles.messageRowReceived}>
            <div className={`${styles.typingIndicator} ${isDark ? styles.typingIndicatorDark : ''}`}>
              <div className={styles.typingDots}>
                <span className={styles.typingDot} />
                <span className={styles.typingDot} />
                <span className={styles.typingDot} />
              </div>
              <span className={styles.typingText}>typing...</span>
            </div>
          </div>
        )}
      </div>

      {/* Reply indicator — X style */}
      {replyTo && (
        <div className={styles.replyIndicator} style={{ background: isDark ? '#0a2b45' : '#ffffff' }}>
          <div className={styles.replyIndicatorContent}>
            <span className={styles.replyIndicatorSender}>Replying to {replyTo.sender.split(' ')[0]}</span>
            <div className={styles.replyIndicatorText}>{replyTo.text}</div>
          </div>
          <button type="button" className={styles.replyCancel} onClick={() => setReplyTo(null)}>✕</button>
        </div>
      )}

      <footer className={styles.inputFooter} style={{ background: isDark ? '#0a2b45' : '#ffffff' }}>
        <input ref={fileInputRef} type="file" className={styles.hiddenInput} onChange={handleAttachment} />
        <button type="button" className={styles.attachButton} onClick={() => fileInputRef.current?.click()} title="Attach">
          ＋
        </button>
        <div className={`${styles.inputBar} ${isDark ? styles.inputBarDark : ''}`}>
          <button
            type="button"
            className={styles.emojiButton}
            onClick={() => setShowEmojiPicker((prev) => !prev)}
            title="Emoji"
          >
            😊
          </button>
          <textarea
            ref={inputRef}
            className={styles.messageInput}
            value={draft}
            placeholder="Type a message..."
            onChange={(event) => onDraftChange(event.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}
          />
        </div>

        {/* Emoji picker */}
        {showEmojiPicker && (
          <div className={`${styles.emojiPicker} ${isDark ? styles.emojiPickerDark : ''}`} style={{ position: 'absolute', bottom: 60, right: 60 }}>
            {emojiPickerEmojis.map((emoji) => (
              <button
                key={emoji}
                type="button"
                className={styles.emojiItem}
                onClick={() => addEmoji(emoji)}
              >
                {emoji}
              </button>
            ))}
          </div>
        )}

        {draft.trim() ? (
          <button
            type="button"
            className={styles.sendButton}
            onClick={handleSend}
            title="Send"
          >
            ➤
          </button>
        ) : (
          <button
            type="button"
            className={`${styles.iconButton} ${recording ? styles.voiceActive : ''}`}
            title={recording ? 'Stop recording' : 'Record voice note'}
            onClick={toggleVoiceNote}
          >
            {recording ? '■' : '🎙️'}
          </button>
        )}
      </footer>

      {attachmentNote && (
        <div className={styles.attachmentHint}>{attachmentNote}</div>
      )}
    </section>
  );
}