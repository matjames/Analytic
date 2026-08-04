import { Conversation, Message, User, UserSettings, PinnedMessage, Task, Notification, Presence, MessageReaction } from '../types';

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api';
export const WS_URL = import.meta.env.VITE_WS_URL ?? (() => {
  if (typeof window === 'undefined') {
    return 'ws://localhost:4000/ws';
  }
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const host = window.location.host || 'localhost:3009';
  return `${protocol}://${host}/ws`;
})();

export async function fetchCurrentUser(): Promise<User> {
  const response = await fetch(`${BASE_URL}/users/me`);
  if (!response.ok) {
    throw new Error(`Failed to fetch current user: ${response.status}`);
  }
  return response.json();
}

export async function fetchConversations(): Promise<Conversation[]> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/conversations`);
  if (!response.ok) {
    throw new Error(`Failed to fetch conversations: ${response.status}`);
  }
  return response.json();
}

export async function fetchMessages(conversationId: string): Promise<Message[]> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages`);
  if (!response.ok) {
    throw new Error(`Failed to fetch messages: ${response.status}`);
  }
  return response.json();
}

export async function sendChatMessage(payload: { conversationId: string; channelId?: string; sender: string; text: string; tenantId?: string }): Promise<Message> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ event: 'message.send', tenantId: payload.tenantId, payload }),
  });
  if (!response.ok) {
    throw new Error(`Failed to send message: ${response.status}`);
  }
  return response.json();
}

export async function uploadChatAttachment(formData: FormData): Promise<Message> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/attachments`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    throw new Error(`Failed to upload attachment: ${response.status}`);
  }
  return response.json();
}

export async function fetchAllUsers(): Promise<User[]> {
  const response = await fetch(`${BASE_URL}/users`);
  if (!response.ok) {
    throw new Error(`Failed to fetch users: ${response.status}`);
  }
  return response.json();
}

export async function createDM(targetUserId: string, targetName: string): Promise<Conversation> {
  const response = await fetch(`${BASE_URL}/conversations/dm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetUserId, targetName }),
  });
  if (!response.ok) {
    throw new Error(`Failed to create DM: ${response.status}`);
  }
  return response.json();
}

export interface GroupTemplate {
  id: string;
  name: string;
  description: string;
  icon?: string;
}

export interface GroupCategory {
  id: string;
  name: string;
  icon: string;
  description: string;
  groups: GroupTemplate[];
}

export async function fetchGroupTemplates(): Promise<GroupCategory[]> {
  const response = await fetch(`${BASE_URL}/groups/templates`);
  if (!response.ok) {
    throw new Error(`Failed to fetch group templates: ${response.status}`);
  }
  return response.json();
}

export async function createGroup(groupId: string, name: string, memberIds: string[]): Promise<Conversation> {
  const response = await fetch(`${BASE_URL}/conversations/group`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ groupId, name, memberIds }),
  });
  if (!response.ok) {
    throw new Error(`Failed to create group: ${response.status}`);
  }
  return response.json();
}

export async function updateProfile(name: string, about: string, avatarUrl: string): Promise<User> {
  const response = await fetch(`${BASE_URL}/users/me/profile`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, about, avatarUrl }),
  });
  if (!response.ok) {
    throw new Error(`Failed to update profile: ${response.status}`);
  }
  return response.json();
}

export async function fetchUserSettings(): Promise<UserSettings> {
  const response = await fetch(`${BASE_URL}/users/me/settings`);
  if (!response.ok) {
    throw new Error(`Failed to fetch settings: ${response.status}`);
  }
  return response.json();
}

export async function updateUserSettings(settings: UserSettings): Promise<UserSettings> {
  const response = await fetch(`${BASE_URL}/users/me/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  });
  if (!response.ok) {
    throw new Error(`Failed to save settings: ${response.status}`);
  }
  return response.json();
}

export interface Post {
  id: string;
  author: string;
  role: string;
  org: string;
  time: string;
  text: string;
  likes: number;
  comments: number;
  shares: number;
  createdAt?: string;
  likedByMe?: boolean;
  commentList?: PostComment[];
}

export interface Connection {
  id: string;
  userId: string;
  connectedToId: string;
  connectedName: string;
  connectedRole: string;
  connectedOrg: string;
  connectedAt?: string;
}

export interface Opportunity {
  id: string;
  badge: string;
  badgeColor: string;
  title: string;
  description: string;
}

export interface Job {
  id: string;
  title: string;
  company: string;
  location: string;
  type: string;
  salary: string;
  icon: string;
}

export interface Meeting {
  id: string;
  title: string;
  date: string;
  time: string;
  duration: string;
  participants: number;
  status: string;
  room: string;
  host: string;
  createdAt?: string;
}

export interface MeetingRoom {
  id: string;
  name: string;
  capacity: number;
  status: string;
  password: string;
  url: string;
}

export interface MeetingRecording {
  id: string;
  title: string;
  date: string;
  duration: string;
  size: string;
  url?: string;
  createdAt?: string;
}

export async function fetchPosts(): Promise<Post[]> {
  const response = await fetch(`${BASE_URL}/collaboration/posts`);
  if (!response.ok) throw new Error(`Failed to fetch posts: ${response.status}`);
  return response.json();
}

export async function createPost(post: Omit<Post, 'id' | 'createdAt'>): Promise<Post> {
  const response = await fetch(`${BASE_URL}/collaboration/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(post),
  });
  if (!response.ok) throw new Error(`Failed to create post: ${response.status}`);
  return response.json();
}

export interface PostComment {
  id: string;
  postId: string;
  author: string;
  role?: string;
  org?: string;
  text: string;
  createdAt: string;
}

export async function togglePostLike(postId: string, userId?: string): Promise<{ liked: boolean }> {
  const response = await fetch(`${BASE_URL}/collaboration/posts/${postId}/like`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to toggle post like: ${response.status}`);
  return response.json();
}

export async function fetchPostComments(postId: string): Promise<PostComment[]> {
  const response = await fetch(`${BASE_URL}/collaboration/posts/${postId}/comments`);
  if (!response.ok) throw new Error(`Failed to fetch comments: ${response.status}`);
  return response.json();
}

export async function addPostComment(postId: string, comment: { author: string; role?: string; org?: string; text: string }): Promise<PostComment> {
  const response = await fetch(`${BASE_URL}/collaboration/posts/${postId}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(comment),
  });
  if (!response.ok) throw new Error(`Failed to add comment: ${response.status}`);
  return response.json();
}

export async function fetchConnections(): Promise<Connection[]> {
  const response = await fetch(`${BASE_URL}/collaboration/connections`);
  if (!response.ok) throw new Error(`Failed to fetch connections: ${response.status}`);
  return response.json();
}

export async function createConnection(targetUserId: string): Promise<Connection> {
  const response = await fetch(`${BASE_URL}/collaboration/connections`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetUserId }),
  });
  if (!response.ok) throw new Error(`Failed to create connection: ${response.status}`);
  return response.json();
}

export async function removeConnection(targetUserId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/collaboration/connections`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetUserId }),
  });
  if (!response.ok) throw new Error(`Failed to remove connection: ${response.status}`);
}

export async function fetchOpportunities(): Promise<Opportunity[]> {
  const response = await fetch(`${BASE_URL}/collaboration/opportunities`);
  if (!response.ok) throw new Error(`Failed to fetch opportunities: ${response.status}`);
  return response.json();
}

export async function fetchJobs(): Promise<Job[]> {
  const response = await fetch(`${BASE_URL}/collaboration/jobs`);
  if (!response.ok) throw new Error(`Failed to fetch jobs: ${response.status}`);
  return response.json();
}

export async function fetchMeetings(): Promise<Meeting[]> {
  const response = await fetch(`${BASE_URL}/meetings`);
  if (!response.ok) throw new Error(`Failed to fetch meetings: ${response.status}`);
  return response.json();
}

export async function createMeeting(meeting: Omit<Meeting, 'id' | 'createdAt'>): Promise<Meeting> {
  const response = await fetch(`${BASE_URL}/meetings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(meeting),
  });
  if (!response.ok) throw new Error(`Failed to create meeting: ${response.status}`);
  return response.json();
}

export async function fetchMeetingRooms(): Promise<MeetingRoom[]> {
  const response = await fetch(`${BASE_URL}/meetings/rooms`);
  if (!response.ok) throw new Error(`Failed to fetch meeting rooms: ${response.status}`);
  return response.json();
}

export async function fetchMeetingRecordings(): Promise<MeetingRecording[]> {
  const response = await fetch(`${BASE_URL}/meetings/recordings`);
  if (!response.ok) throw new Error(`Failed to fetch recordings: ${response.status}`);
  return response.json();
}

export interface WellnessPost {
  id: string;
  author: string;
  handle: string;
  avatar: string;
  category: string;
  time: string;
  text: string;
  likes: number;
  comments: number;
  shares: number;
  bookmarks: number;
  tags: string[];
  createdAt?: string;
}

export async function fetchWellnessPosts(): Promise<WellnessPost[]> {
  const response = await fetch(`${BASE_URL}/wellness/posts`);
  if (!response.ok) throw new Error(`Failed to fetch wellness posts: ${response.status}`);
  return response.json();
}

export async function createWellnessPost(post: Omit<WellnessPost, 'id' | 'createdAt'>): Promise<WellnessPost> {
  const response = await fetch(`${BASE_URL}/wellness/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(post),
  });
  if (!response.ok) throw new Error(`Failed to create wellness post: ${response.status}`);
  return response.json();
}

export interface KnowledgeExpert {
  id: string;
  name: string;
  role: string;
  org: string;
  specialties: string[];
  followers: number;
  articles: number;
  rating: number;
  avatar: string;
}

export interface KnowledgeArticle {
  id: string;
  title: string;
  author: string;
  category: string;
  readTime: string;
  excerpt: string;
  likes: number;
  views: number;
  published: string;
}

export interface KnowledgeIdea {
  id: string;
  title: string;
  author: string;
  category: string;
  description: string;
  votes: number;
  status: string;
}

export interface KnowledgePost {
  id: string;
  title: string;
  author: string;
  category: string;
  content: string;
  createdBy: string;
  createdAt: string;
}

export async function fetchKnowledgeExperts(): Promise<KnowledgeExpert[]> {
  const response = await fetch(`${BASE_URL}/knowledge/experts`);
  if (!response.ok) throw new Error(`Failed to fetch experts: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgeArticles(): Promise<KnowledgeArticle[]> {
  const response = await fetch(`${BASE_URL}/knowledge/articles`);
  if (!response.ok) throw new Error(`Failed to fetch articles: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgeIdeas(): Promise<KnowledgeIdea[]> {
  const response = await fetch(`${BASE_URL}/knowledge/ideas`);
  if (!response.ok) throw new Error(`Failed to fetch ideas: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgePosts(): Promise<KnowledgePost[]> {
  const response = await fetch(`${BASE_URL}/knowledge/posts`);
  if (!response.ok) throw new Error(`Failed to fetch posts: ${response.status}`);
  return response.json();
}

export async function createKnowledgePost(post: Omit<KnowledgePost, 'id' | 'createdAt'>): Promise<KnowledgePost> {
  const response = await fetch(`${BASE_URL}/knowledge/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(post),
  });
  if (!response.ok) throw new Error(`Failed to create post: ${response.status}`);
  return response.json();
}

// ── Message Edit / Delete ──

export async function editChatMessage(messageId: string, text: string): Promise<Message> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages/${messageId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!response.ok) throw new Error(`Failed to edit message: ${response.status}`);
  return response.json();
}

export async function deleteChatMessage(messageId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages/${messageId}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(`Failed to delete message: ${response.status}`);
}

// ── Message Reactions ──

export async function addReaction(messageId: string, emoji: string, userId?: string): Promise<MessageReaction> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages/${messageId}/reactions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ emoji, userId }),
  });
  if (!response.ok) throw new Error(`Failed to add reaction: ${response.status}`);
  return response.json();
}

export async function removeReaction(messageId: string, emoji: string, userId?: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages/${messageId}/reactions`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ emoji, userId }),
  });
  if (!response.ok) throw new Error(`Failed to remove reaction: ${response.status}`);
}

// ── Read Receipts ──

export async function markMessageRead(messageId: string, userId?: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/messages/${messageId}/read`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to mark message as read: ${response.status}`);
}

// ── Pinned Messages ──

export async function fetchPinnedMessages(conversationId: string): Promise<PinnedMessage[]> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/conversations/${conversationId}/pinned`);
  if (!response.ok) throw new Error(`Failed to fetch pinned messages: ${response.status}`);
  return response.json();
}

export async function pinMessage(conversationId: string, messageId: string, pinnedBy?: string): Promise<PinnedMessage> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/conversations/${conversationId}/pinned`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageId, pinnedBy }),
  });
  if (!response.ok) throw new Error(`Failed to pin message: ${response.status}`);
  return response.json();
}

export async function unpinMessage(conversationId: string, messageId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/chat/conversations/${conversationId}/pinned`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageId }),
  });
  if (!response.ok) throw new Error(`Failed to unpin message: ${response.status}`);
}

// ── Tasks ──

export async function fetchTasks(conversationId?: string): Promise<Task[]> {
  const query = conversationId ? `?conversationId=${encodeURIComponent(conversationId)}` : '';
  const response = await fetch(`${BASE_URL}/api/v1/tasks${query}`);
  if (!response.ok) throw new Error(`Failed to fetch tasks: ${response.status}`);
  return response.json();
}

export async function createTask(task: Omit<Task, 'id' | 'createdAt' | 'updatedAt'>): Promise<Task> {
  const response = await fetch(`${BASE_URL}/api/v1/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task),
  });
  if (!response.ok) throw new Error(`Failed to create task: ${response.status}`);
  return response.json();
}

export async function updateTaskStatus(taskId: string, status: string): Promise<Task> {
  const response = await fetch(`${BASE_URL}/api/v1/tasks/${taskId}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
  if (!response.ok) throw new Error(`Failed to update task status: ${response.status}`);
  return response.json();
}

// ── Notifications ──

export async function fetchNotifications(): Promise<Notification[]> {
  const response = await fetch(`${BASE_URL}/api/v1/notifications`);
  if (!response.ok) throw new Error(`Failed to fetch notifications: ${response.status}`);
  return response.json();
}

export async function markNotificationRead(notificationId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/notifications/${notificationId}/read`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to mark notification as read: ${response.status}`);
}

export async function markAllNotificationsRead(): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/v1/notifications/read-all`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to mark all notifications as read: ${response.status}`);
}

// ── Presence ──

export async function fetchPresence(userId?: string): Promise<Presence[] | Presence> {
  const query = userId ? `?userId=${encodeURIComponent(userId)}` : '';
  const response = await fetch(`${BASE_URL}/api/v1/presence${query}`);
  if (!response.ok) throw new Error(`Failed to fetch presence: ${response.status}`);
  return response.json();
}

export async function updatePresence(status: string, userId?: string): Promise<Presence> {
  const response = await fetch(`${BASE_URL}/api/v1/presence`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status, userId }),
  });
  if (!response.ok) throw new Error(`Failed to update presence: ${response.status}`);
  return response.json();
}
