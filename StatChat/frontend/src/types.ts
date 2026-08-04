export interface Channel {
  id: string;
  name: string;
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
}

export interface Conversation {
  id: string;
  tenantId?: string;
  name: string;
  type: 'channel' | 'direct' | 'group';
  memberIds?: string[];
  category?: string;
  latestPreview?: string;
  latestMessageAt?: string;
  attachmentCount?: number;
  unreadCount?: number;
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