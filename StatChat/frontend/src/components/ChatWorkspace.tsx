import { useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import type { Conversation, Message, MessageAttachment, CallSession, CallParticipant } from '../types';
import { addReaction, removeReaction, pinMessage, unpinMessage, editChatMessage, deleteChatMessage, markMessageRead, clearConversation, muteConversation, unmuteConversation, toggleFavourite } from '../api/client';
import { useCall } from '../hooks/useCall';
import CallOverlay from './CallOverlay';
import AudioAttachment from './AudioAttachment';
import styles from './ChatWorkspace.module.css';

interface Props {
  conversation?: Conversation;
  messages: Message[];
  draft: string;
  theme: 'light' | 'dark';
  isMobile: boolean;
  currentUserName?: string;
  currentUserId?: string;
  onDraftChange: (value: string) => void;
  onSend: (textOverride?: string, replyContext?: { parentMessageId?: string; threadRootId?: string }) => void;
  onUploadAttachment?: (file: File, text?: string) => Promise<void> | void;
  onCloseMobile?: () => void;
  onMessageUpdate?: (updated: Message) => void;
  onMessageDelete?: (messageId: string) => void;
}

const quickEmojis = ['❤️', '😂', '👍', '😮', '😢', '🙏'];
const emojiPickerEmojis = ['😀', '😃', '😄', '😁', '😆', '😅', '🤣', '😂', '🙂', '🙃', '😉', '😊', '😇', '🥰', '😍', '🤩', '😘', '😗', '😚', '😙', '🥲', '😋', '😛', '😜', '🤪', '😝', '🤑', '🤗', '🤭', '🤫', '🤔', '🤐', '🤨', '😐', '😑', '😶', '😏', '😒', '🙄', '😬', '😮‍💨', '🥥', '🤥', '😌', '😔', '😪', '🤤', '😴', '😷', '🤒'];

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
  currentUserId,
  onDraftChange,
  onSend,
  onUploadAttachment,
  onCloseMobile,
  onMessageUpdate,
  onMessageDelete,
}: Props) {
  const [callActive, setCallActive] = useState(false);
  const [remoteStreams, setRemoteStreams] = useState<Record<string, MediaStream>>({});
  const {
    session,
    participants,
    localStream,
    micMuted,
    cameraOff,
    connecting,
    error: callError,
    startCall,
    joinCall,
    toggleMute,
    toggleCamera,
    hangUp,
    endCall,
  } = useCall({
    user: { id: currentUserId ?? 'user-001', name: currentUserName ?? 'StatChat User' },
    onIncomingRemoteStream: (stream, userId) => {
      setRemoteStreams((prev) => ({ ...prev, [userId]: stream }));
    },
    onRemoteLeave: (userId) => {
      setRemoteStreams((prev) => {
        const next = { ...prev };
        delete next[userId];
        return next;
      });
    },
    onCallEnded: () => {
      setCallActive(false);
      setRemoteStreams({});
    },
  });

  const handleStartCall = async (kind: 'voice' | 'video') => {
    try {
      setCallActive(true);
      await startCall(kind, conversation?.name, conversation?.id);
    } catch {
      setCallActive(false);
    }
  };
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [recording, setRecording] = useState(false);
  const [attachmentNote, setAttachmentNote] = useState<string | null>(null);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [typing, setTyping] = useState(false);
  const [reactions, setReactions] = useState<Record<string, Reaction[]>>({});
  const [activeQuickReactions, setActiveQuickReactions] = useState<string | null>(null);
  const [activeMessageActions, setActiveMessageActions] = useState<string | null>(null);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [pendingVoiceNote, setPendingVoiceNote] = useState<{ blob: Blob; url: string } | null>(null);
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
      const target = event.target as HTMLElement;
      if (menuRef.current && !menuRef.current.contains(target)) {
        setMenuOpen(false);
        setActiveQuickReactions(null);
      }
      const isActionButton = target.closest(`.${styles.messageActionButton}`);
      const isActionsPopover = target.closest(`.${styles.messageActionsPopover}`);
      if (!isActionButton && !isActionsPopover) {
        setActiveMessageActions(null);
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
            reacted: message.reactions?.some((r) => r.emoji === emoji && r.userId === currentUserId) ?? false,
          }));
          if (JSON.stringify(merged) !== JSON.stringify(existing)) {
            next[message.id] = merged;
            changed = true;
          }
        }
      }
      return changed ? next : prev;
    });
  }, [messages, currentUserId]);

  // Mark incoming messages as read when the conversation is visible
  useEffect(() => {
    if (!conversation || !messages.length) return;
    const incoming = messages.filter((m) => m.sender !== currentUserName && m.sender !== 'StatChat User');
    if (incoming.length) {
      Promise.all(incoming.map((message) => markMessageRead(message.id))).catch(() => {});
    }
  }, [conversation, messages, currentUserName]);

  useEffect(() => {
    return () => {
      if (pendingVoiceNote) {
        URL.revokeObjectURL(pendingVoiceNote.url);
      }
    };
  }, [pendingVoiceNote]);

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

  const sendPendingVoiceNote = async () => {
    if (!pendingVoiceNote) return;
    const file = new File([pendingVoiceNote.blob], `voice-note-${Date.now()}.webm`, { type: 'audio/webm' });
    const noteText = draft.trim() ? draft.trim() : 'Voice note';
    try {
      await onUploadAttachment?.(file, noteText);
      setAttachmentNote('🎙️ Voice note sent');
      if (onDraftChange) onDraftChange('');
    } catch (error) {
      console.error('Failed to upload voice note', error);
      setAttachmentNote('⚠️ Could not send voice note');
    } finally {
      URL.revokeObjectURL(pendingVoiceNote.url);
      setPendingVoiceNote(null);
    }
  };

  const cancelPendingVoiceNote = () => {
    if (pendingVoiceNote) {
      URL.revokeObjectURL(pendingVoiceNote.url);
    }
    setPendingVoiceNote(null);
    setAttachmentNote('⚠️ Voice note cancelled');
  };

  const toggleVoiceNote = async () => {
    if (recording) {
      mediaRecorderRef.current?.stop();
      if (recordingTimerRef.current) {
        window.clearInterval(recordingTimerRef.current);
        recordingTimerRef.current = null;
      }
      setRecording(false);
      setAttachmentNote('🎙️ Voice note ready. Tap send to send, cancel to discard.');
      return;
    }

    if (pendingVoiceNote) {
      cancelPendingVoiceNote();
      return;
    }

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

      mediaRecorder.onstop = () => {
        stream.getTracks().forEach((track) => track.stop());
        const blob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
        const url = URL.createObjectURL(blob);
        setPendingVoiceNote({ blob, url });
        setRecordingSeconds(0);
      };

      mediaRecorder.start();
      setRecording(true);
      setRecordingSeconds(0);
      setAttachmentNote('🎙️ Recording... click the mic again to stop');

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
    const previousReactions = reactions[messageId] || [];
    const found = previousReactions.find((r) => r.emoji === emoji);
    const remove = found?.reacted ?? false;

    setReactions((prev) => {
      const existing = prev[messageId] || [];
      const foundPrev = existing.find((r) => r.emoji === emoji);
      if (foundPrev) {
        return {
          ...prev,
          [messageId]: existing
            .map((r) =>
              r.emoji === emoji
                ? { ...r, count: r.reacted ? r.count - 1 : r.count + 1, reacted: !r.reacted }
                : r
            )
            .filter((r) => r.count > 0),
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
      if (remove) {
        await removeReaction(messageId, emoji, currentUserId);
      } else {
        await addReaction(messageId, emoji, currentUserId);
      }
    } catch (error) {
      console.error('Failed to persist reaction', error);
      setAttachmentNote('⚠️ Could not update reaction');
      setReactions((prev) => ({
        ...prev,
        [messageId]: previousReactions,
      }));
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

  const [muted, setMuted] = useState(false);

  const handleToggleMute = async () => {
    if (!conversation) return;
    const next = !muted;
    setMuted(next);
    setMenuOpen(false);
    try {
      if (next) {
        await muteConversation(conversation.id);
      } else {
        await unmuteConversation(conversation.id);
      }
      setAttachmentNote(next ? '🔔 Notifications muted for this conversation' : '🔔 Notifications unmuted');
    } catch {
      setMuted(!next);
      setAttachmentNote('⚠️ Could not update mute setting');
    }
  };

  const handleToggleFavourite = async () => {
    if (!conversation) return;
    setMenuOpen(false);
    try {
      const result = await toggleFavourite(conversation.id);
      setAttachmentNote(result.favourite ? '⭐ Added to favourites' : '⭐ Removed from favourites');
    } catch {
      setAttachmentNote('⚠️ Could not update favourites');
    }
  };

  const handleClearChat = async () => {
    if (!conversation || !messages.length) return;
    if (!window.confirm('Clear all messages in this conversation?')) return;
    setMenuOpen(false);
    try {
      await clearConversation(conversation.id);
      // Remove all messages locally so the UI reflects the cleared conversation
      for (const message of messages) {
        onMessageDelete?.(message.id);
      }
      setAttachmentNote('🗑️ Chat cleared');
    } catch {
      setAttachmentNote('⚠️ Could not clear chat');
    }
  };

  const handleSend = () => {
    if (editingMessage) {
      handleEditMessage();
      return;
    }
    if (pendingVoiceNote) {
      sendPendingVoiceNote();
      return;
    }

    if (replyTo) {
      const replyContext = {
        parentMessageId: replyTo.parentMessageId ?? replyTo.id,
        threadRootId: replyTo.threadRootId ?? replyTo.id,
      };
      onSend(draft, replyContext);
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
    const isAudio = attachment.mimeType?.startsWith('audio/') || /\.(mp3|wav|ogg|m4a|aac|flac|opus)$/i.test(attachment.fileName);
    const isVideo = attachment.mimeType?.startsWith('video/') || /\.(mp4|mov|avi)$/i.test(attachment.fileName);
    const isImage = attachment.mimeType?.startsWith('image/') || /\.(png|jpg|jpeg|gif|webp|svg)$/i.test(attachment.fileName);

    if (isAudio) {
      return <AudioAttachment key={attachment.id} attachment={attachment} />;
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
          <button
            type="button"
            className={styles.iconButton}
            title="Voice call"
            onClick={() => handleStartCall('voice')}
          >
            📞
          </button>
          <button
            type="button"
            className={styles.iconButton}
            title="Video call"
            onClick={() => handleStartCall('video')}
          >
            🎥
          </button>
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
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={handleToggleMute}>🔔 Mute notifications</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={handleToggleFavourite}>⭐ Add to favourites</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={handleExportChat}>📊 Export chat</button>
                <button
                  type="button"
                  className={`${styles.menuItem} ${styles.menuItemDanger}`}
                  onClick={handleClearChat}
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

                      <div className={styles.messageMetaWrapper}>
                        <div className={styles.messageMeta}>
                          <span>{formatMessageTime(message.createdAt)}</span>
                          {isOwnMessage && (
                            <span className={styles.receiptRead}>
                              {message.deliveryStatus === 'read' || (message.readBy && message.readBy.length > 0)
                                ? '✓✓'
                                : message.deliveryStatus === 'delivered' ? '✓✓' : '✓'}
                            </span>
                          )}
                        </div>
                        <button
                          type="button"
                          className={styles.messageActionButton}
                          onClick={(e) => {
                            e.stopPropagation();
                            setActiveMessageActions((current) => (current === message.id ? null : message.id));
                          }}
                          aria-label="Message actions"
                        >
                          ⋯
                        </button>
                        {activeMessageActions === message.id && (
                          <div className={styles.messageActionsPopover}>
                            <button
                              type="button"
                              className={styles.messageActionItem}
                              onClick={(e) => {
                                e.stopPropagation();
                                setReplyTo(message);
                                setActiveMessageActions(null);
                              }}
                            >
                              ↩️ Reply
                            </button>
                            {isOwnMessage && (
                              <button
                                type="button"
                                className={styles.messageActionItem}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  startEdit(message);
                                  setActiveMessageActions(null);
                                }}
                              >
                                ✏️ Edit
                              </button>
                            )}
                            {isOwnMessage && (
                              <button
                                type="button"
                                className={`${styles.messageActionItem} ${styles.messageActionDanger}`}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  if (window.confirm('Delete this message for everyone?')) {
                                    handleDeleteMessage(message.id, { forEveryone: true });
                                  }
                                  setActiveMessageActions(null);
                                }}
                              >
                                🗑️ Delete
                              </button>
                            )}
                          </div>
                        )}
                      </div>

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

        {(recording || pendingVoiceNote) && (
          <div className={`${styles.inputHint} ${isDark ? styles.inputHintDark : ''}`}>
            {recording
              ? `Recording voice note — ${recordingSeconds}s`
              : 'Voice note ready. Send or cancel.'}
          </div>
        )}

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

        {editingMessage ? (
          <div className={styles.editToolbar}>
            <button type="button" className={styles.cancelButton} onClick={cancelEdit}>
              Cancel
            </button>
            <button type="button" className={styles.sendButton} onClick={handleSend} title="Save edit">
              Save
            </button>
          </div>
        ) : draft.trim() ? (
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
            className={`${styles.voiceButton} ${recording ? styles.voiceActive : ''}`}
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
      {pendingVoiceNote && (
        <div className={styles.voicePreview}>
          <audio controls src={pendingVoiceNote.url} className={styles.voicePreviewAudio} />
          <div className={styles.voicePreviewActions}>
            <button type="button" className={styles.sendButton} onClick={sendPendingVoiceNote}>
              Send voice note
            </button>
            <button type="button" className={styles.cancelButton} onClick={cancelPendingVoiceNote}>
              Cancel
            </button>
          </div>
        </div>
      )}

      {callActive && session && (
        <CallOverlay
          session={session}
          participants={participants}
          localStream={localStream}
          remoteStreams={remoteStreams}
          micMuted={micMuted}
          cameraOff={cameraOff}
          connecting={connecting}
          currentUserName={currentUserName}
          onToggleMute={toggleMute}
          onToggleCamera={toggleCamera}
          onHangUp={() => {
            setCallActive(false);
            hangUp();
          }}
          onEndCall={() => {
            setCallActive(false);
            endCall();
          }}
        />
      )}
    </section>
  );
}
