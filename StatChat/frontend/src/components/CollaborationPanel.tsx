import { useEffect, useState } from 'react';
import type { User } from '../types';
import type { SubHeaderView } from '../App';
import {
  fetchAllUsers,
  fetchPosts,
  createPost,
  fetchConnections,
  createConnection,
  removeConnection,
  fetchOpportunities,
  fetchJobs,
  fetchCommunities,
  createCommunity,
  joinCommunity,
  leaveCommunity,
  fetchCommunityMembers,
  addCommunityMember,
  removeCommunityMember,
  transferCommunityOwner,
  fetchCommunityTopics,
  createCommunityTopic,
  deleteCommunityTopic,
  fetchCommunityReplies,
  createCommunityReply,
  deleteCommunityReply,
  togglePostLike,
  sharePost,
  addPostComment,
  fetchPostComments,
  fetchPolls,
  createPoll,
  votePoll,
  type Post,
  type Connection,
  type Opportunity,
  type Job,
  type PostComment,
  type Community,
  type CommunityMember,
  type CommunityTopic,
  type CommunityReply,
} from '../api/client';
import type { Poll } from '../types';
import styles from './CollaborationPanel.module.css';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
  activeSubView: SubHeaderView;
}

type FeedPost = Post;
type JobListing = Job;
const seedOpportunities: Opportunity[] = [];
const seedPosts: FeedPost[] = [];
const seedJobs: JobListing[] = [];

function getAvatarText(name: string) {
  const parts = name.trim().split(' ');
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

export default function CollaborationPanel({ user, theme, isMobile, activeSubView }: Props) {
  const [posts, setPosts] = useState<FeedPost[]>([]);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [connections, setConnections] = useState<Connection[]>([]);
  const [opportunities, setOpportunities] = useState<Opportunity[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
const [postDraft, setPostDraft] = useState('');
  const [showPostForm, setShowPostForm] = useState(false);
  const [commentDrafts, setCommentDrafts] = useState<Record<string, string>>({});
  const [expandedComments, setExpandedComments] = useState<Record<string, boolean>>({});
  const [postComments, setPostComments] = useState<Record<string, PostComment[]>>({});
  const [polls, setPolls] = useState<Poll[]>([]);
  const [pollQuestion, setPollQuestion] = useState('');
  const [pollOptions, setPollOptions] = useState(['', '']);
  const [showPollForm, setShowPollForm] = useState(false);
  const [communities, setCommunities] = useState<Community[]>([]);
  const [selectedCommunityId, setSelectedCommunityId] = useState('');
  const [communityMembers, setCommunityMembers] = useState<Record<string, CommunityMember[]>>({});
  const [communityTopics, setCommunityTopics] = useState<Record<string, CommunityTopic[]>>({});
  const [communityReplies, setCommunityReplies] = useState<Record<string, CommunityReply[]>>({});
  const [expandedTopicId, setExpandedTopicId] = useState('');
  const [communityDraft, setCommunityDraft] = useState({ name: '', description: '', visibility: 'public' as 'public' | 'private' });
  const [topicDraft, setTopicDraft] = useState({ title: '', body: '' });
  const [replyDrafts, setReplyDrafts] = useState<Record<string, string>>({});
  const [inviteTargetId, setInviteTargetId] = useState('');

  const connectedIds = new Set(connections.map((c) => c.connectedToId));

  const isDark = theme === 'dark';
  const bg = isDark ? '#0a2b45' : '#ffffff';
  const textColor = isDark ? '#e8eef4' : '#1a1a1a';
  const borderColor = isDark ? '#6b7280' : '#e5e7eb';

  useEffect(() => {
    fetchPosts().then(setPosts).catch(() => {});
    fetchConnections().then(setConnections).catch(() => {});
    fetchOpportunities().then(setOpportunities).catch(() => {});
    fetchJobs().then(setJobs).catch(() => {});
    fetchPolls().then(setPolls).catch(() => {});
    fetchCommunities().then((items) => {
      setCommunities(items);
      setSelectedCommunityId((current) => current || items[0]?.id || '');
    }).catch(() => {});
  }, []);

  useEffect(() => {
    if ((activeSubView === 'connect' || activeSubView === 'network' || activeSubView === 'communities') && allUsers.length === 0) {
      fetchAllUsers()
        .then((users) => setAllUsers(users.filter((u) => u.id !== user?.id).slice(0, 24)))
        .catch(() => {});
    }
  }, [activeSubView, allUsers.length, user?.id]);

  const handlePost = async () => {
    if (!postDraft.trim()) return;
    try {
      const newPost = await createPost({
        author: user?.name ?? 'StatChat User',
        role: user?.roles?.[0] ?? 'Member',
        org: user?.organizationId ?? 'StatGate',
        time: 'now',
        text: postDraft.trim(),
        likes: 0,
        comments: 0,
        shares: 0,
      });
      setPosts((prev) => [newPost, ...prev]);
      setPostDraft('');
      setShowPostForm(false);
    } catch {
      // ignore
    }
  };

  const handleCreatePoll = async () => {
    const options = pollOptions.map((option) => option.trim()).filter(Boolean);
    if (!pollQuestion.trim() || options.length < 2) return;
    try {
      const poll = await createPoll(pollQuestion.trim(), options);
      setPolls((previous) => [poll, ...previous]);
      setPollQuestion('');
      setPollOptions(['', '']);
      setShowPollForm(false);
    } catch { /* Keep the collaboration feed usable. */ }
  };

  const selectedCommunity = communities.find((community) => community.id === selectedCommunityId);

  const refreshCommunityTopics = async (communityId: string) => {
    const topics = await fetchCommunityTopics(communityId);
    setCommunityTopics((previous) => ({ ...previous, [communityId]: topics }));
  };

  const refreshCommunityMembers = async (communityId: string) => {
    const members = await fetchCommunityMembers(communityId);
    setCommunityMembers((previous) => ({ ...previous, [communityId]: members }));
  };

  const handleCreateCommunity = async () => {
    if (!communityDraft.name.trim()) return;
    try {
      const created = await createCommunity({
        name: communityDraft.name.trim(),
        description: communityDraft.description.trim(),
        visibility: communityDraft.visibility,
      });
      setCommunities((previous) => [created, ...previous]);
      setSelectedCommunityId(created.id);
      setCommunityMembers((previous) => ({
        ...previous,
        [created.id]: [{
          communityId: created.id,
          userId: user?.id || created.createdBy,
          name: user?.name || 'Community Owner',
          role: 'owner',
          userRole: user?.roles?.[0],
          org: user?.organizationId,
          joinedAt: created.createdAt,
        }],
      }));
      setCommunityDraft({ name: '', description: '', visibility: 'public' });
    } catch { /* Keep the community workspace usable. */ }
  };

  const handleCommunityMembership = async (community: Community) => {
    try {
      if (community.joined && community.role !== 'owner') {
        await leaveCommunity(community.id);
        setCommunities((previous) => previous.map((item) => item.id === community.id ? { ...item, joined: false, canPost: false, role: '', memberCount: Math.max(0, item.memberCount - 1) } : item));
        return;
      }
      if (!community.joined) {
        const joined = await joinCommunity(community.id);
        setCommunities((previous) => previous.map((item) => item.id === community.id ? joined : item));
        setSelectedCommunityId(joined.id);
        refreshCommunityMembers(joined.id).catch(() => {});
      }
    } catch { /* Private communities stay invite-only through owner-managed membership. */ }
  };

  const handleInviteMember = async () => {
    if (!selectedCommunity || !inviteTargetId) return;
    try {
      const member = await addCommunityMember(selectedCommunity.id, inviteTargetId);
      setCommunityMembers((previous) => ({ ...previous, [selectedCommunity.id]: [member, ...(previous[selectedCommunity.id] || []).filter((item) => item.userId !== member.userId)] }));
      setCommunities((previous) => previous.map((item) => item.id === selectedCommunity.id ? { ...item, memberCount: item.memberCount + 1 } : item));
      setInviteTargetId('');
    } catch { /* ignore */ }
  };

  const handleRemoveMember = async (memberId: string) => {
    if (!selectedCommunity) return;
    try {
      await removeCommunityMember(selectedCommunity.id, memberId);
      setCommunityMembers((previous) => ({ ...previous, [selectedCommunity.id]: (previous[selectedCommunity.id] || []).filter((member) => member.userId !== memberId) }));
      setCommunities((previous) => previous.map((item) => item.id === selectedCommunity.id ? { ...item, memberCount: Math.max(1, item.memberCount - 1) } : item));
    } catch { /* ignore */ }
  };

  const handleTransferOwner = async (memberId: string) => {
    if (!selectedCommunity) return;
    try {
      await transferCommunityOwner(selectedCommunity.id, memberId);
      setCommunityMembers((previous) => ({
        ...previous,
        [selectedCommunity.id]: (previous[selectedCommunity.id] || []).map((member) => ({
          ...member,
          role: member.userId === memberId ? 'owner' : member.userId === user?.id ? 'member' : member.role,
        })),
      }));
      setCommunities((previous) => previous.map((item) => item.id === selectedCommunity.id ? { ...item, role: 'member', createdBy: memberId } : item));
    } catch { /* ignore */ }
  };

  const handleCreateTopic = async () => {
    if (!selectedCommunity || !topicDraft.title.trim() || !topicDraft.body.trim()) return;
    try {
      const topic = await createCommunityTopic(selectedCommunity.id, { title: topicDraft.title.trim(), body: topicDraft.body.trim() });
      setCommunityTopics((previous) => ({ ...previous, [selectedCommunity.id]: [topic, ...(previous[selectedCommunity.id] || [])] }));
      setCommunities((previous) => previous.map((item) => item.id === selectedCommunity.id ? { ...item, topicCount: item.topicCount + 1 } : item));
      setTopicDraft({ title: '', body: '' });
    } catch { /* ignore */ }
  };

  const toggleTopicReplies = async (communityId: string, topicId: string) => {
    const nextTopicId = expandedTopicId === topicId ? '' : topicId;
    setExpandedTopicId(nextTopicId);
    if (nextTopicId && !communityReplies[topicId]) {
      try {
        const replies = await fetchCommunityReplies(communityId, topicId);
        setCommunityReplies((previous) => ({ ...previous, [topicId]: replies }));
      } catch { /* ignore */ }
    }
  };

  const handleCreateReply = async (communityId: string, topicId: string) => {
    const body = replyDrafts[topicId]?.trim();
    if (!body) return;
    try {
      const reply = await createCommunityReply(communityId, topicId, body);
      setCommunityReplies((previous) => ({ ...previous, [topicId]: [...(previous[topicId] || []), reply] }));
      setCommunityTopics((previous) => ({
        ...previous,
        [communityId]: (previous[communityId] || []).map((topic) => topic.id === topicId ? { ...topic, replyCount: topic.replyCount + 1 } : topic),
      }));
      setReplyDrafts((previous) => ({ ...previous, [topicId]: '' }));
    } catch { /* ignore */ }
  };

  const handleDeleteTopic = async (communityId: string, topicId: string) => {
    try {
      await deleteCommunityTopic(communityId, topicId);
      setCommunityTopics((previous) => ({ ...previous, [communityId]: (previous[communityId] || []).filter((topic) => topic.id !== topicId) }));
      setCommunities((previous) => previous.map((item) => item.id === communityId ? { ...item, topicCount: Math.max(0, item.topicCount - 1) } : item));
    } catch { /* ignore */ }
  };

  const handleDeleteReply = async (communityId: string, topicId: string, replyId: string) => {
    try {
      await deleteCommunityReply(communityId, topicId, replyId);
      setCommunityReplies((previous) => ({ ...previous, [topicId]: (previous[topicId] || []).filter((reply) => reply.id !== replyId) }));
      setCommunityTopics((previous) => ({
        ...previous,
        [communityId]: (previous[communityId] || []).map((topic) => topic.id === topicId ? { ...topic, replyCount: Math.max(0, topic.replyCount - 1) } : topic),
      }));
    } catch { /* ignore */ }
  };

  const handleVote = async (poll: Poll, optionId: string) => {
    if (poll.voted) return;
    try {
      await votePoll(poll.id, optionId);
      setPolls((previous) => previous.map((item) => item.id === poll.id ? {
        ...item,
        voted: true,
        options: item.options.map((option) => option.id === optionId ? { ...option, votes: option.votes + 1 } : option),
      } : item));
    } catch { /* Keep the feed usable if a vote is rejected. */ }
  };

  const toggleConnect = async (userId: string) => {
    try {
      if (connectedIds.has(userId)) {
        await removeConnection(userId);
        setConnections((prev) => prev.filter((c) => c.connectedToId !== userId));
      } else {
        const conn = await createConnection(userId);
        setConnections((prev) => [conn, ...prev]);
      }
    } catch {
      // ignore
    }
  };

  const handleToggleLike = async (postId: string) => {
    try {
      const { liked } = await togglePostLike(postId, user?.id);
      setPosts((prev) =>
        prev.map((p) =>
          p.id === postId
            ? { ...p, likedByMe: liked, likes: Math.max(0, p.likes + (liked ? 1 : -1)) }
            : p
        )
      );
    } catch {
      // ignore
    }
  };

  const handleSharePost = async (postId: string) => {
    try {
      const { shares } = await sharePost(postId);
      setPosts((prev) => prev.map((post) => post.id === postId ? { ...post, shares } : post));
    } catch {
      // Keep the feed usable if the count update fails.
    }
  };

  const toggleComments = async (postId: string) => {
    const isOpen = expandedComments[postId];
    setExpandedComments((prev) => ({ ...prev, [postId]: !isOpen }));
    if (!isOpen && !postComments[postId]) {
      try {
        const comments = await fetchPostComments(postId);
        setPostComments((prev) => ({ ...prev, [postId]: comments }));
      } catch {
        // ignore
      }
    }
  };

  const handleAddComment = async (postId: string) => {
    const text = commentDrafts[postId]?.trim();
    if (!text) return;
    try {
      const comment = await addPostComment(postId, {
        author: user?.name ?? 'StatChat User',
        role: user?.roles?.[0],
        org: user?.organizationId,
        text,
      });
      setPostComments((prev) => ({ ...prev, [postId]: [...(prev[postId] || []), comment] }));
      setPosts((prev) =>
        prev.map((p) => (p.id === postId ? { ...p, comments: p.comments + 1 } : p))
      );
      setCommentDrafts((prev) => ({ ...prev, [postId]: '' }));
    } catch {
      // ignore
    }
  };

  const renderHome = () => (
    <div>
      {/* Create post */}
      <div className={styles.createPostBox} style={{ background: bg, borderColor }}>
        <div className={styles.createPostRow}>
          <div className={styles.postAvatar} style={{ width: 40, height: 40, fontSize: 14 }}>
            {getAvatarText(user?.name ?? 'U')}
          </div>
          {showPostForm ? (
            <div style={{ flex: 1 }}>
              <textarea
                autoFocus
                value={postDraft}
                onChange={(e) => setPostDraft(e.target.value)}
                placeholder="Share an update, article, or question..."
                style={{
                  width: '100%',
                  boxSizing: 'border-box',
                  padding: '12px 16px',
                  borderRadius: 12,
                  border: `1px solid ${borderColor}`,
                  background: 'transparent',
                  color: textColor,
                  fontSize: 14,
                  fontFamily: 'inherit',
                  resize: 'vertical',
                  minHeight: 60,
                }}
              />
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 8 }}>
                <button
                  type="button"
                  onClick={() => { setShowPostForm(false); setPostDraft(''); }}
                  style={{ padding: '8px 16px', borderRadius: 999, border: `1px solid ${borderColor}`, background: 'transparent', color: textColor, cursor: 'pointer', fontSize: 13, fontWeight: 600 }}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handlePost}
                  disabled={!postDraft.trim()}
                  style={{ padding: '8px 20px', borderRadius: 999, border: 'none', background: '#165c92', color: '#fff', cursor: 'pointer', fontSize: 13, fontWeight: 700, opacity: postDraft.trim() ? 1 : 0.5 }}
                >
                  Post
                </button>
              </div>
            </div>
          ) : (
            <button
              type="button"
              className={styles.createPostInput}
              onClick={() => setShowPostForm(true)}
              style={{ color: textColor, textAlign: 'left' }}
            >
              Share an update, article, or question...
            </button>
          )}
        </div>
        {!showPostForm && (
          <div className={styles.createPostActions}>
            <button type="button" className={styles.createPostAction}>📷 Photo</button>
            <button type="button" className={styles.createPostAction}>🎥 Video</button>
            <button type="button" className={styles.createPostAction}>📄 Article</button>
            <button type="button" className={styles.createPostAction} onClick={() => setShowPollForm((value) => !value)}>📊 Poll</button>
          </div>
        )}
      </div>

      {showPollForm && (
        <div className={styles.postCard} style={{ background: bg, borderColor }}>
          <strong>Create a poll</strong>
          <input value={pollQuestion} onChange={(event) => setPollQuestion(event.target.value)} placeholder="Ask a question" style={{ width: '100%', boxSizing: 'border-box', marginTop: 10, padding: 10, borderRadius: 8, border: `1px solid ${borderColor}` }} />
          {pollOptions.map((option, index) => <input key={index} value={option} onChange={(event) => setPollOptions((previous) => previous.map((item, itemIndex) => itemIndex === index ? event.target.value : item))} placeholder={`Option ${index + 1}`} style={{ width: '100%', boxSizing: 'border-box', marginTop: 8, padding: 10, borderRadius: 8, border: `1px solid ${borderColor}` }} />)}
          <button type="button" onClick={() => setPollOptions((previous) => previous.length < 10 ? [...previous, ''] : previous)} style={{ marginTop: 8 }}>Add option</button>
          <button type="button" onClick={handleCreatePoll} style={{ marginTop: 10, marginLeft: 8, padding: '8px 14px', border: 0, borderRadius: 8, background: '#165c92', color: '#fff' }}>Publish poll</button>
        </div>
      )}

      {polls.map((poll) => {
        const totalVotes = poll.options.reduce((sum, option) => sum + option.votes, 0);
        return <div key={poll.id} className={styles.postCard} style={{ background: bg, borderColor }}><strong>{poll.question}</strong>{poll.options.map((option) => <button key={option.id} type="button" disabled={poll.voted} onClick={() => handleVote(poll, option.id)} style={{ display: 'block', width: '100%', marginTop: 8, padding: 10, textAlign: 'left', border: `1px solid ${borderColor}`, borderRadius: 8, background: 'transparent', color: textColor }}>{option.label} <span style={{ float: 'right', opacity: .7 }}>{option.votes}{totalVotes ? ` (${Math.round(option.votes / totalVotes * 100)}%)` : ''}</span></button>)}</div>;
      })}

      {/* Feed */}
      {posts.map((post) => (
        <div key={post.id} className={styles.postCard} style={{ background: bg, borderColor }}>
          <div className={styles.postHeader}>
            <div className={styles.postAvatar}>{getAvatarText(post.author)}</div>
            <div className={styles.postAuthorInfo}>
              <div className={styles.postAuthorName}>{post.author}</div>
              <div className={styles.postAuthorMeta}>{post.role} · {post.org}</div>
            </div>
            <span className={styles.postTime}>{post.time}</span>
          </div>
<div className={styles.postBody}>{post.text}</div>
          <div className={styles.postActions}>
            <button
              type="button"
              className={`${styles.postAction} ${post.likedByMe ? styles.postActionActive : ''}`}
              onClick={() => handleToggleLike(post.id)}
              style={post.likedByMe ? { color: '#165c92', fontWeight: 700 } : undefined}
            >
              {post.likedByMe ? '👍 Liked' : '👍 Like'} ({post.likes})
            </button>
            <button type="button" className={styles.postAction} onClick={() => toggleComments(post.id)}>
              💬 Comment ({post.comments})
            </button>
            <button type="button" className={styles.postAction} onClick={() => handleSharePost(post.id)}>
              ↗ Share ({post.shares})
            </button>
          </div>

          {expandedComments[post.id] && (
            <div className={styles.commentsSection} style={{ borderTop: `1px solid ${borderColor}`, paddingTop: 12, marginTop: 8 }}>
              {postComments[post.id]?.map((comment) => (
                <div key={comment.id} className={styles.commentRow} style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                  <div className={styles.postAvatar} style={{ width: 28, height: 28, fontSize: 11 }}>
                    {getAvatarText(comment.author)}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: 13, fontWeight: 600 }}>
                      {comment.author}
                      {comment.role && <span style={{ fontWeight: 400, opacity: 0.6 }}> · {comment.role}</span>}
                    </div>
                    <div style={{ fontSize: 13, opacity: 0.9 }}>{comment.text}</div>
                    <div style={{ fontSize: 11, opacity: 0.5, marginTop: 2 }}>
                      {new Date(comment.createdAt).toLocaleString()}
                    </div>
                  </div>
                </div>
              ))}
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                <div className={styles.postAvatar} style={{ width: 28, height: 28, fontSize: 11 }}>
                  {getAvatarText(user?.name ?? 'U')}
                </div>
                <input
                  type="text"
                  value={commentDrafts[post.id] || ''}
                  onChange={(e) => setCommentDrafts((prev) => ({ ...prev, [post.id]: e.target.value }))}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleAddComment(post.id);
                  }}
                  placeholder="Write a comment..."
                  style={{
                    flex: 1,
                    padding: '8px 12px',
                    borderRadius: 999,
                    border: `1px solid ${borderColor}`,
                    background: 'transparent',
                    color: textColor,
                    fontSize: 13,
                    outline: 'none',
                  }}
                />
                <button
                  type="button"
                  onClick={() => handleAddComment(post.id)}
                  disabled={!commentDrafts[post.id]?.trim()}
                  style={{
                    padding: '8px 16px',
                    borderRadius: 999,
                    border: 'none',
                    background: '#165c92',
                    color: '#fff',
                    cursor: 'pointer',
                    fontSize: 13,
                    fontWeight: 700,
                    opacity: commentDrafts[post.id]?.trim() ? 1 : 0.5,
                  }}
                >
                  Post
                </button>
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );

  const renderConnect = () => (
    <div>
      <h2 className={styles.sectionHeader}>Grow your network</h2>
      <p className={styles.sectionSubheader}>Connect with professionals across the StatGate enterprise</p>
      <div className={styles.networkGrid}>
        {allUsers.map((u) => {
          const connected = connectedIds.has(u.id);
          return (
            <div key={u.id} className={styles.networkCard} style={{ background: bg, borderColor }}>
              <div className={styles.networkAvatar}>{getAvatarText(u.name)}</div>
              <div className={styles.networkName}>{u.name}</div>
              <div className={styles.networkRole}>{u.roles?.[0] ?? 'Member'}</div>
              <div className={styles.networkOrg}>{u.organizationId}</div>
              <button
                type="button"
                className={`${styles.connectButton} ${connected ? styles.connectedButton : ''}`}
                onClick={() => toggleConnect(u.id)}
              >
                {connected ? '✓ Connected' : '+ Connect'}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );

  const renderNetwork = () => (
    <div>
      <h2 className={styles.sectionHeader}>My Network</h2>
      <div className={styles.statsRow}>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{connectedIds.size}</div>
          <div className={styles.statLabel}>Connections</div>
        </div>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>{allUsers.length}</div>
          <div className={styles.statLabel}>People you may know</div>
        </div>
        <div className={styles.statCard} style={{ background: bg, borderColor }}>
          <div className={styles.statValue}>12</div>
          <div className={styles.statLabel}>Pending requests</div>
        </div>
      </div>
      <h3 className={styles.sectionHeader} style={{ fontSize: 16 }}>Your connections</h3>
      <div className={styles.networkGrid}>
        {allUsers.filter((u) => connectedIds.has(u.id)).map((u) => (
          <div key={u.id} className={styles.networkCard} style={{ background: bg, borderColor }}>
            <div className={styles.networkAvatar}>{getAvatarText(u.name)}</div>
            <div className={styles.networkName}>{u.name}</div>
            <div className={styles.networkRole}>{u.roles?.[0] ?? 'Member'}</div>
            <div className={styles.networkOrg}>{u.organizationId}</div>
            <button
              type="button"
              className={`${styles.connectButton} ${styles.connectedButton}`}
              onClick={() => toggleConnect(u.id)}
            >
              ✓ Connected
            </button>
          </div>
        ))}
        {connectedIds.size === 0 && (
          <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: 40, opacity: 0.6 }}>
            No connections yet. Go to Connect to find professionals to connect with.
          </div>
        )}
      </div>
    </div>
  );

  const renderCommunities = () => {
    const topics = selectedCommunity ? communityTopics[selectedCommunity.id] || [] : [];
    const members = selectedCommunity ? communityMembers[selectedCommunity.id] || [] : [];
    const isCommunityOwner = selectedCommunity?.role === 'owner';
    const memberIds = new Set(members.map((member) => member.userId));
    const inviteCandidates = allUsers.filter((candidate) => !memberIds.has(candidate.id));
    return (
      <div>
        <h2 className={styles.sectionHeader}>Communities</h2>
        <p className={styles.sectionSubheader}>Tenant communities for practice groups, forums, and long-running discussion threads</p>
        <div className={styles.communityLayout}>
          <div className={styles.communityList}>
            <div className={styles.postCard} style={{ background: bg, borderColor, padding: 16 }}>
              <strong>Create community</strong>
              <input value={communityDraft.name} onChange={(event) => setCommunityDraft((previous) => ({ ...previous, name: event.target.value }))} placeholder="Community name" className={styles.communityInput} style={{ borderColor }} />
              <textarea value={communityDraft.description} onChange={(event) => setCommunityDraft((previous) => ({ ...previous, description: event.target.value }))} placeholder="Purpose or scope" className={styles.communityTextArea} style={{ borderColor }} />
              <select value={communityDraft.visibility} onChange={(event) => setCommunityDraft((previous) => ({ ...previous, visibility: event.target.value as 'public' | 'private' }))} className={styles.communityInput} style={{ borderColor }}>
                <option value="public">Public</option>
                <option value="private">Private</option>
              </select>
              <button type="button" className={styles.connectButton} onClick={handleCreateCommunity}>Create</button>
            </div>
            {communities.map((community) => (
              <button key={community.id} type="button" className={`${styles.communityItem} ${selectedCommunityId === community.id ? styles.communityItemActive : ''}`} onClick={() => { setSelectedCommunityId(community.id); if (community.joined && !communityTopics[community.id]) refreshCommunityTopics(community.id).catch(() => {}); if (community.joined && !communityMembers[community.id]) refreshCommunityMembers(community.id).catch(() => {}); }} style={{ borderColor }}>
                <span className={styles.communityName}>{community.name}</span>
                <span className={styles.communityMeta}>{community.visibility} · {community.memberCount} members · {community.topicCount} topics</span>
              </button>
            ))}
          </div>
          <div className={styles.communityDetail}>
            {selectedCommunity ? (
              <>
                <div className={styles.postCard} style={{ background: bg, borderColor, padding: 18 }}>
                  <div className={styles.communityHeaderRow}>
                    <div>
                      <h3 className={styles.sectionHeader} style={{ marginBottom: 6 }}>{selectedCommunity.name}</h3>
                      <p className={styles.sectionSubheader} style={{ marginBottom: 0 }}>{selectedCommunity.description || 'No description yet.'}</p>
                    </div>
                    <button type="button" className={`${styles.connectButton} ${selectedCommunity.joined ? styles.connectedButton : ''}`} onClick={() => handleCommunityMembership(selectedCommunity)}>
                      {selectedCommunity.joined ? selectedCommunity.role === 'owner' ? 'Owner' : 'Joined' : 'Join'}
                    </button>
                  </div>
                </div>
                {selectedCommunity.canPost && (
                  <div className={styles.postCard} style={{ background: bg, borderColor, padding: 16 }}>
                    <strong>Start a discussion</strong>
                    <input value={topicDraft.title} onChange={(event) => setTopicDraft((previous) => ({ ...previous, title: event.target.value }))} placeholder="Topic title" className={styles.communityInput} style={{ borderColor }} />
                    <textarea value={topicDraft.body} onChange={(event) => setTopicDraft((previous) => ({ ...previous, body: event.target.value }))} placeholder="What should the community discuss?" className={styles.communityTextArea} style={{ borderColor }} />
                    <button type="button" className={styles.connectButton} onClick={handleCreateTopic}>Post topic</button>
                  </div>
                )}
                {selectedCommunity.joined && members.length === 0 && (
                  <button type="button" className={styles.connectButton} onClick={() => refreshCommunityMembers(selectedCommunity.id).catch(() => {})}>Load members</button>
                )}
                {selectedCommunity.joined && members.length > 0 && (
                  <div className={styles.postCard} style={{ background: bg, borderColor, padding: 16 }}>
                    <div className={styles.communityHeaderRow}>
                      <strong>Members</strong>
                      {isCommunityOwner && (
                        <div className={styles.communityInviteRow}>
                          <select value={inviteTargetId} onChange={(event) => setInviteTargetId(event.target.value)} className={styles.communityInput} style={{ borderColor, marginTop: 0 }}>
                            <option value="">Select user</option>
                            {inviteCandidates.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.name}</option>)}
                          </select>
                          <button type="button" className={styles.connectButton} onClick={handleInviteMember} disabled={!inviteTargetId}>Add</button>
                        </div>
                      )}
                    </div>
                    <div className={styles.communityMembers}>
                      {members.map((member) => (
                        <div key={member.userId} className={styles.communityMemberRow}>
                          <div>
                            <div className={styles.postAuthorName}>{member.name}</div>
                            <div className={styles.postAuthorMeta}>{member.role} · {member.userRole || 'Member'} · {member.org || 'Tenant'}</div>
                          </div>
                          {isCommunityOwner && member.role !== 'owner' && (
                            <div className={styles.communityMemberActions}>
                              <button type="button" className={styles.linkButton} onClick={() => handleTransferOwner(member.userId)}>Make owner</button>
                              <button type="button" className={styles.linkButton} onClick={() => handleRemoveMember(member.userId)}>Remove</button>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {selectedCommunity.joined && topics.length === 0 && (
                  <button type="button" className={styles.connectButton} onClick={() => refreshCommunityTopics(selectedCommunity.id).catch(() => {})}>Load discussions</button>
                )}
                {topics.map((topic) => (
                  <div key={topic.id} className={styles.postCard} style={{ background: bg, borderColor, padding: 16 }}>
                    <div className={styles.postAuthorName}>{topic.title}</div>
                    <div className={styles.postAuthorMeta}>{topic.author} · {topic.role || 'Member'} · {new Date(topic.createdAt).toLocaleString()}</div>
                    <div className={styles.postBody} style={{ padding: '10px 0 0' }}>{topic.body}</div>
                    <div className={styles.communityTopicActions}>
                      <button type="button" className={styles.postAction} onClick={() => toggleTopicReplies(topic.communityId, topic.id)} style={{ justifyContent: 'flex-start', paddingLeft: 0 }}>
                        Replies ({topic.replyCount})
                      </button>
                      {(isCommunityOwner || topic.authorId === user?.id) && (
                        <button type="button" className={styles.linkButton} onClick={() => handleDeleteTopic(topic.communityId, topic.id)}>Delete topic</button>
                      )}
                    </div>
                    {expandedTopicId === topic.id && (
                      <div className={styles.commentsSection}>
                        {(communityReplies[topic.id] || []).map((reply) => (
                          <div key={reply.id} className={styles.commentRow}>
                            <strong>{reply.author}</strong>
                            <span style={{ opacity: 0.65 }}> · {new Date(reply.createdAt).toLocaleString()}</span>
                            <div>{reply.body}</div>
                            {(isCommunityOwner || reply.authorId === user?.id) && (
                              <button type="button" className={styles.linkButton} onClick={() => handleDeleteReply(topic.communityId, topic.id, reply.id)}>Delete reply</button>
                            )}
                          </div>
                        ))}
                        <div className={styles.communityReplyRow}>
                          <input value={replyDrafts[topic.id] || ''} onChange={(event) => setReplyDrafts((previous) => ({ ...previous, [topic.id]: event.target.value }))} placeholder="Reply..." className={styles.communityInput} style={{ borderColor }} />
                          <button type="button" className={styles.connectButton} onClick={() => handleCreateReply(topic.communityId, topic.id)}>Reply</button>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </>
            ) : (
              <div className={styles.postCard} style={{ background: bg, borderColor, padding: 24 }}>Create the first tenant community to begin.</div>
            )}
          </div>
        </div>
      </div>
    );
  };

  const renderOpportunities = () => (
    <div>
      <h2 className={styles.sectionHeader}>Opportunities</h2>
      <p className={styles.sectionSubheader}>Collaborations, research grants, training, conferences, and publications</p>
      {opportunities.map((opp) => (
        <div key={opp.id} className={styles.oppCard} style={{ background: bg, borderColor }}>
          <span className={styles.oppBadge} style={{ background: `${opp.badgeColor}20`, color: opp.badgeColor }}>
            {opp.badge}
          </span>
          <div className={styles.oppTitle}>{opp.title}</div>
          <div className={styles.oppDesc}>{opp.description}</div>
          <button
            type="button"
            className={styles.connectButton}
            style={{ marginTop: 12 }}
          >
            Learn More
          </button>
        </div>
      ))}
    </div>
  );

  const renderJobs = () => (
    <div>
      <h2 className={styles.sectionHeader}>Jobs</h2>
      <p className={styles.sectionSubheader}>Find your next role in the StatGate enterprise</p>
      {jobs.map((job) => (
        <div key={job.id} className={styles.jobCard} style={{ background: bg, borderColor }}>
          <div className={styles.jobIcon}>{job.icon}</div>
          <div className={styles.jobInfo}>
            <div className={styles.jobTitle}>{job.title}</div>
            <div className={styles.jobCompany}>{job.company}</div>
            <div className={styles.jobMeta}>
              <span>📍 {job.location}</span>
              <span>💼 {job.type}</span>
              <span>💰 {job.salary}</span>
            </div>
          </div>
          <button type="button" className={styles.jobApply}>Apply</button>
        </div>
      ))}
    </div>
  );

  const renderContent = () => {
    switch (activeSubView) {
      case 'home': return renderHome();
      case 'communities': return renderCommunities();
      case 'connect': return renderConnect();
      case 'network': return renderNetwork();
      case 'opportunities': return renderOpportunities();
      case 'jobs': return renderJobs();
      default: return renderHome();
    }
  };

  return (
    <section className={styles.collabShell} style={{ background: 'transparent', color: textColor }}>
      <div className={styles.collabScroll}>
        {renderContent()}
      </div>
    </section>
  );
}
