import { useDeferredValue, useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import type { Conversation, Message, MessageAttachment, CallSession, CallParticipant, User, ScheduledMessage, MediaCatalogItem, RetentionPolicy, LegalHold, ComplianceAuditEvent } from '../types';
import { addReaction, removeReaction, pinMessage, unpinMessage, editChatMessage, deleteChatMessage, forwardChatMessage, fetchScheduledMessages, scheduleChatMessage, cancelScheduledMessage, searchChatMedia, sendChatMedia, sendChatLocation, markMessageRead, clearConversation, muteConversation, unmuteConversation, toggleFavourite, requestChatAssistant, fetchConversationMembers, addConversationMember, removeConversationMember, fetchAllUsers, fetchSavedMessages, saveChatMessage, unsaveChatMessage, exportConversation, fetchRetentionPolicy, updateRetentionPolicy, enforceRetention, fetchLegalHolds, createLegalHold, releaseLegalHold, fetchComplianceAudit, removeCallParticipant, updateCallParticipantRole, type ConversationMembership } from '../api/client';
import { useCall } from '../hooks/useCall';
import CallOverlay from './CallOverlay';
import AudioAttachment from './AudioAttachment';
import styles from './ChatWorkspace.module.css';

interface Props {
  conversation?: Conversation;
  conversations: Conversation[];
  messages: Message[];
  draft: string;
  theme: 'light' | 'dark';
  isMobile: boolean;
  currentUserName?: string;
  currentUserId?: string;
  currentUserRoles?: string[];
  typingUsers?: string[];
  viewType?: 'direct' | 'group' | 'channel';
  onDraftChange: (value: string) => void;
  onSend: (textOverride?: string, replyContext?: { parentMessageId?: string; threadRootId?: string }, mentionContext?: { mentionUserIds?: string[]; mentionAll?: boolean }) => void;
  onUploadAttachment?: (file: File, text?: string) => Promise<void> | void;
  onCloseMobile?: () => void;
  onMessageUpdate?: (updated: Message) => void;
  onMessageCreated?: (created: Message) => void;
  onMessageDelete?: (messageId: string) => void;
  focusedMessageId?: string;
  onOpenMessage?: (conversationId: string, messageId: string) => void;
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

function mentionAlias(user: User) {
  const emailAlias = user.email?.split('@')[0]?.replace(/[^A-Za-z0-9._-]/g, '');
  return emailAlias || user.name.replace(/\s+/g, '.').replace(/[^A-Za-z0-9._-]/g, '');
}

interface Reaction {
  emoji: string;
  count: number;
  reacted: boolean;
}

export default function ChatWorkspace({
  conversation,
  conversations,
  messages,
  draft,
  theme,
  isMobile,
  currentUserName,
  currentUserId,
  currentUserRoles = [],
  typingUsers = [],
  onDraftChange,
  onSend,
  onUploadAttachment,
  onCloseMobile,
  onMessageUpdate,
  onMessageCreated,
  onMessageDelete,
  focusedMessageId,
  onOpenMessage,
  viewType = 'direct',
}: Props) {
  const [callActive, setCallActive] = useState(false);
  const [remoteStreams, setRemoteStreams] = useState<Record<string, MediaStream>>({});
  const {
    session,
    participants,
    localStream,
    screenStream,
    isScreenSharing,
    remoteScreenSharers,
    screenShareSupported,
    connectionState,
    quality,
    qualityMetrics,
    micMuted,
    cameraOff,
    connecting,
    error: callError,
    startCall,
    joinCall,
    toggleMute,
    toggleCamera,
    toggleScreenShare,
    requestMute,
    pendingMuteRequest,
    respondToMuteRequest,
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
  const [reactions, setReactions] = useState<Record<string, Reaction[]>>({});
  const [activeQuickReactions, setActiveQuickReactions] = useState<string | null>(null);
  const [activeMessageActions, setActiveMessageActions] = useState<string | null>(null);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [pendingVoiceNote, setPendingVoiceNote] = useState<{ blob: Blob; url: string; mimeType: string } | null>(null);
  const [recordingSeconds, setRecordingSeconds] = useState(0);
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [assistantLoading, setAssistantLoading] = useState(false);
  const [assistantResponse, setAssistantResponse] = useState('');
  const [membership, setMembership] = useState<ConversationMembership | null>(null);
  const [memberDirectory, setMemberDirectory] = useState<User[]>([]);
  const [memberManagerOpen, setMemberManagerOpen] = useState(false);
  const [selectedMemberId, setSelectedMemberId] = useState('');
  const [memberError, setMemberError] = useState('');
  const [memberBusy, setMemberBusy] = useState(false);
  const [forwardingMessage, setForwardingMessage] = useState<Message | null>(null);
  const [forwardTargetId, setForwardTargetId] = useState('');
  const [forwardError, setForwardError] = useState('');
  const [forwardBusy, setForwardBusy] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const [scheduledFor, setScheduledFor] = useState('');
  const [scheduledMessages, setScheduledMessages] = useState<ScheduledMessage[]>([]);
  const [scheduleError, setScheduleError] = useState('');
  const [scheduleBusy, setScheduleBusy] = useState(false);
  const [mediaPickerOpen, setMediaPickerOpen] = useState(false);
  const [mediaKind, setMediaKind] = useState<'gif' | 'sticker'>('gif');
  const [mediaQuery, setMediaQuery] = useState('');
  const deferredMediaQuery = useDeferredValue(mediaQuery);
  const [mediaItems, setMediaItems] = useState<MediaCatalogItem[]>([]);
  const [mediaBusy, setMediaBusy] = useState(false);
  const [mediaError, setMediaError] = useState('');
  const [locationOpen, setLocationOpen] = useState(false);
  const [pendingLocation, setPendingLocation] = useState<{ latitude: number; longitude: number; accuracyMeters: number } | null>(null);
  const [locationLabel, setLocationLabel] = useState('');
  const [locationBusy, setLocationBusy] = useState(false);
  const [locationError, setLocationError] = useState('');
  const [selectedMentionIds, setSelectedMentionIds] = useState<string[]>([]);
  const [mentionAll, setMentionAll] = useState(false);
  const [savedMessageIds, setSavedMessageIds] = useState<Set<string>>(new Set());
  const [savedMessages, setSavedMessages] = useState<Message[]>([]);
  const [savedPanelOpen, setSavedPanelOpen] = useState(false);
  const [savedBusy, setSavedBusy] = useState(false);
  const [savedError, setSavedError] = useState('');
  const [complianceOpen, setComplianceOpen] = useState(false);
  const [retentionPolicy, setRetentionPolicy] = useState<RetentionPolicy | null>(null);
  const [legalHolds, setLegalHolds] = useState<LegalHold[]>([]);
  const [complianceAudit, setComplianceAudit] = useState<ComplianceAuditEvent[]>([]);
  const [retentionDays, setRetentionDays] = useState(365);
  const [retentionEnabled, setRetentionEnabled] = useState(false);
  const [holdName, setHoldName] = useState('');
  const [holdReason, setHoldReason] = useState('');
  const [holdScope, setHoldScope] = useState<'tenant' | 'conversation'>('conversation');
  const [complianceBusy, setComplianceBusy] = useState(false);
  const [complianceError, setComplianceError] = useState('');
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef = useRef<Blob[]>([]);
  const recordingTimerRef = useRef<number | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const streamRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLTextAreaElement | null>(null);

  const isDark = theme === 'dark';
  const isChannel = conversation?.type === 'channel';
  const mentionMatch = draft.match(/@([A-Za-z0-9._-]*)$/);
  const mentionQuery = mentionMatch?.[1].toLowerCase() ?? '';
  const mentionCandidates = useMemo(() => {
    if (!mentionMatch || !conversation) return [];
    const memberIDs = new Set(conversation.type === 'channel' ? memberDirectory.map((user) => user.id) : (membership?.memberIds ?? conversation.memberIds ?? []));
    return memberDirectory
      .filter((candidate) => candidate.id !== currentUserId && memberIDs.has(candidate.id))
      .filter((candidate) => `${candidate.name} ${mentionAlias(candidate)}`.toLowerCase().includes(mentionQuery))
      .slice(0, 6);
  }, [conversation, currentUserId, memberDirectory, membership?.memberIds, mentionMatch, mentionQuery]);

  const handleComposerChange = (value: string) => {
    onDraftChange(value);
    setSelectedMentionIds((current) => current.filter((id) => {
      const mentionedUser = memberDirectory.find((candidate) => candidate.id === id);
      return mentionedUser ? value.includes(`@${mentionAlias(mentionedUser)}`) : false;
    }));
    if (!/@(?:all|channel)\b/i.test(value)) setMentionAll(false);
  };
  const handleRemoveCallParticipant = async (userId: string) => {
    if (!session) return;
    try {
      await removeCallParticipant(session.id, userId);
    } catch {
      setAttachmentNote('Could not remove that call participant.');
    }
  };

  const handleChangeCallParticipantRole = async (userId: string, role: 'moderator' | 'participant') => {
    if (!session) return;
    try {
      await updateCallParticipantRole(session.id, userId, role);
    } catch {
      setAttachmentNote('Could not update that call participant role.');
    }
  };

  const selectMention = (candidate?: User, selectAll = false) => {
    const atIndex = draft.lastIndexOf('@');
    if (atIndex < 0) return;
    const token = selectAll ? '@all' : `@${mentionAlias(candidate as User)}`;
    onDraftChange(`${draft.slice(0, atIndex)}${token} `);
    if (selectAll) {
      setMentionAll(true);
    } else if (candidate) {
      setSelectedMentionIds((current) => current.includes(candidate.id) ? current : [...current, candidate.id]);
    }
    inputRef.current?.focus();
  };

  const renderMessageText = (message: Message) => {
    const aliases = new Set((message.mentionUserIds ?? []).map((id) => {
      const mentionedUser = memberDirectory.find((candidate) => candidate.id === id);
      return mentionedUser ? `@${mentionAlias(mentionedUser)}`.toLowerCase() : '';
    }).filter(Boolean));
    return message.text.split(/(@[A-Za-z0-9._-]+)/g).map((part, index) => {
      const isMention = aliases.has(part.toLowerCase()) || (message.mentionAll && /^@(all|channel)$/i.test(part));
      return isMention ? <mark key={`${part}-${index}`} className={styles.mentionToken}>{part}</mark> : part;
    });
  };

  const openSavedMessages = async () => {
    setSavedPanelOpen(true);
    setSavedBusy(true);
    setSavedError('');
    try {
      const saved = await fetchSavedMessages();
      setSavedMessages(saved);
      setSavedMessageIds(new Set(saved.map((message) => message.id)));
    } catch (error) {
      setSavedError(error instanceof Error ? error.message : 'Unable to load saved messages');
    } finally {
      setSavedBusy(false);
    }
  };

  const toggleSavedMessage = async (message: Message) => {
    const isSaved = savedMessageIds.has(message.id);
    setSavedBusy(true);
    setSavedError('');
    try {
      if (isSaved) {
        await unsaveChatMessage(message.id);
        setSavedMessageIds((current) => { const next = new Set(current); next.delete(message.id); return next; });
        setSavedMessages((current) => current.filter((item) => item.id !== message.id));
        setAttachmentNote('Removed from saved messages');
      } else {
        const saved = await saveChatMessage(message.id);
        setSavedMessageIds((current) => new Set(current).add(message.id));
        setSavedMessages((current) => [{ ...message, savedAt: saved.savedAt }, ...current.filter((item) => item.id !== message.id)]);
        setAttachmentNote('Message saved privately');
      }
    } catch (error) {
      setSavedError(error instanceof Error ? error.message : 'Unable to update saved message');
    } finally {
      setSavedBusy(false);
      setActiveMessageActions(null);
    }
  };

  const handleForwardMessage = async () => {
    if (!forwardingMessage || !forwardTargetId) return;
    setForwardBusy(true);
    setForwardError('');
    try {
      await forwardChatMessage(forwardingMessage.id, forwardTargetId);
      setForwardingMessage(null);
      setForwardTargetId('');
      setAttachmentNote('Message forwarded');
    } catch (error) {
      setForwardError(error instanceof Error ? error.message : 'Unable to forward message');
    } finally {
      setForwardBusy(false);
    }
  };

  const openScheduleManager = async () => {
    if (!conversation) return;
    const localDefault = new Date(Date.now() + 5 * 60 * 1000);
    localDefault.setMinutes(localDefault.getMinutes() - localDefault.getTimezoneOffset());
    setScheduledFor(localDefault.toISOString().slice(0, 16));
    setScheduleError('');
    setScheduleOpen(true);
    try {
      setScheduledMessages(await fetchScheduledMessages(conversation.id));
    } catch (error) {
      setScheduleError(error instanceof Error ? error.message : 'Unable to load scheduled messages');
    }
  };

  const handleScheduleMessage = async () => {
    if (!conversation || !draft.trim() || !scheduledFor) return;
    setScheduleBusy(true);
    setScheduleError('');
    try {
      const created = await scheduleChatMessage(conversation.id, draft.trim(), new Date(scheduledFor).toISOString());
      setScheduledMessages((current) => [...current, created].sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor)));
      onDraftChange('');
      setAttachmentNote('Message scheduled');
    } catch (error) {
      setScheduleError(error instanceof Error ? error.message : 'Unable to schedule message');
    } finally {
      setScheduleBusy(false);
    }
  };

  const handleCancelScheduledMessage = async (id: string) => {
    setScheduleBusy(true);
    setScheduleError('');
    try {
      await cancelScheduledMessage(id);
      setScheduledMessages((current) => current.filter((item) => item.id !== id));
    } catch (error) {
      setScheduleError(error instanceof Error ? error.message : 'Unable to cancel scheduled message');
    } finally {
      setScheduleBusy(false);
    }
  };

  useEffect(() => {
    if (!mediaPickerOpen) return;
    let cancelled = false;
    setMediaBusy(true);
    setMediaError('');
    searchChatMedia(deferredMediaQuery, mediaKind)
      .then((items) => { if (!cancelled) setMediaItems(items); })
      .catch((error) => { if (!cancelled) setMediaError(error instanceof Error ? error.message : 'Unable to search media'); })
      .finally(() => { if (!cancelled) setMediaBusy(false); });
    return () => { cancelled = true; };
  }, [deferredMediaQuery, mediaKind, mediaPickerOpen]);

  const handleSendMedia = async (item: MediaCatalogItem) => {
    if (!conversation) return;
    setMediaBusy(true);
    setMediaError('');
    try {
      const created = await sendChatMedia(conversation.id, item.id);
      onMessageCreated?.(created);
      setMediaPickerOpen(false);
      setAttachmentNote(`${item.label} sent`);
    } catch (error) {
      setMediaError(error instanceof Error ? error.message : 'Unable to send media');
    } finally {
      setMediaBusy(false);
    }
  };

  const requestLocation = () => {
    if (!conversation) return;
    setLocationError('');
    setPendingLocation(null);
    setLocationLabel('');
    if (!navigator.geolocation) {
      setLocationError('Location access is not supported by this browser.');
      setLocationOpen(true);
      return;
    }
    setLocationBusy(true);
    navigator.geolocation.getCurrentPosition(
      (position) => {
        setPendingLocation({ latitude: position.coords.latitude, longitude: position.coords.longitude, accuracyMeters: position.coords.accuracy });
        setLocationLabel('');
        setLocationOpen(true);
        setLocationBusy(false);
      },
      (error) => {
        const message = error.code === error.PERMISSION_DENIED
          ? 'Location permission was not granted.'
          : error.code === error.TIMEOUT
            ? 'Location lookup timed out. Please try again.'
            : 'Your location could not be determined.';
        setLocationError(message);
        setLocationOpen(true);
        setLocationBusy(false);
      },
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 0 },
    );
  };

  const handleSendLocation = async () => {
    if (!conversation || !pendingLocation) return;
    setLocationBusy(true);
    setLocationError('');
    try {
      const created = await sendChatLocation(conversation.id, { ...pendingLocation, label: locationLabel.trim() || undefined });
      onMessageCreated?.(created);
      setLocationOpen(false);
      setPendingLocation(null);
      setLocationLabel('');
      setAttachmentNote('Location shared');
    } catch (error) {
      setLocationError(error instanceof Error ? error.message : 'Unable to share location');
    } finally {
      setLocationBusy(false);
    }
  };
  // A conversation is only actionable when it belongs to the active sidebar
  // view. e.g. the default #general channel must not render its composer while
  // the user is on the "Chats" (direct) or "Groups" (group) view — it should
  // still ask them to pick something first.
  const hasValidConversation = Boolean(
    conversation &&
      (viewType === 'direct' ? conversation.type === 'direct'
        : viewType === 'group' ? conversation.type === 'group'
        : conversation.type === 'channel')
  );

  const reloadMembership = async () => {
    if (!conversation || conversation.type === 'direct') {
      setMembership(null);
      return;
    }
    const result = await fetchConversationMembers(conversation.id);
    setMembership(result);
  };

  useEffect(() => {
    setMemberManagerOpen(false);
    setMemberError('');
    setSelectedMemberId('');
    setSelectedMentionIds([]);
    setMentionAll(false);
    reloadMembership().catch(() => setMembership(null));
    fetchAllUsers().then(setMemberDirectory).catch(() => setMemberDirectory([]));
    if (conversation) {
      fetchSavedMessages({ conversationId: conversation.id })
        .then((saved) => setSavedMessageIds(new Set(saved.map((message) => message.id))))
        .catch(() => setSavedMessageIds(new Set()));
    }
  }, [conversation?.id, conversation?.type]);

  useEffect(() => {
    if (!focusedMessageId || !messages.some((message) => message.id === focusedMessageId)) return;
    const frame = window.requestAnimationFrame(() => document.getElementById(`message-${focusedMessageId}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' }));
    return () => window.cancelAnimationFrame(frame);
  }, [focusedMessageId, messages]);

  const openMemberManager = async () => {
    setMenuOpen(false);
    setMemberError('');
    setMemberManagerOpen(true);
    try {
      const [nextMembership, users] = await Promise.all([
        conversation ? fetchConversationMembers(conversation.id) : Promise.reject(new Error('conversation unavailable')),
        fetchAllUsers(),
      ]);
      setMembership(nextMembership);
      setMemberDirectory(users);
    } catch (reason) {
      setMemberError(reason instanceof Error ? reason.message : 'Unable to load members');
    }
  };

  const handleAddMember = async () => {
    if (!conversation || !selectedMemberId) return;
    setMemberBusy(true);
    setMemberError('');
    try {
      await addConversationMember(conversation.id, selectedMemberId);
      await reloadMembership();
      setSelectedMemberId('');
    } catch (reason) {
      setMemberError(reason instanceof Error ? reason.message : 'Unable to add member');
    } finally {
      setMemberBusy(false);
    }
  };

  const handleRemoveMember = async (userId: string) => {
    if (!conversation) return;
    setMemberBusy(true);
    setMemberError('');
    try {
      await removeConversationMember(conversation.id, userId);
      await reloadMembership();
    } catch (reason) {
      setMemberError(reason instanceof Error ? reason.message : 'Unable to remove member');
    } finally {
      setMemberBusy(false);
    }
  };

  const runAssistant = async (mode: 'summary' | 'actions' | 'draft') => {
    if (!conversation) return;
    setAssistantLoading(true);
    try {
      const response = await requestChatAssistant(conversation.id, mode);
      setAssistantResponse(response);
      if (mode === 'draft') onDraftChange(response);
    } catch (reason) {
      setAssistantResponse(reason instanceof Error ? reason.message : 'Assistant request failed');
    } finally {
      setAssistantLoading(false);
    }
  };

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

  const activeTypingUsers = typingUsers.filter((name) => name !== currentUserName);

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
    const extension = pendingVoiceNote.mimeType.includes('ogg') ? 'ogg' : 'webm';
    const file = new File([pendingVoiceNote.blob], `voice-note-${Date.now()}.${extension}`, { type: pendingVoiceNote.mimeType });
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
      const voiceMime = MediaRecorder.isTypeSupported('audio/ogg;codecs=opus')
        ? 'audio/ogg;codecs=opus'
        : 'audio/webm;codecs=opus';
      const mediaRecorder = new MediaRecorder(stream, { mimeType: voiceMime });
      mediaRecorderRef.current = mediaRecorder;
      audioChunksRef.current = [];

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          audioChunksRef.current.push(event.data);
        }
      };

      mediaRecorder.onstop = () => {
        stream.getTracks().forEach((track) => track.stop());
        const mimeType = voiceMime.split(';')[0];
        const blob = new Blob(audioChunksRef.current, { type: mimeType });
        const url = URL.createObjectURL(blob);
        setPendingVoiceNote({ blob, url, mimeType });
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

  const canManageCompliance = currentUserRoles.some((role) => ['admin', 'superadmin', 'tenant_admin', 'platform_admin'].includes(role.toLowerCase()));

  const handleExportChat = async (format: 'json' | 'csv', includeDeleted = false) => {
    if (!conversation) return;
    setMenuOpen(false);
    try {
      const { blob, filename } = await exportConversation(conversation.id, format, includeDeleted);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = filename;
      anchor.click();
      URL.revokeObjectURL(url);
      setAttachmentNote(`Conversation exported as ${format.toUpperCase()}.`);
    } catch {
      setAttachmentNote('Could not export this conversation.');
    }
  };

  const loadCompliance = async () => {
    setComplianceBusy(true);
    setComplianceError('');
    try {
      const [policy, holds, audit] = await Promise.all([fetchRetentionPolicy(), fetchLegalHolds(), fetchComplianceAudit()]);
      setRetentionPolicy(policy);
      setRetentionDays(policy.retentionDays);
      setRetentionEnabled(policy.enabled);
      setLegalHolds(holds);
      setComplianceAudit(audit);
    } catch {
      setComplianceError('Could not load compliance controls.');
    } finally {
      setComplianceBusy(false);
    }
  };

  const openCompliance = () => {
    setMenuOpen(false);
    setComplianceOpen(true);
    void loadCompliance();
  };

  const handleSaveRetention = async () => {
    if (retentionDays < 1 || retentionDays > 3650) {
      setComplianceError('Retention must be between 1 and 3650 days.');
      return;
    }
    setComplianceBusy(true);
    setComplianceError('');
    try {
      const policy = await updateRetentionPolicy(retentionDays, retentionEnabled);
      setRetentionPolicy(policy);
      setComplianceAudit(await fetchComplianceAudit());
    } catch {
      setComplianceError('Could not save the retention policy.');
    } finally {
      setComplianceBusy(false);
    }
  };

  const handleEnforceRetention = async () => {
    if (!window.confirm('Apply the active retention policy now? Eligible messages are permanently removed unless protected by a legal hold.')) return;
    setComplianceBusy(true);
    setComplianceError('');
    try {
      const result = await enforceRetention();
      setAttachmentNote(`Retention complete: ${result.deletedMessages} message(s) removed.`);
      await loadCompliance();
    } catch {
      setComplianceError('Could not enforce the retention policy.');
      setComplianceBusy(false);
    }
  };

  const handleCreateHold = async () => {
    if (!holdName.trim() || !holdReason.trim() || (holdScope === 'conversation' && !conversation)) return;
    setComplianceBusy(true);
    setComplianceError('');
    try {
      await createLegalHold({ name: holdName.trim(), reason: holdReason.trim(), conversationId: holdScope === 'conversation' ? conversation?.id : undefined });
      setHoldName('');
      setHoldReason('');
      await loadCompliance();
    } catch {
      setComplianceError('Could not create the legal hold.');
      setComplianceBusy(false);
    }
  };

  const handleReleaseHold = async (hold: LegalHold) => {
    if (!window.confirm(`Release legal hold "${hold.name}"?`)) return;
    setComplianceBusy(true);
    setComplianceError('');
    try {
      await releaseLegalHold(hold.id);
      await loadCompliance();
    } catch {
      setComplianceError('Could not release the legal hold.');
      setComplianceBusy(false);
    }
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

    const activeMentionIds = selectedMentionIds.filter((id) => {
      const mentionedUser = memberDirectory.find((candidate) => candidate.id === id);
      return mentionedUser ? draft.includes(`@${mentionAlias(mentionedUser)}`) : false;
    });
    const mentionContext = { mentionUserIds: activeMentionIds, mentionAll: mentionAll && /@(?:all|channel)\b/i.test(draft) };

    if (replyTo) {
      const replyContext = {
        parentMessageId: replyTo.parentMessageId ?? replyTo.id,
        threadRootId: replyTo.threadRootId ?? replyTo.id,
      };
      onSend(draft, replyContext, mentionContext);
      setReplyTo(null);
    } else {
      onSend(draft, undefined, mentionContext);
    }
    setSelectedMentionIds([]);
    setMentionAll(false);
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
    const isAudio = attachment.mimeType?.startsWith('audio/') || attachment.fileName.toLowerCase().startsWith('voice-note') || /\.(mp3|wav|ogg|m4a|aac|flac|opus|webm)$/i.test(attachment.fileName);
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
            <h2 className={styles.chatName}>{hasValidConversation ? conversation?.name : 'Select a conversation'}</h2>
            <p className={`${styles.chatStatus} ${conversation?.type === 'channel' ? styles.chatStatusOffline : ''}`}>
              {!hasValidConversation ? 'Select a chat from the list to start messaging' : conversation?.type === 'direct' ? '● online' : conversation?.type === 'channel' ? `${conversation.memberIds?.length ?? 0} members` : `${conversation.memberIds?.length ?? 0} members`}
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
          {hasValidConversation && <button type="button" className={styles.savedHeaderButton} onClick={openSavedMessages} title="Saved messages">Saved</button>}
          {hasValidConversation && (
          <>
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
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={() => handleExportChat('json')}>Export JSON</button>
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={() => handleExportChat('csv')}>Export CSV</button>
                {canManageCompliance && <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={openCompliance}>Compliance and retention</button>}
                <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={openScheduleManager}>Scheduled messages</button>
                {membership?.canManage && conversation?.type !== 'direct' && (
                  <button type="button" className={styles.menuItem} style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }} onClick={openMemberManager}>Manage members</button>
                )}
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
          </>
          )}
        </div>
      </header>

      {complianceOpen && canManageCompliance && (
        <div role="presentation" onClick={() => setComplianceOpen(false)} style={{ position: 'absolute', inset: 0, zIndex: 35, display: 'grid', placeItems: 'center', padding: 16, background: 'rgba(8, 25, 38, 0.68)' }}>
          <section role="dialog" aria-modal="true" aria-label="Compliance and retention" onClick={(event) => event.stopPropagation()} style={{ width: 'min(760px, 100%)', maxHeight: '88vh', overflow: 'auto', display: 'grid', gap: 20, padding: 24, background: isDark ? '#0f3f5f' : '#fff', color: isDark ? '#e8eef4' : '#17212b', boxShadow: '0 24px 70px rgba(0,0,0,.32)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16 }}>
              <div><h3 style={{ margin: 0 }}>Compliance and retention</h3><p style={{ margin: '6px 0 0', opacity: .7, fontSize: 13 }}>Tenant-wide controls. Every change and export is audited.</p></div>
              <button type="button" onClick={() => setComplianceOpen(false)} aria-label="Close compliance controls" style={{ border: 0, background: 'transparent', color: 'inherit', fontSize: 22, cursor: 'pointer' }}>x</button>
            </div>

            <div style={{ display: 'grid', gap: 10, padding: 16, background: isDark ? '#0a324d' : '#f3f7fa' }}>
              <strong>Retention policy</strong>
              <p style={{ margin: 0, opacity: .72, fontSize: 13 }}>Disabled policies preserve all messages. Active legal holds always override deletion.</p>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center' }}>
                <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}><input type="checkbox" checked={retentionEnabled} onChange={(event) => setRetentionEnabled(event.target.checked)} /> Enabled</label>
                <label style={{ display: 'flex', gap: 8, alignItems: 'center' }}>Days <input type="number" min={1} max={3650} value={retentionDays} onChange={(event) => setRetentionDays(Number(event.target.value))} style={{ width: 100, padding: '8px 10px' }} /></label>
                <button type="button" disabled={complianceBusy} onClick={handleSaveRetention} style={{ border: 0, background: '#165c92', color: '#fff', padding: '9px 14px', cursor: 'pointer' }}>Save policy</button>
                <button type="button" disabled={complianceBusy || !retentionPolicy?.enabled} onClick={handleEnforceRetention} style={{ border: '1px solid #d92d20', background: 'transparent', color: '#d92d20', padding: '8px 13px', cursor: 'pointer' }}>Enforce now</button>
              </div>
            </div>

            <div style={{ display: 'grid', gap: 10 }}>
              <strong>Create legal hold</strong>
              <div style={{ display: 'flex', gap: 16, fontSize: 13 }}>
                <label><input type="radio" checked={holdScope === 'conversation'} onChange={() => setHoldScope('conversation')} /> Current conversation</label>
                <label><input type="radio" checked={holdScope === 'tenant'} onChange={() => setHoldScope('tenant')} /> Entire tenant</label>
              </div>
              <input value={holdName} maxLength={120} onChange={(event) => setHoldName(event.target.value)} placeholder="Hold name" style={{ padding: '10px 12px' }} />
              <textarea value={holdReason} maxLength={1000} onChange={(event) => setHoldReason(event.target.value)} placeholder="Reason and case reference" rows={3} style={{ padding: '10px 12px', resize: 'vertical' }} />
              <button type="button" disabled={complianceBusy || !holdName.trim() || !holdReason.trim()} onClick={handleCreateHold} style={{ justifySelf: 'start', border: 0, background: '#1f7a68', color: '#fff', padding: '9px 14px', cursor: 'pointer' }}>Place hold</button>
            </div>

            <div style={{ display: 'grid', gap: 8 }}>
              <strong>Legal holds</strong>
              {legalHolds.length === 0 && <span style={{ opacity: .68, fontSize: 13 }}>No legal holds have been created.</span>}
              {legalHolds.map((hold) => <div key={hold.id} style={{ display: 'flex', justifyContent: 'space-between', gap: 14, padding: 12, background: isDark ? '#0a324d' : '#f3f7fa' }}><div><b>{hold.name}</b><span style={{ display: 'block', opacity: .7, fontSize: 12 }}>{hold.conversationId ? `Conversation ${hold.conversationId}` : 'Entire tenant'} | {hold.status}</span><span style={{ display: 'block', marginTop: 3, fontSize: 13 }}>{hold.reason}</span></div>{hold.status === 'active' && <button type="button" disabled={complianceBusy} onClick={() => handleReleaseHold(hold)} style={{ alignSelf: 'center', border: '1px solid #d92d20', background: 'transparent', color: '#d92d20', padding: '7px 10px', cursor: 'pointer' }}>Release</button>}</div>)}
            </div>

            {conversation && <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}><strong style={{ width: '100%' }}>Administrator export</strong><span style={{ width: '100%', opacity: .68, fontSize: 12 }}>Includes soft-deleted records; attachment files are represented by audited metadata links.</span><button type="button" onClick={() => handleExportChat('json', true)}>Export all JSON</button><button type="button" onClick={() => handleExportChat('csv', true)}>Export all CSV</button></div>}

            <div style={{ display: 'grid', gap: 7 }}><strong>Recent audit activity</strong>{complianceAudit.slice(0, 20).map((event) => <div key={event.id} style={{ display: 'grid', gridTemplateColumns: 'minmax(130px, 1fr) 2fr auto', gap: 10, fontSize: 12, padding: '7px 0', borderBottom: '1px solid rgba(128,145,155,.25)' }}><b>{event.action}</b><span>{event.actorId}</span><time>{new Date(event.createdAt).toLocaleString()}</time></div>)}</div>
            {complianceError && <p role="alert" style={{ margin: 0, color: '#d92d20' }}>{complianceError}</p>}
            {complianceBusy && <span aria-live="polite" style={{ fontSize: 13 }}>Updating compliance state...</span>}
          </section>
        </div>
      )}

      {memberManagerOpen && membership?.canManage && (
        <div role="dialog" aria-modal="true" aria-label="Manage conversation members" style={{ position: 'absolute', inset: 0, zIndex: 30, display: 'grid', placeItems: 'center', padding: 20, background: 'rgba(8, 25, 38, 0.56)' }}>
          <div style={{ width: 'min(460px, 100%)', maxHeight: '80vh', overflow: 'auto', padding: 22, background: isDark ? '#0f3f5f' : '#fff', color: isDark ? '#e8eef4' : '#17212b', boxShadow: '0 24px 70px rgba(0,0,0,.28)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'center' }}>
              <div>
                <h3 style={{ margin: 0 }}>Conversation members</h3>
                <p style={{ margin: '5px 0 0', opacity: .68, fontSize: 13 }}>Only the owner or a tenant administrator can make changes.</p>
              </div>
              <button type="button" onClick={() => setMemberManagerOpen(false)} style={{ border: 0, background: 'transparent', color: 'inherit', fontSize: 22, cursor: 'pointer' }} aria-label="Close member manager">x</button>
            </div>

            <div style={{ display: 'grid', gap: 8, margin: '20px 0' }}>
              {membership.memberIds.map((memberId) => {
                const member = memberDirectory.find((user) => user.id === memberId);
                const isOwner = memberId === membership.ownerId;
                return (
                  <div key={memberId} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, padding: '10px 12px', background: isDark ? '#0a324d' : '#f3f7fa' }}>
                    <div>
                      <strong style={{ display: 'block', fontSize: 14 }}>{member?.name ?? memberId}</strong>
                      <span style={{ opacity: .65, fontSize: 12 }}>{isOwner ? 'Owner' : member?.email ?? memberId}</span>
                    </div>
                    {!isOwner && <button type="button" disabled={memberBusy} onClick={() => handleRemoveMember(memberId)} style={{ border: '1px solid #d92d20', background: 'transparent', color: '#d92d20', padding: '7px 10px', cursor: 'pointer' }}>Remove</button>}
                  </div>
                );
              })}
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
              <select value={selectedMemberId} onChange={(event) => setSelectedMemberId(event.target.value)} disabled={memberBusy} style={{ flex: 1, minWidth: 0, padding: '9px 10px', border: '1px solid #b8c7d1', background: isDark ? '#0a324d' : '#fff', color: 'inherit' }}>
                <option value="">Select a tenant member</option>
                {memberDirectory.filter((user) => !membership.memberIds.includes(user.id)).map((user) => <option key={user.id} value={user.id}>{user.name}</option>)}
              </select>
              <button type="button" disabled={memberBusy || !selectedMemberId} onClick={handleAddMember} style={{ border: 0, background: '#165c92', color: '#fff', padding: '9px 14px', cursor: 'pointer' }}>Add</button>
            </div>
            {memberError && <p role="alert" style={{ margin: '12px 0 0', color: '#d92d20', fontSize: 13 }}>{memberError}</p>}
          </div>
        </div>
      )}

      {forwardingMessage && (
        <div role="presentation" onClick={() => setForwardingMessage(null)} style={{ position: 'absolute', inset: 0, zIndex: 31, display: 'grid', placeItems: 'center', padding: 20, background: 'rgba(8, 25, 38, 0.62)' }}>
          <section role="dialog" aria-modal="true" aria-label="Forward message" onClick={(event) => event.stopPropagation()} style={{ width: 'min(440px, 100%)', display: 'grid', gap: 16, padding: 22, background: isDark ? '#0f3f5f' : '#fff', color: isDark ? '#e8eef4' : '#17212b', boxShadow: '0 24px 70px rgba(0,0,0,.28)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'start' }}>
              <div><h3 style={{ margin: 0 }}>Forward message</h3><p style={{ margin: '5px 0 0', opacity: .68, fontSize: 13 }}>Choose a conversation you can access.</p></div>
              <button type="button" onClick={() => setForwardingMessage(null)} aria-label="Close forward dialog" style={{ border: 0, background: 'transparent', color: 'inherit', fontSize: 22, cursor: 'pointer' }}>x</button>
            </div>
            <blockquote className={styles.forwardPreview}>{forwardingMessage.text}</blockquote>
            <select value={forwardTargetId} onChange={(event) => setForwardTargetId(event.target.value)} disabled={forwardBusy} style={{ width: '100%', padding: '10px 12px', border: '1px solid #b8c7d1', background: isDark ? '#0a324d' : '#fff', color: 'inherit' }}>
              <option value="">Select destination</option>
              {conversations.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
            </select>
            {forwardError && <p role="alert" style={{ margin: 0, color: '#d92d20', fontSize: 13 }}>{forwardError}</p>}
            <button type="button" disabled={!forwardTargetId || forwardBusy} onClick={handleForwardMessage} style={{ border: 0, background: '#1f7a68', color: '#fff', padding: '10px 14px', cursor: 'pointer', opacity: !forwardTargetId || forwardBusy ? .55 : 1 }}>
              {forwardBusy ? 'Forwarding...' : 'Forward'}
            </button>
          </section>
        </div>
      )}

      {scheduleOpen && conversation && (
        <div role="presentation" onClick={() => setScheduleOpen(false)} style={{ position: 'absolute', inset: 0, zIndex: 32, display: 'grid', placeItems: 'center', padding: 20, background: 'rgba(8, 25, 38, 0.64)' }}>
          <section role="dialog" aria-modal="true" aria-label="Scheduled messages" onClick={(event) => event.stopPropagation()} style={{ width: 'min(520px, 100%)', maxHeight: '82vh', overflow: 'auto', display: 'grid', gap: 16, padding: 22, background: isDark ? '#0f3f5f' : '#fff', color: isDark ? '#e8eef4' : '#17212b', boxShadow: '0 24px 70px rgba(0,0,0,.28)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 16, alignItems: 'start' }}>
              <div><h3 style={{ margin: 0 }}>Scheduled messages</h3><p style={{ margin: '5px 0 0', opacity: .68, fontSize: 13 }}>{conversation.name}. Times are shown in your local timezone.</p></div>
              <button type="button" onClick={() => setScheduleOpen(false)} aria-label="Close scheduled messages" style={{ border: 0, background: 'transparent', color: 'inherit', fontSize: 22, cursor: 'pointer' }}>x</button>
            </div>
            <div style={{ display: 'grid', gap: 8 }}>
              <label htmlFor="scheduled-message-time" style={{ fontSize: 13, fontWeight: 700 }}>Delivery time</label>
              <input id="scheduled-message-time" type="datetime-local" value={scheduledFor} onChange={(event) => setScheduledFor(event.target.value)} style={{ padding: '10px 12px', border: '1px solid #b8c7d1', background: isDark ? '#0a324d' : '#fff', color: 'inherit' }} />
              <div className={styles.scheduleDraftPreview}>{draft.trim() || 'Type a draft in the composer before scheduling.'}</div>
              <button type="button" disabled={scheduleBusy || !draft.trim() || !scheduledFor} onClick={handleScheduleMessage} style={{ border: 0, background: '#1f7a68', color: '#fff', padding: '10px 14px', cursor: 'pointer', opacity: scheduleBusy || !draft.trim() || !scheduledFor ? .55 : 1 }}>
                {scheduleBusy ? 'Saving...' : 'Schedule draft'}
              </button>
            </div>
            <div style={{ display: 'grid', gap: 8 }}>
              <strong style={{ fontSize: 13 }}>Pending</strong>
              {scheduledMessages.length === 0 && <span style={{ opacity: .65, fontSize: 13 }}>No pending messages in this conversation.</span>}
              {scheduledMessages.map((item) => (
                <div key={item.id} style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'center', padding: '10px 12px', background: isDark ? '#0a324d' : '#f3f7fa' }}>
                  <div style={{ minWidth: 0 }}><span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 14 }}>{item.text}</span><time style={{ opacity: .65, fontSize: 12 }}>{new Date(item.scheduledFor).toLocaleString()}</time></div>
                  <button type="button" disabled={scheduleBusy} onClick={() => handleCancelScheduledMessage(item.id)} style={{ border: '1px solid #d92d20', background: 'transparent', color: '#d92d20', padding: '7px 10px', cursor: 'pointer' }}>Cancel</button>
                </div>
              ))}
            </div>
            {scheduleError && <p role="alert" style={{ margin: 0, color: '#d92d20', fontSize: 13 }}>{scheduleError}</p>}
          </section>
        </div>
      )}

      {locationOpen && (
        <div role="presentation" onClick={() => setLocationOpen(false)} className={styles.locationBackdrop}>
          <section role="dialog" aria-modal="true" aria-label="Confirm location sharing" onClick={(event) => event.stopPropagation()} className={`${styles.locationDialog} ${isDark ? styles.locationDialogDark : ''}`}>
            <div className={styles.locationDialogHeader}>
              <div><h3>Share your location?</h3><p>Your position is sent only after you confirm below.</p></div>
              <button type="button" onClick={() => setLocationOpen(false)} aria-label="Close location dialog">x</button>
            </div>
            {pendingLocation && (
              <div className={styles.locationCoordinates}>
                <strong>{pendingLocation.latitude.toFixed(5)}, {pendingLocation.longitude.toFixed(5)}</strong>
                <span>Estimated accuracy: {Math.round(pendingLocation.accuracyMeters)} m</span>
              </div>
            )}
            <label className={styles.locationLabel}>
              Optional label
              <input value={locationLabel} onChange={(event) => setLocationLabel(event.target.value)} maxLength={120} placeholder="e.g. Nairobi office" disabled={!pendingLocation || locationBusy} />
            </label>
            <p className={styles.locationPrivacy}>The map provider is not contacted while viewing this message. A map opens only when someone selects its link.</p>
            {locationError && <p role="alert" className={styles.mediaError}>{locationError}</p>}
            <div className={styles.locationActions}>
              <button type="button" onClick={() => setLocationOpen(false)}>Cancel</button>
              <button type="button" disabled={!pendingLocation || locationBusy} onClick={handleSendLocation}>{locationBusy ? 'Sharing...' : 'Confirm and share'}</button>
            </div>
          </section>
        </div>
      )}

      {savedPanelOpen && (
        <div role="presentation" className={styles.savedBackdrop} onClick={() => setSavedPanelOpen(false)}>
          <section role="dialog" aria-modal="true" aria-label="Saved messages" className={`${styles.savedPanel} ${isDark ? styles.savedPanelDark : ''}`} onClick={(event) => event.stopPropagation()}>
            <div className={styles.savedPanelHeader}>
              <div><h3>Saved messages</h3><p>Private to your account and limited to conversations you can still access.</p></div>
              <button type="button" onClick={() => setSavedPanelOpen(false)} aria-label="Close saved messages">x</button>
            </div>
            {savedError && <p role="alert" className={styles.mediaError}>{savedError}</p>}
            <div className={styles.savedMessageList} aria-busy={savedBusy}>
              {savedMessages.map((message) => (
                <article key={message.id} className={styles.savedMessageCard}>
                  <button type="button" onClick={() => { onOpenMessage?.(message.conversationId, message.id); setSavedPanelOpen(false); }}>
                    <strong>{message.sender}</strong><span>{message.text}</span><time>{new Date(message.savedAt ?? message.createdAt).toLocaleString()}</time>
                  </button>
                  <button type="button" disabled={savedBusy} onClick={() => toggleSavedMessage(message)} aria-label="Remove saved message">Remove</button>
                </article>
              ))}
              {!savedBusy && savedMessages.length === 0 && <p className={styles.savedEmpty}>No saved messages yet. Use a message's action menu to save it.</p>}
            </div>
          </section>
        </div>
      )}

      <div
        ref={streamRef}
        className={`${styles.messageStream} ${isDark ? styles.messageStreamDark : ''}`}
      >
        <div className={styles.systemBanner}>
          <div className={`${styles.systemBannerInner} ${isDark ? styles.systemBannerInnerDark : ''}`}>
            Messages are access-controlled within your StatGate tenant
          </div>
        </div>

        {visibleMessages.length === 0 ? (
          <div className={styles.emptyState}>
            {hasValidConversation
              ? 'No messages yet. Start the conversation!'
              : 'Select a conversation from the list to start messaging.'}
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
                const isOwnMessage = Boolean(currentUserId && message.senderId === currentUserId);
                const prevMessage = msgIndex > 0 ? group.messages[msgIndex - 1] : null;
                const showSender = !isOwnMessage && (!prevMessage || prevMessage.sender !== message.sender);
                const msgReactions = reactions[message.id] || [];

                return (
                  <div
                    id={`message-${message.id}`}
                    key={message.id}
                    className={`${styles.messageRow} ${isOwnMessage ? styles.messageRowSent : styles.messageRowReceived} ${focusedMessageId === message.id ? styles.focusedMessage : ''}`}
                  >
                    <div
                    className={`${styles.messageBubble} ${isOwnMessage ? (isDark ? styles.sentDark : styles.sent) : (isDark ? styles.receivedDark : styles.received)} ${(message.mentionAll || message.mentionUserIds?.includes(currentUserId ?? '')) ? styles.mentionedBubble : ''}`}
                      onClick={() => setActiveQuickReactions(activeQuickReactions === message.id ? null : message.id)}
                      onDoubleClick={() => toggleReaction(message.id, '❤️')}
                    >
                      {showSender && <div className={styles.senderName}>{message.sender}</div>}
                      {message.forwardedFromMessageId && (
                        <div className={styles.forwardedLabel}>Forwarded from {message.forwardedFromSender || 'another conversation'}</div>
                      )}
                      <div className={styles.messageText}>{renderMessageText(message)}</div>
                      {message.location && (
                        <div className={styles.locationCard} onClick={(event) => event.stopPropagation()}>
                          <div className={styles.locationPin} aria-hidden="true"><span /></div>
                          <div className={styles.locationDetails}>
                            <strong>{message.location.label || 'Shared location'}</strong>
                            <span>{message.location.latitude.toFixed(5)}, {message.location.longitude.toFixed(5)}</span>
                            {message.location.accuracyMeters ? <small>Accuracy about {Math.round(message.location.accuracyMeters)} m</small> : null}
                          </div>
                          <a href={`https://www.openstreetmap.org/?mlat=${encodeURIComponent(message.location.latitude)}&mlon=${encodeURIComponent(message.location.longitude)}#map=16/${encodeURIComponent(message.location.latitude)}/${encodeURIComponent(message.location.longitude)}`} target="_blank" rel="noreferrer">Open map</a>
                        </div>
                      )}
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
                            <button
                              type="button"
                              className={styles.messageActionItem}
                              onClick={(e) => {
                                e.stopPropagation();
                                setForwardingMessage(message);
                                setForwardTargetId('');
                                setForwardError('');
                                setActiveMessageActions(null);
                              }}
                            >
                              Forward
                            </button>
                            <button type="button" className={styles.messageActionItem} disabled={savedBusy} onClick={(event) => { event.stopPropagation(); void toggleSavedMessage(message); }}>
                              {savedMessageIds.has(message.id) ? 'Remove saved' : 'Save message'}
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
        {activeTypingUsers.length > 0 && (
          <div className={styles.messageRowReceived}>
            <div className={`${styles.typingIndicator} ${isDark ? styles.typingIndicatorDark : ''}`}>
              <div className={styles.typingDots}>
                <span className={styles.typingDot} />
                <span className={styles.typingDot} />
                <span className={styles.typingDot} />
              </div>
              <span className={styles.typingText}>
                {activeTypingUsers.length === 1 ? `${activeTypingUsers[0]} is typing...` : 'Several people are typing...'}
              </span>
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

      {assistantOpen && (
        <section className={styles.assistantPanel} style={{ background: isDark ? '#0f3f5f' : '#f4f8fb' }} aria-label="Conversation assistant">
          <div className={styles.assistantHeader}>
            <strong>Conversation assistant</strong>
            <button type="button" onClick={() => setAssistantOpen(false)} aria-label="Close assistant">Close</button>
          </div>
          <p className={styles.assistantNote}>Uses only messages you are permitted to read. Drafts are inserted for review and never sent automatically.</p>
          <div className={styles.assistantActions}>
            <button type="button" disabled={assistantLoading} onClick={() => runAssistant('summary')}>Summarize</button>
            <button type="button" disabled={assistantLoading} onClick={() => runAssistant('actions')}>Find actions</button>
            <button type="button" disabled={assistantLoading} onClick={() => runAssistant('draft')}>Draft reply</button>
          </div>
          {assistantResponse && <pre className={styles.assistantResponse}>{assistantResponse}</pre>}
        </section>
      )}

      {hasValidConversation && (
      <footer className={styles.inputFooter} style={{ background: isDark ? '#0a2b45' : '#ffffff' }}>
        <input ref={fileInputRef} type="file" className={styles.hiddenInput} accept="audio/*,video/*,image/*,.pdf,.doc,.docx,.txt" onChange={handleAttachment} />
        <button type="button" className={styles.attachButton} onClick={() => fileInputRef.current?.click()} title="Attach">
          ＋
        </button>
        <button type="button" className={styles.assistantButton} onClick={() => setAssistantOpen((value) => !value)} title="Conversation assistant" aria-pressed={assistantOpen}>AI</button>
        <button type="button" className={styles.mediaButton} onClick={() => setMediaPickerOpen((value) => !value)} title="GIFs and stickers" aria-pressed={mediaPickerOpen}>GIF</button>
        <button type="button" className={styles.locationButton} onClick={requestLocation} title="Share location" disabled={locationBusy} aria-label="Share current location">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21s7-6.1 7-12A7 7 0 1 0 5 9c0 5.9 7 12 7 12Zm0-9.5A2.5 2.5 0 1 1 12 6a2.5 2.5 0 0 1 0 5.5Z" fill="currentColor" /></svg>
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
            onChange={(event) => handleComposerChange(event.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            style={{ color: isDark ? '#e8eef4' : '#1a1a1a' }}
          />
        </div>

        {mentionMatch && (mentionCandidates.length > 0 || (membership?.canManage && /^(all|channel)?$/i.test(mentionQuery))) && (
          <div className={`${styles.mentionPicker} ${isDark ? styles.mentionPickerDark : ''}`} role="listbox" aria-label="Mention a conversation member">
            {membership?.canManage && /^(all|channel)?$/i.test(mentionQuery) && (
              <button type="button" onClick={() => selectMention(undefined, true)} role="option">
                <strong>@all</strong><span>Notify everyone in this conversation</span>
              </button>
            )}
            {mentionCandidates.map((candidate) => (
              <button key={candidate.id} type="button" onClick={() => selectMention(candidate)} role="option">
                <strong>{candidate.name}</strong><span>@{mentionAlias(candidate)}</span>
              </button>
            ))}
          </div>
        )}

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

        {mediaPickerOpen && (
          <section className={`${styles.mediaPicker} ${isDark ? styles.mediaPickerDark : ''}`} role="dialog" aria-label="GIF and sticker picker">
            <div className={styles.mediaPickerHeader}>
              <div className={styles.mediaTabs}>
                <button type="button" className={mediaKind === 'gif' ? styles.mediaTabActive : ''} onClick={() => setMediaKind('gif')}>Animated</button>
                <button type="button" className={mediaKind === 'sticker' ? styles.mediaTabActive : ''} onClick={() => setMediaKind('sticker')}>Stickers</button>
              </div>
              <button type="button" onClick={() => setMediaPickerOpen(false)} aria-label="Close media picker">x</button>
            </div>
            <input type="search" value={mediaQuery} onChange={(event) => setMediaQuery(event.target.value)} placeholder={`Search ${mediaKind === 'gif' ? 'animated reactions' : 'stickers'}`} maxLength={60} />
            {mediaError && <p role="alert" className={styles.mediaError}>{mediaError}</p>}
            <div className={styles.mediaGrid} aria-busy={mediaBusy}>
              {mediaItems.map((item) => (
                <button key={item.id} type="button" disabled={mediaBusy} onClick={() => handleSendMedia(item)} title={`Send ${item.label}`}>
                  <img src={item.previewUrl} alt="" />
                  <span>{item.label}</span>
                </button>
              ))}
              {!mediaBusy && mediaItems.length === 0 && <p>No matching media.</p>}
            </div>
          </section>
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
            {recording ? `■ ${recordingSeconds}s` : (
              <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                <rect x="8" y="3" width="8" height="12" rx="4" fill="currentColor" />
                <path d="M5.5 11.5a6.5 6.5 0 0 0 13 0M12 18v3M8.5 21h7" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
              </svg>
            )}
          </button>
        )}
      </footer>
      )}

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
          screenStream={screenStream}
          remoteStreams={remoteStreams}
          remoteScreenSharers={remoteScreenSharers}
          micMuted={micMuted}
          cameraOff={cameraOff}
          isScreenSharing={isScreenSharing}
          screenShareSupported={screenShareSupported}
          connecting={connecting}
          callError={callError}
          connectionState={connectionState}
          quality={quality}
          qualityMetrics={qualityMetrics}
          currentUserId={currentUserId}
          currentUserName={currentUserName}
          onToggleMute={toggleMute}
          onToggleCamera={toggleCamera}
          onToggleScreenShare={toggleScreenShare}
          onRemoveParticipant={handleRemoveCallParticipant}
          onRequestMute={requestMute}
          onChangeParticipantRole={handleChangeCallParticipantRole}
          pendingMuteRequest={pendingMuteRequest}
          onRespondToMuteRequest={respondToMuteRequest}
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
