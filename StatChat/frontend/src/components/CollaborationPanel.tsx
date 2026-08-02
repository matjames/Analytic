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
  type Post,
  type Connection,
  type Opportunity,
  type Job,
} from '../api/client';
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
  }, []);

  useEffect(() => {
    if ((activeSubView === 'connect' || activeSubView === 'network') && allUsers.length === 0) {
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
            <button type="button" className={styles.createPostAction}>📊 Poll</button>
          </div>
        )}
      </div>

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
            <button type="button" className={styles.postAction}>👍 Like ({post.likes})</button>
            <button type="button" className={styles.postAction}>💬 Comment ({post.comments})</button>
            <button type="button" className={styles.postAction}>↗ Share ({post.shares})</button>
          </div>
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