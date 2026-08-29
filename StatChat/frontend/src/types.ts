export interface Channel {
  id: string;
  tenantId?: string;
  name: string;
  description?: string;
  visibility: 'public' | 'private';
  createdBy?: string;
  createdAt?: string;
  memberCount: number;
  joined: boolean;
  archived?: boolean;
}

export interface PollOption {
  id: string;
  label: string;
  votes: number;
}

export interface Poll {
  id: string;
  tenantId?: string;
  question: string;
  options: PollOption[];
  createdBy: string;
  createdAt: string;
  voted?: boolean;
}

export interface MessageAttachment {
  id: string;
  messageId?: string;
  fileName: string;
  fileType: string;
  url: string;
  mimeType?: string;
  createdAt?: string;
}

export interface MessageReaction {
  id: string;
  messageId: string;
  userId: string;
  userName?: string;
  emoji: string;
  createdAt?: string;
}

export interface Message {
  id: string;
  tenantId?: string;
  conversationId: string;
  channelId?: string;
  sender: string;
  senderId?: string;
  text: string;
  createdAt: string;
  updatedAt?: string;
  deletedAt?: string;
  parentMessageId?: string;
  threadRootId?: string;
  status?: string;
  deliveryStatus?: string;
  attachments?: MessageAttachment[];
  reactions?: MessageReaction[];
  pinned?: boolean;
  readBy?: string[];
  forwardedFromMessageId?: string;
  forwardedFromSender?: string;
  location?: MessageLocation;
  mentionUserIds?: string[];
  mentionAll?: boolean;
  savedAt?: string;
}

export interface MessageLocation {
  messageId?: string;
  latitude: number;
  longitude: number;
  accuracyMeters?: number;
  label?: string;
  createdAt: string;
}

export interface ScheduledMessage {
  id: string;
  tenantId?: string;
  conversationId: string;
  senderId: string;
  sender: string;
  text: string;
  scheduledFor: string;
  status: 'pending' | 'sent' | 'cancelled';
  messageId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface MediaCatalogItem {
  id: string;
  kind: 'gif' | 'sticker';
  label: string;
  tags: string[];
  previewUrl: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  organizationId: string;
  roles: string[];
  avatarUrl?: string;
  about?: string;
  presence?: string;
}

export interface UserSettings {
  userId: string;
  theme: string;
  accentColor: string;
  fontSize: string;
  enterToSend: boolean;
  language: string;
  lastSeen: string;
  profilePhoto: string;
  readReceipts: boolean;
  typingIndicator: boolean;
  voiceNotes: boolean;
  readByDefault: boolean;
  autoDownload: string;
  notifMessages: boolean;
  notifGroups: boolean;
  notifMentions: boolean;
  notifMeetings: boolean;
  notifCollaboration: boolean;
  notifFiles: boolean;
  notifKnowledge: boolean;
  notifWellness: boolean;
  notifSound: boolean;
  notifPreview: boolean;
  crossServiceAlerts: boolean;
  downloadImages: string;
  downloadVideos: string;
  downloadDocuments: string;
  wallpaper?: string;
}

export interface Conversation {
  id: string;
  tenantId?: string;
  objectRef?: string;
  name: string;
  type: 'channel' | 'direct' | 'group';
  memberIds?: string[];
  category?: string;
  latestPreview?: string;
  latestMessageAt?: string;
  attachmentCount?: number;
  unreadCount?: number;
  favourite?: boolean;
  muted?: boolean;
  metadata?: Record<string, unknown>;
}

export interface Favourite {
  conversationId: string;
  favourite: boolean;
}

export interface SearchResult {
  users: User[];
  conversations: Conversation[];
  messages: Message[];
  channels: Channel[];
}

export interface GatewayEnvelope {
  event: string;
  tenantId?: string;
  payload: any;
}

export interface PinnedMessage {
  id: string;
  conversationId: string;
  messageId: string;
  pinnedBy: string;
  pinnedAt: string;
}

export interface Task {
  id: string;
  title: string;
  description?: string;
  assignee?: string;
  priority?: string;
  dueDate?: string;
  status: string;
  conversationId?: string;
  createdBy: string;
  createdAt: string;
  updatedAt?: string;
}

export interface MessageSearchFilters {
  conversationId?: string;
  sender?: string;
  from?: string;
  to?: string;
  hasAttachment?: boolean;
  savedOnly?: boolean;
}

export interface CalendarEvent {
  id: string;
  tenantId?: string;
  title: string;
  description?: string;
  location?: string;
  startAt: string;
  endAt: string;
  allDay: boolean;
  conversationId?: string;
  attendeeIds: string[];
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface Notification {
  id: string;
  userId: string;
  type: string;
  title: string;
  body: string;
  link?: string;
  read: boolean;
  createdAt: string;
}

export interface Presence {
  userId: string;
  status: string;
  updatedAt: string;
}

export interface RetentionPolicy {
  tenantId: string;
  retentionDays: number;
  enabled: boolean;
  updatedBy?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface LegalHold {
  id: string;
  tenantId: string;
  conversationId?: string;
  name: string;
  reason: string;
  status: 'active' | 'released';
  createdBy: string;
  createdAt: string;
  releasedBy?: string;
  releasedAt?: string;
}

export interface ComplianceAuditEvent {
  id: string;
  tenantId: string;
  actorId: string;
  action: string;
  targetType: string;
  targetId: string;
  details?: Record<string, unknown>;
  createdAt: string;
}

// ── Conferencing (Native WebRTC) ──

export type CallKind = 'voice' | 'video';
export type CallStatus = 'scheduled' | 'live' | 'ended';

export interface CallSession {
  id: string;
  tenantId?: string;
  roomId: string;
  roomName: string;
  kind: CallKind;
  hostId: string;
  hostName: string;
  status: CallStatus;
  conversationId?: string;
  createdAt: string;
  endedAt?: string;
}

export interface CallParticipant {
  id: string;
  sessionId: string;
  userId: string;
  userName: string;
  role: string;
  joinedAt: string;
  leftAt?: string;
}

export interface CallRecording {
  id: string;
  sessionId: string;
  title: string;
  fileName: string;
  url: string;
  size: number;
  duration?: string;
  createdAt: string;
}

export type CallQuality = 'excellent' | 'good' | 'fair' | 'poor' | 'offline' | 'unknown';
export type CallConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'offline';

export interface CallQualitySample {
  id: string;
  sessionId: string;
  userId: string;
  rttMs: number;
  jitterMs: number;
  packetLossPct: number;
  bitrateKbps: number;
  quality: Exclude<CallQuality, 'offline' | 'unknown'>;
  createdAt: string;
}

export interface CallSignal {
  type: string; // offer | answer | ice-candidate | screen-share-* | mute-requested | mute-accepted | mute-declined
  sessionId: string;
  from: string;
  fromName?: string;
  to?: string;
  payload?: string;
}
