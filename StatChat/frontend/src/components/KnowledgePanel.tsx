import { useEffect, useState } from 'react';
import type { User } from '../types';
import {
  fetchKnowledgeExperts,
  fetchKnowledgeArticles,
  fetchKnowledgeIdeas,
  fetchKnowledgePosts,
  createKnowledgePost,
  upvoteKnowledgeIdea,
  followKnowledgeExpert,
  type KnowledgeExpert,
  type KnowledgeArticle,
  type KnowledgeIdea,
  type KnowledgePost,
} from '../api/client';
import styles from './KnowledgePanel.module.css';

interface Props {
  user: User | null;
  theme: 'light' | 'dark';
  isMobile: boolean;
}

const categories = ['All', 'Statistics', 'Analysis', 'Research', 'Development', 'Data Science', 'Machine Learning', 'Public Health', 'Education'] as const;

export default function KnowledgePanel({ user, theme, isMobile }: Props) {
  const [activeCategory, setActiveCategory] = useState<string>('All');
  const [experts, setExperts] = useState<KnowledgeExpert[]>([]);
  const [articles, setArticles] = useState<KnowledgeArticle[]>([]);
  const [ideas, setIdeas] = useState<KnowledgeIdea[]>([]);
  const [allPosts, setAllPosts] = useState<KnowledgePost[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedArticle, setSelectedArticle] = useState<KnowledgeArticle | null>(null);
  const [showCreatePost, setShowCreatePost] = useState(false);
  const [newPostTitle, setNewPostTitle] = useState('');
  const [newPostCategory, setNewPostCategory] = useState('Statistics');
  const [newPostContent, setNewPostContent] = useState('');
  const [actionNote, setActionNote] = useState<string | null>(null);

  const isDark = theme === 'dark';
  const bg = isDark ? '#0a2b45' : '#ffffff';
  const textColor = isDark ? '#e8eef4' : '#1a1a1a';
  const borderColor = isDark ? '#6b7280' : '#e5e7eb';

  useEffect(() => {
    Promise.all([
      fetchKnowledgeExperts().then(setExperts),
      fetchKnowledgeArticles().then(setArticles),
      fetchKnowledgeIdeas().then(setIdeas),
      fetchKnowledgePosts().then(setAllPosts),
    ])
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const filteredExperts = activeCategory === 'All'
    ? experts
    : experts.filter((e) => e.specialties.some((s) => s.toLowerCase().includes(activeCategory.toLowerCase())));

  const filteredArticles = activeCategory === 'All'
    ? articles
    : articles.filter((a) => a.category === activeCategory);

  const filteredIdeas = activeCategory === 'All'
    ? ideas
    : ideas.filter((i) => i.category === activeCategory);

  const handleFollow = async (expertId: string) => {
    try {
      const result = await followKnowledgeExpert(expertId);
      setExperts((prev) => prev.map((e) => e.id === expertId ? { ...e, followers: result.followers } : e));
      setActionNote('✅ Following');
    } catch {
      setActionNote('⚠️ Could not follow');
    }
  };

  const handleUpvote = async (ideaId: string) => {
    try {
      const result = await upvoteKnowledgeIdea(ideaId);
      setIdeas((prev) => prev.map((i) => i.id === ideaId ? { ...i, votes: result.votes } : i));
      setActionNote('▲ Upvoted');
    } catch {
      setActionNote('⚠️ Could not upvote');
    }
  };

  const handleCreatePost = async () => {
    const title = newPostTitle.trim();
    const content = newPostContent.trim();
    if (!title || !content) return;
    try {
      const post = await createKnowledgePost({
        title,
        author: user?.name ?? 'StatChat User',
        category: newPostCategory,
        content,
        createdBy: user?.id ?? 'user-001',
      });
      setAllPosts((prev) => [post, ...prev]);
      setShowCreatePost(false);
      setNewPostTitle('');
      setNewPostContent('');
      setActionNote('✅ Knowledge post created');
    } catch {
      setActionNote('⚠️ Could not create post');
    }
  };

  return (
    <section className={styles.shell} style={{ color: textColor }}>
      <div className={styles.scroll}>
        {/* Category tabs */}
        <div className={styles.categoryRow}>
          {categories.map((cat) => (
            <button
              key={cat}
              type="button"
              className={`${styles.categoryPill} ${activeCategory === cat ? styles.categoryActive : ''}`}
              onClick={() => setActiveCategory(cat)}
              style={activeCategory === cat ? { background: '#0ea5e9', color: '#fff', boxShadow: '0 12px 30px rgba(14, 165, 233, 0.2)' } : undefined}
            >
              {cat}
            </button>
          ))}
        </div>

        {/* Hero banner */}
        <div
          className={styles.heroBanner}
          style={{
            background: 'linear-gradient(135deg, #1d4ed8 0%, #0ea5e9 55%, #38bdf8 100%)',
            boxShadow: '0 28px 80px rgba(14, 165, 233, 0.18)',
            border: '1px solid rgba(255,255,255,0.18)',
          }}
        >
          <div className={styles.heroTitle} style={{ textShadow: '0 20px 40px rgba(0,0,0,0.18)' }}>
            Where Knowledge Meets Professionals
          </div>
          <div className={styles.heroSubtitle} style={{ maxWidth: 680, letterSpacing: '0.01em' }}>
            Talented statisticians, analysts, researchers, and developers share their expertise. Learn from the best, contribute your own knowledge.
          </div>
        </div>

        {/* Create post toggle */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
          <button
            type="button"
            className={styles.followBtn}
            onClick={() => setShowCreatePost((prev) => !prev)}
          >
            {showCreatePost ? 'Cancel' : '+ Share Knowledge'}
          </button>
        </div>

        {/* Create post form */}
        {showCreatePost && (
          <div className={styles.articleCard} style={{ background: bg, borderColor, padding: 16, marginBottom: 16 }}>
            <input
              type="text"
              placeholder="Post title"
              value={newPostTitle}
              onChange={(e) => setNewPostTitle(e.target.value)}
              style={{ width: '100%', padding: '8px 12px', marginBottom: 8, borderRadius: 8, border: `1px solid ${borderColor}`, background: isDark ? '#0f3f5f' : '#f0f7fb', color: textColor, boxSizing: 'border-box' }}
            />
            <select
              value={newPostCategory}
              onChange={(e) => setNewPostCategory(e.target.value)}
              style={{ width: '100%', padding: '8px 12px', marginBottom: 8, borderRadius: 8, border: `1px solid ${borderColor}`, background: isDark ? '#0f3f5f' : '#f0f7fb', color: textColor, boxSizing: 'border-box' }}
            >
              {categories.filter((c) => c !== 'All').map((cat) => (
                <option key={cat} value={cat}>{cat}</option>
              ))}
            </select>
            <textarea
              placeholder="Share your knowledge..."
              value={newPostContent}
              onChange={(e) => setNewPostContent(e.target.value)}
              rows={4}
              style={{ width: '100%', padding: '8px 12px', marginBottom: 8, borderRadius: 8, border: `1px solid ${borderColor}`, background: isDark ? '#0f3f5f' : '#f0f7fb', color: textColor, boxSizing: 'border-box', resize: 'vertical' }}
            />
            <button
              type="button"
              className={styles.followBtn}
              onClick={handleCreatePost}
              disabled={!newPostTitle.trim() || !newPostContent.trim()}
            >
              Publish
            </button>
          </div>
        )}

        {/* Action note */}
        {actionNote && (
          <div style={{ padding: '8px 12px', marginBottom: 8, fontSize: 13, opacity: 0.8 }}>{actionNote}</div>
        )}

        {/* Featured experts */}
        <h2 className={styles.sectionHeader}>Featured Experts</h2>
        <div className={styles.expertGrid}>
          {filteredExperts.map((expert) => (
            <div
              key={expert.id}
              className={styles.expertCard}
              style={{
                background: isDark ? 'rgba(10, 40, 70, 0.92)' : 'linear-gradient(180deg, #ffffff 0%, #eef7ff 100%)',
                borderColor,
                boxShadow: '0 22px 42px rgba(14, 165, 233, 0.14)',
              }}
            >
              <div className={styles.expertAvatar}>{expert.avatar}</div>
              <div className={styles.expertName}>{expert.name}</div>
              <div className={styles.expertRole}>{expert.role}</div>
              <div className={styles.expertOrg}>{expert.org}</div>
              <div className={styles.expertTags}>
                {expert.specialties.map((s) => (
                  <span key={s} className={styles.expertTag}>{s}</span>
                ))}
              </div>
              <div className={styles.expertStats}>
                <span>👥 {expert.followers.toLocaleString()}</span>
                <span>📄 {expert.articles}</span>
                <span>⭐ {expert.rating}</span>
              </div>
              <button type="button" className={styles.followBtn} onClick={() => handleFollow(expert.id)}>Follow</button>
            </div>
          ))}
        </div>

        {/* Knowledge articles */}
        <h2 className={styles.sectionHeader}>Latest Knowledge Articles</h2>
        <div className={styles.articleList}>
          {filteredArticles.map((article) => (
            <div
              key={article.id}
              className={styles.articleCard}
              style={{
                background: isDark ? 'rgba(10, 40, 70, 0.92)' : '#fbfdff',
                borderColor,
                boxShadow: '0 18px 40px rgba(22, 92, 146, 0.1)',
                borderRadius: 22,
              }}
            >
              <h3 className={styles.articleTitle}>{article.title}</h3>
              <div className={styles.articleMeta}>
                <span>👤 {article.author}</span>
                <span className={styles.articleCategory}>{article.category}</span>
                <span>⏱ {article.readTime}</span>
              </div>
              <p className={styles.articleExcerpt}>{article.excerpt}</p>
              <div className={styles.articleActions}>
                <span>❤️ {article.likes}</span>
                <span>👁 {article.views.toLocaleString()}</span>
                <span>{article.published}</span>
                <button type="button" className={styles.readBtn} onClick={() => setSelectedArticle(article)}>Read Article</button>
              </div>
            </div>
          ))}
        </div>

        {/* Knowledge ideas */}
        <h2 className={styles.sectionHeader}>Knowledge Ideas</h2>
        <div className={styles.ideaList}>
          {filteredIdeas.map((idea) => (
            <div
              key={idea.id}
              className={styles.ideaCard}
              style={{
                background: isDark ? 'rgba(10, 40, 70, 0.92)' : '#f6fbff',
                borderColor,
                boxShadow: '0 18px 38px rgba(22, 92, 146, 0.1)',
                borderLeft: '4px solid #0ea5e9',
              }}
            >
              <div className={styles.ideaVotes}>
                <span className={styles.voteCount}>{idea.votes}</span>
                <span className={styles.voteLabel}>votes</span>
              </div>
              <div className={styles.ideaContent}>
                <h4 className={styles.ideaTitle}>{idea.title}</h4>
                <div className={styles.ideaMeta}>
                  <span>👤 {idea.author}</span>
                  <span className={styles.ideaCategory}>{idea.category}</span>
                  <span className={`${styles.ideaStatus} ${styles[idea.status as keyof typeof styles]}`}>
                    {idea.status === 'discussing' ? '💭 Discussing' : idea.status === 'developing' ? '🔧 Developing' : '🚀 Launched'}
                  </span>
                </div>
                <p className={styles.ideaDescription}>{idea.description}</p>
                <button type="button" className={styles.voteBtn} onClick={() => handleUpvote(idea.id)}>▲ Upvote</button>
              </div>
            </div>
          ))}
        </div>

        {/* User-created knowledge posts */}
        {allPosts.length > 0 && (
          <>
            <h2 className={styles.sectionHeader}>Community Posts</h2>
            <div className={styles.articleList}>
              {allPosts.map((post) => (
                <div key={post.id} className={styles.articleCard} style={{ background: bg, borderColor }}>
                  <h3 className={styles.articleTitle}>{post.title}</h3>
                  <div className={styles.articleMeta}>
                    <span>👤 {post.author}</span>
                    <span className={styles.articleCategory}>{post.category}</span>
                  </div>
                  <p className={styles.articleExcerpt}>{post.content}</p>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Article detail modal */}
      {selectedArticle && (
        <div
          style={{ position: 'fixed', inset: 0, zIndex: 150, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24 }}
          onClick={() => setSelectedArticle(null)}
        >
          <div
            style={{ background: isDark ? '#0a2b45' : '#ffffff', borderRadius: 16, padding: 24, maxWidth: 640, width: '100%', color: textColor, border: `1px solid ${borderColor}` }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
              <h2 style={{ margin: 0, fontSize: 20 }}>{selectedArticle.title}</h2>
              <button type="button" onClick={() => setSelectedArticle(null)} style={{ background: 'none', border: 'none', color: textColor, fontSize: 20, cursor: 'pointer' }}>✕</button>
            </div>
            <div style={{ display: 'flex', gap: 12, marginBottom: 12, fontSize: 13, opacity: 0.7 }}>
              <span>👤 {selectedArticle.author}</span>
              <span>📂 {selectedArticle.category}</span>
              <span>⏱ {selectedArticle.readTime}</span>
              <span>❤️ {selectedArticle.likes}</span>
              <span>👁 {selectedArticle.views.toLocaleString()}</span>
            </div>
            <p style={{ lineHeight: 1.6, fontSize: 15 }}>{selectedArticle.excerpt}</p>
            <div style={{ marginTop: 16, padding: 12, background: isDark ? '#0f3f5f' : '#f0f7fb', borderRadius: 8, fontSize: 13, opacity: 0.7 }}>
              📖 This is a preview excerpt. Full article content will be available in the next phase.
            </div>
          </div>
        </div>
      )}
    </section>
  );
}