import { Conversation, Message, User, UserSettings, PinnedMessage, Task, Notification, Presence, MessageReaction, CallSession, CallParticipant, CallRecording, CallQualitySample, SearchResult, Favourite, Channel, CalendarEvent, Poll, ScheduledMessage, MediaCatalogItem, MessageSearchFilters, RetentionPolicy, LegalHold, ComplianceAuditEvent } from '../types';

const BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '');
const REGISTRY_API = (import.meta.env.VITE_REGISTRY_API_URL ?? 'http://localhost:9090/api').replace(/\/$/, '');

export interface OrganisationBranding {
  display_name: string;
  logo_url: string;
  primary_color: string;
  secondary_color: string;
}

function getStoredToken(): string {
  if (typeof window === 'undefined') {
    return '';
  }
  return window.localStorage.getItem('statchat_token')?.trim() ?? '';
}

async function apiFetch(input: RequestInfo | URL, init: RequestInit = {}) {
  const headers = new Headers(init.headers ?? {});
  const token = getStoredToken();
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  const workspaceID = typeof window !== 'undefined' ? window.localStorage.getItem('workspace_id')?.trim() ?? '' : '';
  if (workspaceID) {
    headers.set('X-Workspace-ID', workspaceID);
  }
  if (!headers.has('Content-Type') && init.body && typeof init.body === 'string') {
    headers.set('Content-Type', 'application/json');
  }
  return fetch(input, { ...init, headers });
}

export const WS_URL = import.meta.env.VITE_WS_URL ?? (() => {
  if (typeof window === 'undefined') {
    return 'ws://localhost:4000/ws';
  }
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const host = window.location.host || 'localhost:3009';
  return `${protocol}://${host}/ws`;
})();

export function getWebSocketURL(): string {
  if (typeof window === 'undefined') return WS_URL;
  const url = new URL(WS_URL, window.location.origin);
  if (url.protocol === 'http:') url.protocol = 'ws:';
  if (url.protocol === 'https:') url.protocol = 'wss:';
  return url.toString();
}

// Browser WebSocket clients cannot attach an Authorization header. Carry the
// Registry JWT in the negotiated subprotocol instead, which keeps credentials
// out of URLs, browser history, reverse-proxy logs, and monitoring traces.
export function getWebSocketProtocols(): string[] {
  const token = getStoredToken();
  return token ? [`Bearer.${token}`] : [];
}

// A launcher may hand a Registry-issued token to StatChat once. Remove it from
// the address immediately so it is not retained in browser history or copied
// when a user shares a link.
export function bootstrapSharedSignOn(): void {
  if (typeof window === 'undefined') return;
  const url = new URL(window.location.href);
  const token = url.searchParams.get('statgate_token') ?? url.searchParams.get('registry_token') ?? url.searchParams.get('access_token');
  if (!token) return;
  window.localStorage.setItem('statchat_token', token);
  url.searchParams.delete('statgate_token');
  url.searchParams.delete('registry_token');
  url.searchParams.delete('access_token');
  window.history.replaceState({}, document.title, `${url.pathname}${url.search}${url.hash}`);
}

export async function fetchCurrentUser(): Promise<User> {
  const response = await apiFetch(`${BASE_URL}/users/me`);
  if (!response.ok) {
    throw new Error(`Failed to fetch current user: ${response.status}`);
  }
  return response.json();
}

export async function fetchOrganisationBranding(): Promise<OrganisationBranding> {
  const response = await apiFetch(`${REGISTRY_API}/organisation/branding`);
  if (!response.ok) {
    throw new Error(`Failed to fetch organisation branding: ${response.status}`);
  }
  return response.json();
}

export async function fetchConversations(): Promise<Conversation[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations`);
  if (!response.ok) {
    throw new Error(`Failed to fetch conversations: ${response.status}`);
  }
  return response.json();
}

export async function fetchMessages(conversationId: string): Promise<Message[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages`);
  if (!response.ok) {
    throw new Error(`Failed to fetch messages: ${response.status}`);
  }
  return response.json();
}

export async function sendChatMessage(payload: {
  conversationId: string;
  channelId?: string;
  sender: string;
  text: string;
  tenantId?: string;
  parentMessageId?: string;
  threadRootId?: string;
  mentionUserIds?: string[];
  mentionAll?: boolean;
}): Promise<Message> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages`, {
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
  const response = await apiFetch(`${BASE_URL}/v1/chat/attachments`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    throw new Error(`Failed to upload attachment: ${response.status}`);
  }
  return response.json();
}

export async function fetchAllUsers(): Promise<User[]> {
  const response = await apiFetch(`${BASE_URL}/users`);
  if (!response.ok) {
    throw new Error(`Failed to fetch users: ${response.status}`);
  }
  return response.json();
}

export async function createDM(targetUserId: string, targetName: string): Promise<Conversation> {
  const response = await apiFetch(`${BASE_URL}/conversations/dm`, {
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
  const response = await apiFetch(`${BASE_URL}/groups/templates`);
  if (!response.ok) {
    throw new Error(`Failed to fetch group templates: ${response.status}`);
  }
  return response.json();
}

export async function createGroup(groupId: string, name: string, memberIds: string[]): Promise<Conversation> {
  const response = await apiFetch(`${BASE_URL}/conversations/group`, {
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
  const response = await apiFetch(`${BASE_URL}/users/me/profile`, {
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
  const response = await apiFetch(`${BASE_URL}/users/me/settings`);
  if (!response.ok) {
    throw new Error(`Failed to fetch settings: ${response.status}`);
  }
  return response.json();
}

export async function updateUserSettings(settings: UserSettings): Promise<UserSettings> {
  const response = await apiFetch(`${BASE_URL}/users/me/settings`, {
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
  tenantId?: string;
  authorId?: string;
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
  status: 'pending' | 'accepted' | 'declined';
  direction?: 'incoming' | 'outgoing' | 'accepted';
  canRespond?: boolean;
}

export interface Community {
  id: string;
  name: string;
  description: string;
  visibility: 'public' | 'private';
  createdBy: string;
  createdAt: string;
  memberCount: number;
  topicCount: number;
  joined: boolean;
  role?: string;
  canPost: boolean;
  latestPostAt?: string;
}

export interface CommunityMember {
  communityId: string;
  userId: string;
  name: string;
  role: 'owner' | 'member';
  userRole?: string;
  org?: string;
  joinedAt: string;
}

export interface CommunityTopic {
  id: string;
  communityId: string;
  title: string;
  body: string;
  authorId: string;
  author: string;
  role?: string;
  org?: string;
  createdAt: string;
  updatedAt?: string;
  replyCount: number;
}

export interface CommunityReply {
  id: string;
  communityId: string;
  topicId: string;
  authorId: string;
  author: string;
  role?: string;
  org?: string;
  body: string;
  createdAt: string;
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
  const response = await apiFetch(`${BASE_URL}/collaboration/posts`);
  if (!response.ok) throw new Error(`Failed to fetch posts: ${response.status}`);
  return response.json();
}

export async function createPost(post: Omit<Post, 'id' | 'createdAt'>): Promise<Post> {
  const response = await apiFetch(`${BASE_URL}/collaboration/posts`, {
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
  const response = await apiFetch(`${BASE_URL}/collaboration/posts/${postId}/like`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to toggle post like: ${response.status}`);
  return response.json();
}

export async function sharePost(postId: string): Promise<{ shares: number }> {
  const response = await apiFetch(`${BASE_URL}/collaboration/posts/${encodeURIComponent(postId)}/share`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to share post: ${response.status}`);
  return response.json();
}

export async function fetchPostComments(postId: string): Promise<PostComment[]> {
  const response = await apiFetch(`${BASE_URL}/collaboration/posts/${postId}/comments`);
  if (!response.ok) throw new Error(`Failed to fetch comments: ${response.status}`);
  return response.json();
}

export async function addPostComment(postId: string, comment: { author: string; role?: string; org?: string; text: string }): Promise<PostComment> {
  const response = await apiFetch(`${BASE_URL}/collaboration/posts/${postId}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(comment),
  });
  if (!response.ok) throw new Error(`Failed to add comment: ${response.status}`);
  return response.json();
}

export async function fetchConnections(): Promise<Connection[]> {
  const response = await apiFetch(`${BASE_URL}/collaboration/connections`);
  if (!response.ok) throw new Error(`Failed to fetch connections: ${response.status}`);
  return response.json();
}

export async function createConnection(targetUserId: string): Promise<Connection> {
  const response = await apiFetch(`${BASE_URL}/collaboration/connections`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetUserId }),
  });
  if (!response.ok) throw new Error(`Failed to create connection: ${response.status}`);
  return response.json();
}

export async function removeConnection(targetUserId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/collaboration/connections`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetUserId }),
  });
  if (!response.ok) throw new Error(`Failed to remove connection: ${response.status}`);
}

export async function fetchConnectionRequests(): Promise<Connection[]> {
  const response = await apiFetch(`${BASE_URL}/collaboration/connection-requests`);
  if (!response.ok) throw new Error(`Failed to fetch connection requests: ${response.status}`);
  return response.json();
}

export interface PostMediaUpload {
  url: string;
  fileName: string;
  mimeType: string;
}

export async function uploadPostMedia(file: File): Promise<PostMediaUpload> {
  const formData = new FormData();
  formData.append('file', file);
  const response = await apiFetch(`${BASE_URL}/collaboration/post-media`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) throw new Error(`Failed to upload post media: ${response.status}`);
  return response.json();
}

export async function respondToConnectionRequest(requestId: string, action: 'accept' | 'decline'): Promise<Connection> {
  const response = await apiFetch(`${BASE_URL}/collaboration/connection-requests/${encodeURIComponent(requestId)}/${action}`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to ${action} connection request: ${response.status}`);
  return response.json();
}

export async function fetchCommunities(): Promise<Community[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities`);
  if (!response.ok) throw new Error(`Failed to fetch communities: ${response.status}`);
  return response.json();
}

export async function createCommunity(payload: { name: string; description: string; visibility: 'public' | 'private' }): Promise<Community> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to create community: ${response.status}`);
  return response.json();
}

export async function joinCommunity(communityId: string): Promise<Community> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/join`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to join community: ${response.status}`);
  return response.json();
}

export async function leaveCommunity(communityId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/leave`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to leave community: ${response.status}`);
}

export async function fetchCommunityMembers(communityId: string): Promise<CommunityMember[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/members`);
  if (!response.ok) throw new Error(`Failed to fetch community members: ${response.status}`);
  return response.json();
}

export async function addCommunityMember(communityId: string, userId: string): Promise<CommunityMember> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/members`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to add community member: ${response.status}`);
  return response.json();
}

export async function removeCommunityMember(communityId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to remove community member: ${response.status}`);
}

export async function transferCommunityOwner(communityId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/owner`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to transfer community ownership: ${response.status}`);
}

export async function fetchCommunityTopics(communityId: string): Promise<CommunityTopic[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics`);
  if (!response.ok) throw new Error(`Failed to fetch community topics: ${response.status}`);
  return response.json();
}

export async function createCommunityTopic(communityId: string, payload: { title: string; body: string }): Promise<CommunityTopic> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to create community topic: ${response.status}`);
  return response.json();
}

export async function deleteCommunityTopic(communityId: string, topicId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics/${encodeURIComponent(topicId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete community topic: ${response.status}`);
}

export async function fetchCommunityReplies(communityId: string, topicId: string): Promise<CommunityReply[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics/${encodeURIComponent(topicId)}/replies`);
  if (!response.ok) throw new Error(`Failed to fetch community replies: ${response.status}`);
  return response.json();
}

export async function createCommunityReply(communityId: string, topicId: string, body: string): Promise<CommunityReply> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics/${encodeURIComponent(topicId)}/replies`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body }),
  });
  if (!response.ok) throw new Error(`Failed to create community reply: ${response.status}`);
  return response.json();
}

export async function deleteCommunityReply(communityId: string, topicId: string, replyId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/communities/${encodeURIComponent(communityId)}/topics/${encodeURIComponent(topicId)}/replies/${encodeURIComponent(replyId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete community reply: ${response.status}`);
}

export interface CollaborationDocument {
  id: string;
  tenantId?: string;
  title: string;
  content: string;
  createdBy: string;
  author: string;
  updatedBy: string;
  version: number;
  role: 'owner' | 'editor' | 'viewer';
  canEdit: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CollaborationDocumentMember {
  documentId: string;
  userId: string;
  name: string;
  role: 'owner' | 'editor' | 'viewer';
  org?: string;
  addedAt: string;
}

export interface CollaborationDocumentRevision {
  id: number;
  documentId: string;
  version: number;
  title: string;
  content: string;
  editedBy: string;
  editedAt: string;
}

export async function fetchDocuments(): Promise<CollaborationDocument[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents`);
  if (!response.ok) throw new Error(`Failed to fetch documents: ${response.status}`);
  return response.json();
}

export async function fetchDocument(documentId: string): Promise<CollaborationDocument> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}`);
  if (!response.ok) throw new Error(`Failed to fetch document: ${response.status}`);
  return response.json();
}

export async function createDocument(payload: { title: string; content: string }): Promise<CollaborationDocument> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to create document: ${response.status}`);
  return response.json();
}

export async function updateDocument(documentId: string, payload: { title: string; content: string; version: number }): Promise<CollaborationDocument> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to update document: ${response.status}`);
  return response.json();
}

export async function deleteDocument(documentId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete document: ${response.status}`);
}

export async function fetchDocumentMembers(documentId: string): Promise<CollaborationDocumentMember[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}/members`);
  if (!response.ok) throw new Error(`Failed to fetch document members: ${response.status}`);
  return response.json();
}

export async function addDocumentMember(documentId: string, userId: string, role: 'editor' | 'viewer'): Promise<CollaborationDocumentMember> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}/members`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, role }),
  });
  if (!response.ok) throw new Error(`Failed to add document member: ${response.status}`);
  return response.json();
}

export async function removeDocumentMember(documentId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to remove document member: ${response.status}`);
}

export async function fetchDocumentRevisions(documentId: string): Promise<CollaborationDocumentRevision[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/documents/${encodeURIComponent(documentId)}/revisions`);
  if (!response.ok) throw new Error(`Failed to fetch document revisions: ${response.status}`);
  return response.json();
}

export interface CollaborationWhiteboard {
  id: string;
  tenantId?: string;
  title: string;
  data: string;
  createdBy: string;
  author: string;
  updatedBy: string;
  version: number;
  role: 'owner' | 'editor' | 'viewer';
  canEdit: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CollaborationWhiteboardMember {
  whiteboardId: string;
  userId: string;
  name: string;
  role: 'owner' | 'editor' | 'viewer';
  org?: string;
  addedAt: string;
}

export interface CollaborationWhiteboardRevision {
  id: number;
  whiteboardId: string;
  version: number;
  title: string;
  data: string;
  editedBy: string;
  editedAt: string;
}

export async function fetchWhiteboards(): Promise<CollaborationWhiteboard[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards`);
  if (!response.ok) throw new Error(`Failed to fetch whiteboards: ${response.status}`);
  return response.json();
}

export async function fetchWhiteboard(whiteboardId: string): Promise<CollaborationWhiteboard> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}`);
  if (!response.ok) throw new Error(`Failed to fetch whiteboard: ${response.status}`);
  return response.json();
}

export async function createWhiteboard(payload: { title: string; data: string }): Promise<CollaborationWhiteboard> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to create whiteboard: ${response.status}`);
  return response.json();
}

export async function updateWhiteboard(whiteboardId: string, payload: { title: string; data: string; version: number }): Promise<CollaborationWhiteboard> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to update whiteboard: ${response.status}`);
  return response.json();
}

export async function deleteWhiteboard(whiteboardId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete whiteboard: ${response.status}`);
}

export async function fetchWhiteboardMembers(whiteboardId: string): Promise<CollaborationWhiteboardMember[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}/members`);
  if (!response.ok) throw new Error(`Failed to fetch whiteboard members: ${response.status}`);
  return response.json();
}

export async function addWhiteboardMember(whiteboardId: string, userId: string, role: 'editor' | 'viewer'): Promise<CollaborationWhiteboardMember> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}/members`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, role }),
  });
  if (!response.ok) throw new Error(`Failed to add whiteboard member: ${response.status}`);
  return response.json();
}

export async function removeWhiteboardMember(whiteboardId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to remove whiteboard member: ${response.status}`);
}

export async function fetchWhiteboardRevisions(whiteboardId: string): Promise<CollaborationWhiteboardRevision[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/whiteboards/${encodeURIComponent(whiteboardId)}/revisions`);
  if (!response.ok) throw new Error(`Failed to fetch whiteboard revisions: ${response.status}`);
  return response.json();
}

export interface TranslationLanguage {
  code: string;
  name: string;
}

export interface TranslationResult {
  sourceLanguage: string;
  targetLanguage: string;
  text: string;
  type?: 'text' | 'photo' | 'video' | 'article';
  title?: string;
  mediaUrl?: string;
  mediaMime?: string;
  provider: string;
}

export async function fetchTranslationLanguages(): Promise<TranslationLanguage[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/translation/languages`);
  if (!response.ok) throw new Error(`Failed to fetch translation languages: ${response.status}`);
  return response.json();
}

export async function translateText(payload: { text: string; sourceLanguage: string; targetLanguage: string }): Promise<TranslationResult> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/translation/translate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to translate text: ${response.status}`);
  return response.json();
}

export async function fetchOpportunities(): Promise<Opportunity[]> {
  const response = await apiFetch(`${BASE_URL}/collaboration/opportunities`);
  if (!response.ok) throw new Error(`Failed to fetch opportunities: ${response.status}`);
  return response.json();
}

export async function fetchJobs(): Promise<Job[]> {
  const response = await apiFetch(`${BASE_URL}/collaboration/jobs`);
  if (!response.ok) throw new Error(`Failed to fetch jobs: ${response.status}`);
  return response.json();
}

export async function fetchMeetings(): Promise<Meeting[]> {
  const response = await apiFetch(`${BASE_URL}/meetings`);
  if (!response.ok) throw new Error(`Failed to fetch meetings: ${response.status}`);
  return response.json();
}

export async function createMeeting(meeting: Omit<Meeting, 'id' | 'createdAt'>): Promise<Meeting> {
  const response = await apiFetch(`${BASE_URL}/meetings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(meeting),
  });
  if (!response.ok) throw new Error(`Failed to create meeting: ${response.status}`);
  return response.json();
}

export async function fetchMeetingRooms(): Promise<MeetingRoom[]> {
  const response = await apiFetch(`${BASE_URL}/meetings/rooms`);
  if (!response.ok) throw new Error(`Failed to fetch meeting rooms: ${response.status}`);
  return response.json();
}

export async function fetchMeetingRecordings(): Promise<MeetingRecording[]> {
  const response = await apiFetch(`${BASE_URL}/meetings/recordings`);
  if (!response.ok) throw new Error(`Failed to fetch recordings: ${response.status}`);
  return response.json();
}

export interface WellnessPost {
  id: string;
  tenantId?: string;
  authorId?: string;
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
  likedByMe?: boolean;
  bookmarkedByMe?: boolean;
}

export interface WellnessComment {
  id: string;
  tenantId?: string;
  authorId?: string;
  postId: string;
  author: string;
  role?: string;
  org?: string;
  text: string;
  createdAt: string;
}

export async function fetchWellnessPosts(): Promise<WellnessPost[]> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts`);
  if (!response.ok) throw new Error(`Failed to fetch wellness posts: ${response.status}`);
  return response.json();
}

export async function createWellnessPost(post: Omit<WellnessPost, 'id' | 'createdAt'>): Promise<WellnessPost> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(post),
  });
  if (!response.ok) throw new Error(`Failed to create wellness post: ${response.status}`);
  return response.json();
}

export interface KnowledgeExpert {
  id: string;
  tenantId?: string;
  name: string;
  role: string;
  org: string;
  specialties: string[];
  followers: number;
  articles: number;
  rating: number;
  avatar: string;
  following?: boolean;
}

export interface KnowledgeArticle {
  id: string;
  tenantId?: string;
  title: string;
  author: string;
  category: string;
  readTime: string;
  excerpt: string;
  content?: string;
  likes: number;
  views: number;
  published: string;
}

export interface KnowledgeIdea {
  id: string;
  tenantId?: string;
  title: string;
  author: string;
  authorId?: string;
  category: string;
  description: string;
  votes: number;
  status: string;
  upvoted?: boolean;
}

export interface KnowledgePost {
  id: string;
  tenantId?: string;
  title: string;
  author: string;
  authorId?: string;
  role?: string;
  org?: string;
  category: string;
  content: string;
  createdBy: string;
  createdAt: string;
}

export async function fetchKnowledgeExperts(): Promise<KnowledgeExpert[]> {
  const response = await apiFetch(`${BASE_URL}/knowledge/experts`);
  if (!response.ok) throw new Error(`Failed to fetch experts: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgeArticles(): Promise<KnowledgeArticle[]> {
  const response = await apiFetch(`${BASE_URL}/knowledge/articles`);
  if (!response.ok) throw new Error(`Failed to fetch articles: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgeIdeas(): Promise<KnowledgeIdea[]> {
  const response = await apiFetch(`${BASE_URL}/knowledge/ideas`);
  if (!response.ok) throw new Error(`Failed to fetch ideas: ${response.status}`);
  return response.json();
}

export async function fetchKnowledgePosts(): Promise<KnowledgePost[]> {
  const response = await apiFetch(`${BASE_URL}/knowledge/posts`);
  if (!response.ok) throw new Error(`Failed to fetch posts: ${response.status}`);
  return response.json();
}

export async function createKnowledgePost(post: Pick<KnowledgePost, 'title' | 'category' | 'content'>): Promise<KnowledgePost> {
  const response = await apiFetch(`${BASE_URL}/knowledge/posts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(post),
  });
  if (!response.ok) throw new Error(`Failed to create post: ${response.status}`);
  return response.json();
}

// ── Message Edit / Delete ──

export async function editChatMessage(messageId: string, text: string): Promise<Message> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${messageId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!response.ok) throw new Error(`Failed to edit message: ${response.status}`);
  return response.json();
}

export async function deleteChatMessage(messageId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${messageId}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(`Failed to delete message: ${response.status}`);
}

export async function forwardChatMessage(messageId: string, targetConversationId: string): Promise<Message> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${encodeURIComponent(messageId)}/forward`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targetConversationId }),
  });
  if (!response.ok) throw new Error(`Failed to forward message: ${response.status}`);
  return response.json();
}

export async function fetchWellnessComments(postId: string): Promise<WellnessComment[]> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts/${encodeURIComponent(postId)}/comments`);
  if (!response.ok) throw new Error(`Failed to fetch wellness comments: ${response.status}`);
  return response.json();
}

export async function addWellnessComment(postId: string, text: string): Promise<WellnessComment> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts/${encodeURIComponent(postId)}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  });
  if (!response.ok) throw new Error(`Failed to add wellness comment: ${response.status}`);
  return response.json();
}

export async function toggleWellnessLike(postId: string): Promise<{ liked: boolean; likes: number }> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts/${encodeURIComponent(postId)}/like`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to update wellness like: ${response.status}`);
  return response.json();
}

export async function toggleWellnessBookmark(postId: string): Promise<{ bookmarked: boolean; bookmarks: number }> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts/${encodeURIComponent(postId)}/bookmark`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to update wellness bookmark: ${response.status}`);
  return response.json();
}

export async function shareWellnessPost(postId: string): Promise<{ shares: number }> {
  const response = await apiFetch(`${BASE_URL}/wellness/posts/${encodeURIComponent(postId)}/share`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to share wellness post: ${response.status}`);
  return response.json();
}

export async function fetchScheduledMessages(conversationId?: string): Promise<ScheduledMessage[]> {
  const query = conversationId ? `?conversationId=${encodeURIComponent(conversationId)}` : '';
  const response = await apiFetch(`${BASE_URL}/v1/chat/scheduled${query}`);
  if (!response.ok) throw new Error(`Failed to load scheduled messages: ${response.status}`);
  return response.json();
}

export async function scheduleChatMessage(conversationId: string, text: string, scheduledFor: string): Promise<ScheduledMessage> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/scheduled`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conversationId, text, scheduledFor }),
  });
  if (!response.ok) throw new Error(`Failed to schedule message: ${response.status}`);
  return response.json();
}

export async function cancelScheduledMessage(id: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/scheduled/${encodeURIComponent(id)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to cancel scheduled message: ${response.status}`);
}

export async function searchChatMedia(search = '', kind?: 'gif' | 'sticker'): Promise<MediaCatalogItem[]> {
  const params = new URLSearchParams();
  if (search.trim()) params.set('q', search.trim());
  if (kind) params.set('kind', kind);
  const query = params.size ? `?${params.toString()}` : '';
  const response = await apiFetch(`${BASE_URL}/v1/chat/media${query}`);
  if (!response.ok) throw new Error(`Failed to search media: ${response.status}`);
  return response.json();
}

export async function sendChatMedia(conversationId: string, mediaId: string): Promise<Message> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/media/send`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conversationId, mediaId }),
  });
  if (!response.ok) throw new Error(`Failed to send media: ${response.status}`);
  return response.json();
}

export async function sendChatLocation(conversationId: string, location: { latitude: number; longitude: number; accuracyMeters?: number; label?: string }): Promise<Message> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/location`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conversationId, ...location }),
  });
  if (!response.ok) throw new Error(`Failed to share location: ${response.status}`);
  return response.json();
}

// ── Message Reactions ──

export async function addReaction(messageId: string, emoji: string, userId?: string): Promise<MessageReaction> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${messageId}/reactions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ emoji, userId }),
  });
  if (!response.ok) throw new Error(`Failed to add reaction: ${response.status}`);
  return response.json();
}

export async function removeReaction(messageId: string, emoji: string, userId?: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${messageId}/reactions`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ emoji, userId }),
  });
  if (!response.ok) throw new Error(`Failed to remove reaction: ${response.status}`);
}

// ── Read Receipts ──

export async function markMessageRead(messageId: string, userId?: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${messageId}/read`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to mark message as read: ${response.status}`);
}

// ── Pinned Messages ──

export async function fetchPinnedMessages(conversationId: string): Promise<PinnedMessage[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/pinned`);
  if (!response.ok) throw new Error(`Failed to fetch pinned messages: ${response.status}`);
  return response.json();
}

export async function pinMessage(conversationId: string, messageId: string, pinnedBy?: string): Promise<PinnedMessage> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/pinned`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageId, pinnedBy }),
  });
  if (!response.ok) throw new Error(`Failed to pin message: ${response.status}`);
  return response.json();
}

export async function unpinMessage(conversationId: string, messageId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/pinned`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageId }),
  });
  if (!response.ok) throw new Error(`Failed to unpin message: ${response.status}`);
}

// ── Tasks ──

export async function fetchTasks(conversationId?: string): Promise<Task[]> {
  const query = conversationId ? `?conversationId=${encodeURIComponent(conversationId)}` : '';
  const response = await apiFetch(`${BASE_URL}/v1/tasks${query}`);
  if (!response.ok) throw new Error(`Failed to fetch tasks: ${response.status}`);
  return response.json();
}

export async function createTask(task: Omit<Task, 'id' | 'createdAt' | 'updatedAt'>): Promise<Task> {
  const response = await apiFetch(`${BASE_URL}/v1/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task),
  });
  if (!response.ok) throw new Error(`Failed to create task: ${response.status}`);
  return response.json();
}

export async function updateTaskStatus(taskId: string, status: string): Promise<Task> {
  const response = await apiFetch(`${BASE_URL}/v1/tasks/${taskId}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
  if (!response.ok) throw new Error(`Failed to update task status: ${response.status}`);
  return response.json();
}

export async function fetchObjectConversation(objectRef: string): Promise<Conversation> {
  const query = new URLSearchParams({ objectRef });
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/object?${query}`);
  if (!response.ok) throw new Error(`Failed to fetch object conversation: ${response.status}`);
  return response.json();
}

export async function ensureObjectConversation(input: { objectRef: string; name: string; memberIds?: string[]; metadata?: Record<string, unknown> }): Promise<Conversation> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/object`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input),
  });
  if (!response.ok) throw new Error(`Failed to create object conversation: ${response.status}`);
  return response.json();
}

export interface ConversationMembership {
  memberIds: string[];
  ownerId: string;
  canManage: boolean;
}

export async function fetchConversationMembers(conversationId: string): Promise<ConversationMembership> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${encodeURIComponent(conversationId)}/members`);
  if (!response.ok) throw new Error(`Failed to fetch conversation members: ${response.status}`);
  return response.json();
}

export async function addConversationMember(conversationId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${encodeURIComponent(conversationId)}/members`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ userId }),
  });
  if (!response.ok) throw new Error(`Failed to add conversation member: ${response.status}`);
}

export async function removeConversationMember(conversationId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${encodeURIComponent(conversationId)}/members/${encodeURIComponent(userId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to remove conversation member: ${response.status}`);
}

export async function fetchPolls(): Promise<Poll[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/polls`);
  if (!response.ok) throw new Error(`Failed to fetch polls: ${response.status}`);
  return response.json();
}

export async function createPoll(question: string, options: string[]): Promise<Poll> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/polls`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ question, options: options.map((label) => ({ label })) }),
  });
  if (!response.ok) throw new Error(`Failed to create poll: ${response.status}`);
  return response.json();
}

export async function votePoll(pollId: string, optionId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/polls/${encodeURIComponent(pollId)}/vote`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ optionId }),
  });
  if (!response.ok) throw new Error(`Failed to vote in poll: ${response.status}`);
}

export async function requestChatAssistant(conversationId: string, mode: 'summary' | 'actions' | 'draft'): Promise<string> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/assistant`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conversationId, mode }),
  });
  if (!response.ok) throw new Error(`Assistant request failed: ${response.status}`);
  const result: { response: string } = await response.json();
  return result.response;
}

export async function updateTask(taskId: string, task: Partial<Task>): Promise<Task> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/tasks/${encodeURIComponent(taskId)}`, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(task),
  });
  if (!response.ok) throw new Error(`Failed to update task: ${response.status}`);
  return response.json();
}

export async function deleteTask(taskId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/tasks/${encodeURIComponent(taskId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete task: ${response.status}`);
}

export async function fetchChannels(): Promise<Channel[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels`);
  if (!response.ok) throw new Error(`Failed to fetch channels: ${response.status}`);
  return response.json();
}

export async function fetchChannel(channelId: string): Promise<Channel> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels/${encodeURIComponent(channelId)}`);
  if (!response.ok) throw new Error(`Failed to fetch channel: ${response.status}`);
  return response.json();
}

export async function createChannel(channel: Pick<Channel, 'name' | 'visibility'> & { description?: string }): Promise<Channel> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(channel),
  });
  if (!response.ok) throw new Error(`Failed to create channel: ${response.status}`);
  return response.json();
}

export async function joinChannel(channelId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels/${encodeURIComponent(channelId)}/join`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to join channel: ${response.status}`);
}

export async function leaveChannel(channelId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels/${encodeURIComponent(channelId)}/leave`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to leave channel: ${response.status}`);
}

export async function archiveChannel(channelId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/channels/${encodeURIComponent(channelId)}/archive`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to archive channel: ${response.status}`);
}

export async function fetchCalendarEvents(from?: string, to?: string): Promise<CalendarEvent[]> {
  const query = new URLSearchParams();
  if (from) query.set('from', from);
  if (to) query.set('to', to);
  const response = await apiFetch(`${BASE_URL}/v1/chat/calendar${query.size ? `?${query}` : ''}`);
  if (!response.ok) throw new Error(`Failed to fetch calendar events: ${response.status}`);
  return response.json();
}

export async function fetchCalendarEvent(eventId: string): Promise<CalendarEvent> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/calendar/${encodeURIComponent(eventId)}`);
  if (!response.ok) throw new Error(`Failed to fetch calendar event: ${response.status}`);
  return response.json();
}

export async function createCalendarEvent(event: Omit<CalendarEvent, 'id' | 'createdBy' | 'createdAt' | 'updatedAt'>): Promise<CalendarEvent> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/calendar`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(event),
  });
  if (!response.ok) throw new Error(`Failed to create calendar event: ${response.status}`);
  return response.json();
}

export async function updateCalendarEvent(eventId: string, event: Partial<CalendarEvent>): Promise<CalendarEvent> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/calendar/${encodeURIComponent(eventId)}`, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(event),
  });
  if (!response.ok) throw new Error(`Failed to update calendar event: ${response.status}`);
  return response.json();
}

export async function deleteCalendarEvent(eventId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/calendar/${encodeURIComponent(eventId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to delete calendar event: ${response.status}`);
}

// ── Notifications ──

export async function fetchNotifications(): Promise<Notification[]> {
  const response = await apiFetch(`${BASE_URL}/v1/notifications`);
  if (!response.ok) throw new Error(`Failed to fetch notifications: ${response.status}`);
  return response.json();
}

export async function markNotificationRead(notificationId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/notifications/${notificationId}/read`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to mark notification as read: ${response.status}`);
}

export async function markAllNotificationsRead(): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/notifications/read-all`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to mark all notifications as read: ${response.status}`);
}

// ── Presence ──

export async function fetchPresence(userId?: string): Promise<Presence[] | Presence> {
  const query = userId ? `?userId=${encodeURIComponent(userId)}` : '';
  const response = await apiFetch(`${BASE_URL}/v1/presence${query}`);
  if (!response.ok) throw new Error(`Failed to fetch presence: ${response.status}`);
  return response.json();
}

export async function updatePresence(status: string, userId?: string): Promise<Presence> {
  const response = await apiFetch(`${BASE_URL}/v1/presence`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status, userId }),
  });
  if (!response.ok) throw new Error(`Failed to update presence: ${response.status}`);
  return response.json();
}

// ── Conferencing (Native WebRTC) ──

export async function createCallSession(payload: {
  kind: 'voice' | 'video';
  roomName?: string;
  conversationId?: string;
  hostId?: string;
  hostName?: string;
}): Promise<CallSession> {
  const response = await apiFetch(`${BASE_URL}/v1/calls`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(`Failed to create call session: ${response.status}`);
  return response.json();
}

export async function fetchCallSessions(): Promise<CallSession[]> {
  const response = await apiFetch(`${BASE_URL}/v1/calls`);
  if (!response.ok) throw new Error(`Failed to list call sessions: ${response.status}`);
  return response.json();
}

export async function fetchCallSession(sessionId: string): Promise<{ session: CallSession; participants: CallParticipant[] }> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}`);
  if (!response.ok) throw new Error(`Failed to fetch call session: ${response.status}`);
  return response.json();
}

export async function joinCallSession(sessionId: string, userId?: string, userName?: string): Promise<CallParticipant> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/join`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, userName }),
  });
  if (!response.ok) throw new Error(`Failed to join call session: ${response.status}`);
  return response.json();
}

export async function leaveCallSession(sessionId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/leave`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to leave call session: ${response.status}`);
}

export async function endCallSession(sessionId: string): Promise<CallSession> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/end`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to end call session: ${response.status}`);
  return response.json();
}

export async function fetchCallParticipants(sessionId: string): Promise<CallParticipant[]> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/participants`);
  if (!response.ok) throw new Error(`Failed to fetch call participants: ${response.status}`);
  return response.json();
}

export async function uploadCallRecording(sessionId: string, formData: FormData): Promise<CallRecording> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/recordings`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) throw new Error(`Failed to upload call recording: ${response.status}`);
  return response.json();
}

export async function fetchCallRecordings(sessionId: string): Promise<CallRecording[]> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/recordings`);
  if (!response.ok) throw new Error(`Failed to fetch call recordings: ${response.status}`);
  return response.json();
}

export async function fetchMeetingCallSession(meetingId: string): Promise<CallSession> {
  const response = await apiFetch(`${BASE_URL}/v1/meetings/${meetingId}/session`);
  if (!response.ok) throw new Error(`Failed to fetch meeting session: ${response.status}`);
  return response.json();
}

// ── Auth Token ──

export function setAuthToken(token: string): void {
  if (typeof window === 'undefined') return;
  if (token) {
    window.localStorage.setItem('statchat_token', token);
  } else {
    window.localStorage.removeItem('statchat_token');
  }
}

export function getAuthToken(): string {
  return getStoredToken();
}

// ── Global Search ──

function appendMessageSearchFilters(params: URLSearchParams, filters: MessageSearchFilters = {}) {
  if (filters.conversationId) params.set('conversationId', filters.conversationId);
  if (filters.sender?.trim()) params.set('sender', filters.sender.trim());
  if (filters.from) params.set('from', filters.from);
  if (filters.to) params.set('to', filters.to);
  if (filters.hasAttachment !== undefined) params.set('hasAttachment', String(filters.hasAttachment));
  if (filters.savedOnly !== undefined) params.set('savedOnly', String(filters.savedOnly));
}

export async function removeCallParticipant(sessionId: string, userId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${encodeURIComponent(sessionId)}/participants/${encodeURIComponent(userId)}/remove`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to remove call participant: ${response.status}`);
}

export async function updateCallParticipantRole(sessionId: string, userId: string, role: 'moderator' | 'participant'): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${encodeURIComponent(sessionId)}/participants/${encodeURIComponent(userId)}/role`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  });
  if (!response.ok) throw new Error(`Failed to update call participant role: ${response.status}`);
}

export async function reportCallQuality(sessionId: string, metrics: Pick<CallQualitySample, 'rttMs' | 'jitterMs' | 'packetLossPct' | 'bitrateKbps'>): Promise<CallQualitySample> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/quality`, { method: 'POST', body: JSON.stringify(metrics) });
  if (!response.ok) throw new Error(`Failed to report call quality: ${response.status}`);
  return response.json();
}

export async function fetchCallQuality(sessionId: string, limit = 100): Promise<CallQualitySample[]> {
  const response = await apiFetch(`${BASE_URL}/v1/calls/${sessionId}/quality?limit=${limit}`);
  if (!response.ok) throw new Error(`Failed to fetch call quality: ${response.status}`);
  return response.json();
}

export async function searchAPI(query: string, filters: MessageSearchFilters = {}): Promise<SearchResult> {
  const params = new URLSearchParams();
  if (query.trim()) params.set('q', query.trim());
  appendMessageSearchFilters(params, filters);
  const response = await apiFetch(`${BASE_URL}/v1/search?${params.toString()}`);
  if (!response.ok) throw new Error(`Failed to search: ${response.status}`);
  return response.json();
}

export async function saveChatMessage(messageId: string): Promise<{ messageId: string; savedAt: string }> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${encodeURIComponent(messageId)}/saved`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to save message: ${response.status}`);
  return response.json();
}

export async function unsaveChatMessage(messageId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/messages/${encodeURIComponent(messageId)}/saved`, { method: 'DELETE' });
  if (!response.ok) throw new Error(`Failed to remove saved message: ${response.status}`);
}

export async function fetchSavedMessages(filters: MessageSearchFilters = {}): Promise<Message[]> {
  const params = new URLSearchParams();
  appendMessageSearchFilters(params, filters);
  const suffix = params.size ? `?${params.toString()}` : '';
  const response = await apiFetch(`${BASE_URL}/v1/chat/saved${suffix}`);
  if (!response.ok) throw new Error(`Failed to load saved messages: ${response.status}`);
  return response.json();
}

// Compliance and authenticated exports

export async function fetchRetentionPolicy(): Promise<RetentionPolicy> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/retention`);
  if (!response.ok) throw new Error(`Failed to load retention policy: ${response.status}`);
  return response.json();
}

export async function updateRetentionPolicy(retentionDays: number, enabled: boolean): Promise<RetentionPolicy> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/retention`, {
    method: 'PUT',
    body: JSON.stringify({ retentionDays, enabled }),
  });
  if (!response.ok) throw new Error(`Failed to update retention policy: ${response.status}`);
  return response.json();
}

export async function enforceRetention(): Promise<{ deletedMessages: number }> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/retention/enforce`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to enforce retention policy: ${response.status}`);
  return response.json();
}

export async function fetchLegalHolds(): Promise<LegalHold[]> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/legal-holds`);
  if (!response.ok) throw new Error(`Failed to load legal holds: ${response.status}`);
  return response.json();
}

export async function createLegalHold(input: { name: string; reason: string; conversationId?: string }): Promise<LegalHold> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/legal-holds`, { method: 'POST', body: JSON.stringify(input) });
  if (!response.ok) throw new Error(`Failed to create legal hold: ${response.status}`);
  return response.json();
}

export async function releaseLegalHold(holdId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/legal-holds/${encodeURIComponent(holdId)}/release`, { method: 'POST' });
  if (!response.ok) throw new Error(`Failed to release legal hold: ${response.status}`);
}

export async function fetchComplianceAudit(limit = 100): Promise<ComplianceAuditEvent[]> {
  const response = await apiFetch(`${BASE_URL}/v1/compliance/audit?limit=${limit}`);
  if (!response.ok) throw new Error(`Failed to load compliance audit: ${response.status}`);
  return response.json();
}

export async function exportConversation(conversationId: string, format: 'json' | 'csv', includeDeleted = false): Promise<{ blob: Blob; filename: string }> {
  const params = new URLSearchParams({ format, includeDeleted: String(includeDeleted) });
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${encodeURIComponent(conversationId)}/export?${params}`);
  if (!response.ok) throw new Error(`Failed to export conversation: ${response.status}`);
  const disposition = response.headers.get('Content-Disposition') ?? '';
  const filename = disposition.match(/filename="?([^";]+)"?/i)?.[1] ?? `statchat-${conversationId}.${format}`;
  return { blob: await response.blob(), filename };
}

// ── Favourites ──

export async function toggleFavourite(conversationId: string): Promise<Favourite> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/favourite`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to toggle favourite: ${response.status}`);
  return response.json();
}

export async function fetchFavouriteStatus(conversationId: string): Promise<Favourite> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/favourite`);
  if (!response.ok) throw new Error(`Failed to fetch favourite status: ${response.status}`);
  return response.json();
}

export async function fetchFavouriteIds(): Promise<string[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/favourites`);
  if (!response.ok) throw new Error(`Failed to fetch favourite ids: ${response.status}`);
  return response.json();
}

// ── Mute ──

export async function muteConversation(conversationId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/mute`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to mute conversation: ${response.status}`);
}

export async function unmuteConversation(conversationId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/mute`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(`Failed to unmute conversation: ${response.status}`);
}

export async function fetchMutedIds(): Promise<string[]> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/muted`);
  if (!response.ok) throw new Error(`Failed to fetch muted ids: ${response.status}`);
  return response.json();
}

// ── Knowledge Hub interactions ──

export async function upvoteKnowledgeIdea(ideaId: string): Promise<{ votes: number }> {
  const response = await apiFetch(`${BASE_URL}/knowledge/ideas/${ideaId}/upvote`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to upvote idea: ${response.status}`);
  return response.json();
}

export async function followKnowledgeExpert(expertId: string): Promise<{ followers: number }> {
  const response = await apiFetch(`${BASE_URL}/knowledge/experts/${expertId}/follow`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error(`Failed to follow expert: ${response.status}`);
  return response.json();
}

// ── Clear Conversation ──

export async function clearConversation(conversationId: string): Promise<void> {
  const response = await apiFetch(`${BASE_URL}/v1/chat/conversations/${conversationId}/messages`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error(`Failed to clear conversation: ${response.status}`);
}
