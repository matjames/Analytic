import { useEffect, useState } from 'react';
import type { User } from '../types';
import {
  fetchWellnessPosts,
  createWellnessPost,
  fetchWellnessComments,
  addWellnessComment,
  toggleWellnessLike,
  toggleWellnessBookmark,
  shareWellnessPost,
  type WellnessPost,
  type WellnessComment,
} from '../api/client';
import styles from './WellnessPanel.module.css';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
}

const categories = ['For You', 'Mental Health', 'Mindfulness', 'Motivation', 'Development', 'Self-Care', 'Anxiety', 'Depression'] as const;

function getAvatarText(name: string) {
  const parts = name.trim().split(' ');
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

export default function WellnessPanel({ user, theme, isMobile }: Props) {
  const [activeCategory, setActiveCategory] = useState<string>('For You');
  const [posts, setPosts] = useState<WellnessPost[]>([]);
  const [loading, setLoading] = useState(true);
  const [draft, setDraft] = useState('');
  const [showPostForm, setShowPostForm] = useState(false);
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const [bookmarkedIds, setBookmarkedIds] = useState<Set<string>>(new Set());
  const [commentsByPost, setCommentsByPost] = useState<Record<string, WellnessComment[]>>({});
  const [openCommentsFor, setOpenCommentsFor] = useState<string | null>(null);
  const [commentDraft, setCommentDraft] = useState('');

  useEffect(() => {
    fetchWellnessPosts()
      .then((fetched) => {
        setPosts(fetched);
        setLikedIds(new Set(fetched.filter((post) => post.likedByMe).map((post) => post.id)));
        setBookmarkedIds(new Set(fetched.filter((post) => post.bookmarkedByMe).map((post) => post.id)));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const isDark = theme === 'dark';
  const bg = isDark ? '#0a2b45' : '#ffffff';
  const textColor = isDark ? '#e8eef4' : '#1a1a1a';
  const borderColor = isDark ? '#6b7280' : '#e5e7eb';

  const filteredPosts = activeCategory === 'For You'
    ? posts
    : posts.filter((p) => p.category === activeCategory);

  const updatePost = (id: string, update: Partial<WellnessPost>) => {
    setPosts((previous) => previous.map((post) => post.id === id ? { ...post, ...update } : post));
  };

  const toggleLike = async (id: string) => {
    try {
      const result = await toggleWellnessLike(id);
      updatePost(id, { likes: result.likes });
      setLikedIds((previous) => {
        const next = new Set(previous);
        if (result.liked) next.add(id);
        else next.delete(id);
        return next;
      });
    } catch {
      // Keep the current state when the server rejects the action.
    }
  };

  const toggleBookmark = async (id: string) => {
    try {
      const result = await toggleWellnessBookmark(id);
      updatePost(id, { bookmarks: result.bookmarks });
      setBookmarkedIds((previous) => {
        const next = new Set(previous);
        if (result.bookmarked) next.add(id);
        else next.delete(id);
        return next;
      });
    } catch {
      // Keep the current state when the server rejects the action.
    }
  };

  const handleShare = async (id: string) => {
    try {
      const result = await shareWellnessPost(id);
      updatePost(id, { shares: result.shares });
    } catch {
      // Keep the current state when the server rejects the action.
    }
  };

  const toggleComments = async (id: string) => {
    if (openCommentsFor === id) {
      setOpenCommentsFor(null);
      setCommentDraft('');
      return;
    }
    try {
      const loaded = await fetchWellnessComments(id);
      setCommentsByPost((previous) => ({ ...previous, [id]: loaded }));
      setOpenCommentsFor(id);
      setCommentDraft('');
    } catch {
      // Keep the feed usable when comments cannot be loaded.
    }
  };

  const handleComment = async (id: string) => {
    if (!commentDraft.trim()) return;
    try {
      const comment = await addWellnessComment(id, commentDraft.trim());
      setCommentsByPost((previous) => ({ ...previous, [id]: [...(previous[id] ?? []), comment] }));
      const currentPost = posts.find((post) => post.id === id);
      updatePost(id, { comments: (currentPost?.comments ?? 0) + 1 });
      setCommentDraft('');
    } catch {
      // Keep the draft available when the server rejects the comment.
    }
  };

  const handlePost = async () => {
    if (!draft.trim()) return;
    try {
      const newPost = await createWellnessPost({
        author: user?.name ?? 'StatChat User',
        handle: '@statchatuser',
        avatar: '🧠',
        category: activeCategory === 'For You' ? 'Mental Health' : activeCategory,
        time: 'now',
        text: draft.trim(),
        likes: 0,
        comments: 0,
        shares: 0,
        bookmarks: 0,
        tags: ['#Wellness'],
      });
      setPosts((prev) => [newPost, ...prev]);
      setDraft('');
      setShowPostForm(false);
    } catch {
      // ignore
    }
  };

  return (
    <section className={styles.wellnessShell} style={{ color: textColor }}>
      <div className={styles.wellnessScroll}>
        <div className={styles.categoryRow}>
          {categories.map((cat) => (
            <button
              key={cat}
              type="button"
              className={`${styles.categoryPill} ${activeCategory === cat ? styles.categoryActive : ''}`}
              onClick={() => setActiveCategory(cat)}
            >
              {cat}
            </button>
          ))}
        </div>

        <div className={styles.createPostBox} style={{ background: bg, borderColor }}>
          <div className={styles.createPostRow}>
            <div className={styles.postAvatar} style={{ width: 40, height: 40, fontSize: 18 }}>
              {user?.name ? getAvatarText(user.name) : '🧠'}
            </div>
            {showPostForm ? (
              <div style={{ flex: 1 }}>
                <textarea
                  autoFocus
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  placeholder="Share a wellness thought, tip, or encouragement..."
                  className={styles.postTextarea}
                  style={{ border: `1px solid ${borderColor}`, background: 'transparent', color: textColor }}
                />
                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 8 }}>
                  <button
                    type="button"
                    onClick={() => { setShowPostForm(false); setDraft(''); }}
                    className={styles.cancelBtn}
                    style={{ border: `1px solid ${borderColor}`, color: textColor }}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handlePost}
                    disabled={!draft.trim()}
                    className={styles.postBtn}
                    style={{ opacity: draft.trim() ? 1 : 0.5 }}
                  >
                    Share
                  </button>
                </div>
              </div>
            ) : (
              <button
                type="button"
                className={styles.createPostInput}
                onClick={() => setShowPostForm(true)}
                style={{ color: textColor }}
              >
                Share a wellness thought, tip, or encouragement...
              </button>
            )}
          </div>
        </div>

        {filteredPosts.map((post) => {
          const liked = likedIds.has(post.id);
          const bookmarked = bookmarkedIds.has(post.id);
          return (
            <div key={post.id} className={styles.postCard} style={{ background: bg, borderColor }}>
              <div className={styles.postHeader}>
                <div className={styles.postAvatar}>{post.avatar}</div>
                <div className={styles.postAuthorInfo}>
                  <div className={styles.postAuthorName}>{post.author}</div>
                  <div className={styles.postHandle}>{post.handle} · {post.time}</div>
                </div>
                <span className={styles.categoryBadge}>{post.category}</span>
              </div>
              <div className={styles.postBody}>{post.text}</div>
              {post.tags.length > 0 && (
                <div className={styles.tagsRow}>
                  {post.tags.map((tag) => (
                    <span key={tag} className={styles.tag}>{tag}</span>
                  ))}
                </div>
              )}
              <div className={styles.postActions}>
                <button
                  type="button"
                  className={`${styles.actionBtn} ${liked ? styles.likedBtn : ''}`}
                  onClick={() => toggleLike(post.id)}
                >
                  {liked ? '❤️' : '🤍'} {post.likes}
                </button>
                <button type="button" className={styles.actionBtn} onClick={() => toggleComments(post.id)}>
                  💬 {post.comments}
                </button>
                <button type="button" className={styles.actionBtn} onClick={() => handleShare(post.id)}>
                  ↗ {post.shares}
                </button>
                <button
                  type="button"
                  className={`${styles.actionBtn} ${bookmarked ? styles.bookmarkedBtn : ''}`}
                  onClick={() => toggleBookmark(post.id)}
                >
                  {bookmarked ? '🔖' : '📑'} {post.bookmarks}
                </button>
              </div>
              {openCommentsFor === post.id && (
                <div className={styles.commentsPanel}>
                  {(commentsByPost[post.id] ?? []).map((comment) => (
                    <div key={comment.id} className={styles.commentRow}>
                      <strong>{comment.author}</strong>
                      <span>{comment.text}</span>
                    </div>
                  ))}
                  <div className={styles.commentComposer}>
                    <input
                      value={commentDraft}
                      onChange={(event) => setCommentDraft(event.target.value)}
                      placeholder="Add an encouraging comment..."
                      aria-label="Add a wellness comment"
                    />
                    <button type="button" onClick={() => handleComment(post.id)} disabled={!commentDraft.trim()}>Send</button>
                  </div>
                </div>
              )}
            </div>
          );
        })}

        {filteredPosts.length === 0 && (
          <div className={styles.emptyState}>
            No posts in this category yet. Be the first to share!
          </div>
        )}
      </div>
    </section>
  );
}
