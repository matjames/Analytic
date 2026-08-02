import { useEffect, useState } from 'react';
import type { User } from '../types';
import {
  fetchKnowledgeExperts,
  fetchKnowledgeArticles,
  fetchKnowledgeIdeas,
  fetchKnowledgePosts,
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

type ExpertProfile = KnowledgeExpert;
type KnowledgeArticleItem = KnowledgeArticle;
type KnowledgeIdeaItem = KnowledgeIdea;

const categories = ['All', 'Statistics', 'Analysis', 'Research', 'Development', 'Data Science', 'Machine Learning', 'Public Health', 'Education'] as const;

export default function KnowledgePanel({ user, theme, isMobile }: Props) {
  const [activeCategory, setActiveCategory] = useState<string>('All');
  const [experts, setExperts] = useState<KnowledgeExpert[]>([]);
  const [articles, setArticles] = useState<KnowledgeArticleItem[]>([]);
  const [ideas, setIdeas] = useState<KnowledgeIdeaItem[]>([]);
  const [allPosts, setAllPosts] = useState<KnowledgePost[]>([]);
  const [loading, setLoading] = useState(true);

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
            >
              {cat}
            </button>
          ))}
        </div>

        {/* Hero banner */}
        <div className={styles.heroBanner} style={{ backgroundColor: isDark ? '#165c92' : '#165c92' }}>
          <div className={styles.heroTitle}>Where Knowledge Meets Professionals</div>
          <div className={styles.heroSubtitle}>Talented statisticians, analysts, researchers, and developers share their expertise. Learn from the best, contribute your own knowledge.</div>
        </div>

        {/* Featured experts */}
        <h2 className={styles.sectionHeader}>Featured Experts</h2>
        <div className={styles.expertGrid}>
          {filteredExperts.map((expert) => (
            <div key={expert.id} className={styles.expertCard} style={{ background: bg, borderColor }}>
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
              <button type="button" className={styles.followBtn}>Follow</button>
            </div>
          ))}
        </div>

        {/* Knowledge articles */}
        <h2 className={styles.sectionHeader}>Latest Knowledge Articles</h2>
        <div className={styles.articleList}>
          {filteredArticles.map((article) => (
            <div key={article.id} className={styles.articleCard} style={{ background: bg, borderColor }}>
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
                <button type="button" className={styles.readBtn}>Read Article</button>
              </div>
            </div>
          ))}
        </div>

        {/* Knowledge ideas */}
        <h2 className={styles.sectionHeader}>Knowledge Ideas</h2>
        <div className={styles.ideaList}>
          {filteredIdeas.map((idea) => (
            <div key={idea.id} className={styles.ideaCard} style={{ background: bg, borderColor }}>
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
                <button type="button" className={styles.voteBtn}>▲ Upvote</button>
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
    </section>
  );
}