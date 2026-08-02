export interface Channel {
  id: string;
  name: string;
}

export interface Message {
  id: string;
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
  notifSound: boolean;
  notifPreview: boolean;
  downloadImages: string;
  downloadVideos: string;
  downloadDocuments: string;
}

export interface Conversation {
  id: string;
  name: string;
  type: 'channel' | 'direct' | 'group';
  memberIds?: string[];
  category?: string;
  latestPreview?: string;
  latestMessageAt?: string;
  attachmentCount?: number;
}
