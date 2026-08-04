import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import type { Conversation, Message, MessageAttachment } from '../types';
import { addReaction, removeReaction, pinMessage, unpinMessage, editChatMessage, deleteChatMessage, markMessageRead } from '../api/client';
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
  onUploadAttachment?: (file: File, text?: string) => Promise<void> | void;
  onCloseMobile?: () => void;
  onMessageUpdate?: (updated: Message) => void;
  onMessageDelete?: (messageId: string) => void;
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
  onUploadAttachment,
  onCloseMobile,
  onMessageUpdate,
  onMessageDelete,
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
  const [recordingSeconds, setRecordingSeconds] = useState(0);
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef = useRef<Blob[]>([]);
  const recordingTimerRef = useRef<number | null>(null);
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

// Sync reactions from backend messages into local reaction state
  useEffect(() => {
    if (!messages.length) return;
    setReactions((prev) => {
      const next = { ...prev };
      let changed = false;
      for (const message of messages) {
        if (message.reactions && message.reactions.length > 0) {
          const existing = next[message.id] || [];
          const counts = new Map<string, number>();
          for (const r of existing) counts.set(r.emoji, r.count);
          for (const reaction of message.reactions) {
            counts.set(reaction.emoji, (counts.get(reaction.emoji) ?? 0) + 1);
          }
          const merged: Reaction[] = Array.from(counts.entries()).map(([emoji, count]) => ({
            emoji,
            count,
            reacted: message.reactions?.some((r) => r.emoji === emoji && r.userId === currentUserName) ?? false,
          }));
          if (JSON.stringify(merged) !== JSON.stringify(existing)) {
            next[message.id] = merged;
            changed = true;
          }
        }
      }
      return changed ? next : prev;
    });
  }, [messages, currentUserName]);

  // Mark incoming messages as read when the conversation is visible
  useEffect(() => {
    if (!conversation || !messages.length) return;
    const incoming = messages.filter((m) => m.sender !== currentUserName && m.sender !== 'StatChat User');
    if (incoming.length) {
      markMessageRead(incoming[incoming.length - 1].id).catch(() => {});
    }
  }, [conversation, messages, currentUserName]);

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

  const handleAttachment = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setAttachmentNote(`Uploading ${file.name}...`);
    try {
      await onUploadAttachment?.(file, draft.trim() ? draft : undefined);
      setAttachmentNote(`📎 ${file.name} uploaded`);
    } catch (error) {
      console.error('Failed to upload attachment', error);
      setAttachmentNote(`⚠️ Could not upload ${file.name}`);
    }
    event.target.value = '';
  };

  const toggleVoiceNote = async () => {
    if (recording) {
      // Stop recording and upload the audio blob as a real attachment
      mediaRecorderRef.current?.stop();
      if (recordingTimerRef.current) {
        window.clearInterval(recordingTimerRef.current);
        recordingTimerRef.current = null;
      }
      setRecording(false);
      setAttachmentNote('🎙️ Processing voice note...');
      return;
    }

    // Start recording via MediaRecorder API
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mediaRecorder = new MediaRecorder(stream);
      mediaRecorderRef.current = mediaRecorder;
      audioChunksRef.current = [];

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunksRef.current.push(event.data);
        }
      };

      mediaRecorder.onstop = async () => {
        stream.getTracks().forEach((track) => track.stop());
        const blob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
        const file = new File([blob], `voice-note-${Date.now()}.webm`, { type: 'audio/webm' });
        setRecordingSeconds(0);
        try {
          await onUploadAttachment?.(file);
          setAttachmentNote('🎙️ Voice note sent');
        } catch (error) {
          console.error('Failed to upload voice note', error);
          setAttachmentNote('⚠️ Could not send voice note');
        }
      };

      mediaRecorder.start();
      setRecording(true);
      setRecordingSeconds(0);
      setAttachmentNote('🎙️ Recording... (click to stop)');

      recordingTimerRef.current = window.setInterval(() => {
        setRecordingSeconds((prev) => prev + 1);
      }, 1000);
    } catch (error) {
      console.error('Microphone access denied', error);
      setAttachmentNote('⚠️ Microphone access denied');
    }
  };

  const handlePinMessage = async (messageId: string) => {
    if (!conversation) return;
    try {
      const message = messages.find((m) => m.id === messageId);
      if (message?.pinned) {
        await unpinMessage(conversation.id, messageId);
      } else {
        await pinMessage(conversation.id, messageId);
      }
      setMenuOpen(false);
      setAttachmentNote(message?.pinned ? '📌 Message unpinned' : '📌 Message pinned');
    } catch (error) {
      console.error('Failed to toggle pin', error);
      setAttachmentNote('⚠️ Could not update pin');
    }
  };

  const toggleReaction = async (messageId: string, emoji: string) => {
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

    // Persist to backend
    try {
      const existing = reactions[messageId] || [];
      const found = existing.find((r) => r.emoji === emoji);
      if (found && found.reacted) {
        await removeReaction(messageId, emoji);
      } else {
        await addReaction(messageId, emoji);
      }
    } catch (error) {
      console.error('Failed to persist reaction', error);
    }
  };

const [editingMessage, setEditingMessage] = useState<Message | null>(null);

  const startEdit = (message: Message) => {
    setEditingMessage(message);
    if (onDraftChange) {
      onDraftChange(message.text);
    }
    setMenuOpen(false);
    setActiveQuickReactions(null);
  };

  const cancelEdit = () => {
    setEditingMessage(null);
    if (onDraftChange) {
      onDraftChange('');
    }
  };

  const handleEditMessage = async () => {
    if (!editingMessage) return;
    const text = draft.trim();
    if (!text) return;
    try {
      const updated = await editChatMessage(editingMessage.id, text);
      onMessageUpdate?.(updated);
      setEditingMessage(null);
      onDraftChange('');
      setAttachmentNote('✏️ Message updated');
    } catch (error) {
      console.error('Failed to edit message', error);
      setAttachmentNote('⚠️ Could not update message');
    }
  };

  const handleDeleteMessage = async (messageId: string, options?: { forEveryone?: boolean }) => {
    try {
      await deleteChatMessage(messageId);
      onMessageDelete?.(messageId);
      setMenuOpen(false);
      setAttachmentNote(options?.forEveryone ? '🗑️ Message deleted for everyone' : '🗑️ Message deleted');
    } catch (error) {
      console.error('Failed to delete message', error);
      setAttachmentNote('⚠️ Could not delete message');
    }
  };

  const handleExportChat = () => {
    if (!messages.length) return;
    const lines = messages.map((m) => `[${new Date(m.createdAt).toLocaleString()}] ${m.sender}: ${m.text}`).join('\n');
    const blob = new Blob([lines], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `statchat-${conversation?.name ?? 'chat'}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    setMenuOpen(false);
    setAttachmentNote('📊 Chat exported');
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

  const renderAttachment = (attachment: MessageAttachment) => {
    const isAudio = attachment.mimeType?.startsWith('audio/') || /\.(mp3|wav|ogg|m4a|aac|flac)$/i.test(attachment.fileName);
    const isVideo = attachment.mimeType?.startsWith('video/') || /\.(mp4|mov|webm|avi)$/i.test(attachment.fileName);
    const isImage = attachment.mimeType?.startsWith('image/') || /\.(png|jpg|jpeg|gif|webp|svg)$/i.test(attachment.fileName);

    if (isAudio) {
      return (
        <div key={attachment.id}>
          <audio controls src={attachment.url} style={{ width: '100%', maxWidth: 320 }} />
          <div style={{ fontSize: '12px', opacity: 0.8, marginTop: '4px' }}>{attachment.fileName}</div>
        </div>
      );
    }

    if (isVideo) {
      return (
        <div key={attachment.id}>
          <video controls src={attachment.url} style={{ width: '100%', maxWidth: 360, borderRadius: '8px' }} />
          <div style={{ fontSize: '12px', opacity: 0.8, marginTop: '4px' }}>{attachment.fileName}</div>
        </div>
      );
    }

    if (isImage) {
      return (
        <div key={attachment.id}>
          <img src={attachment.url} alt={attachment.fileName} style={{ maxWidth: '100%', maxHeight: 240, borderRadius: '8px' }} />
          <div style={{ fontSize: '12px', opacity: 0.8, marginTop: '4px' }}>{attachment.fileName}</div>
        </div>
      );
    }

    return (
      <div key={attachment.id} style={{ fontSize: '13px', opacity: 0.9 }}>
        <a href={attachment.url} target="_blank" rel="noreferrer" style={{ color: isDark ? '#7dd3fc' : '#0f766e' }}>
          {attachment.fileName}
        </a>
      </div>
    );
  };

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
                <button
                  type="button"
                  className={styles.menuItem}
                  style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}
                  onClick={() => {
                    const lastMessage = messages[messages.length - 1];
                    if (lastMessage) handlePinMessage(lastMessage.id);
                  }}
                >
                  📌 Pin last message
                </button>
<button
                  type="button"
                  className={styles.menuItem}
                  style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}
                  onClick={() => {
                    const lastMessage = messages[messages.length - 1];
                    if (lastMessage) startEdit(lastMessage);
                  }}
                >
                  ✏️ Edit last message
                </button>
                <button
                  type="button"
                  className={styles.menuItem}
                  style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}
                  onClick={() => {
                    const lastMessage = messages[messages.length - 1];
                    if (lastMessage) handleDeleteMessage(lastMessage.id, { forEveryone: true });
                  }}
                >
                  🗑️ Delete last message
                </button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={() => { setMenuOpen(false); setAttachmentNote('🔔 Notifications muted for this conversation'); }}>🔔 Mute notifications</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={() => { setMenuOpen(false); setAttachmentNote('🖼️ Wallpaper settings'); }}>🖼️ Wallpaper</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={handleExportChat}>📊 Export chat</button>
                <button
                  type="button"
                  className={`${styles.menuItem} ${styles.menuItemDanger}`}
                  onClick={() => {
                    if (messages.length && window.confirm('Clear all messages in this conversation?')) {
                      handleDeleteMessage(messages[messages.length - 1].id, { forEveryone: true });
                    }
                  }}
                >
                  🗑️ Clear chat
                </button>
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
                      onDoubleClick={() => toggleReaction(message.id, '❤️')}
                    >
                      {showSender && <div className={styles.senderName}>{message.sender}</div>}
                      <div className={styles.messageText}>{message.text}</div>
                      {message.attachments?.length ? (
                        <div style={{ marginTop: '8px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                          {message.attachments.map((attachment) => renderAttachment(attachment))}
                        </div>
                      ) : null}
<div className={`${styles.messageMeta} ${isDark ? styles.metaDark : ''}`}>
                        <span>{formatMessageTime(message.createdAt)}</span>
                        {isOwnMessage && (
                          <span className={styles.receiptRead}>
                            {message.deliveryStatus === 'read' || (message.readBy && message.readBy.length > 0)
                              ? '✓✓'
                              : message.deliveryStatus === 'delivered' ? '✓✓' : '✓'}
                          </span>
                        )}
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
                                toggleReaction(message.id, emoji);
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
                                toggleReaction(message.id, r.emoji);
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
        <input ref={fileInputRef} type="file" className={styles.hiddenInput} accept="audio/*,video/*,image/*,.pdf,.doc,.docx,.txt" onChange={handleAttachment} />
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
            {recording ? `■ ${recordingSeconds}s` : '🎙️'}
          </button>
        )}
      </footer>

      {attachmentNote && (
        <div className={styles.attachmentHint}>{attachmentNote}</div>
      )}
    </section>
  );
}