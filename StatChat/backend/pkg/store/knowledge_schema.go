package store

import "context"

func ensureKnowledgeSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
ALTER TABLE knowledge_experts ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS content TEXT NOT NULL DEFAULT '';
ALTER TABLE knowledge_ideas ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE knowledge_ideas ADD COLUMN IF NOT EXISTS author_id TEXT NOT NULL DEFAULT '';
ALTER TABLE knowledge_posts ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
ALTER TABLE knowledge_posts ADD COLUMN IF NOT EXISTS author_id TEXT NOT NULL DEFAULT '';
ALTER TABLE knowledge_posts ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT '';
ALTER TABLE knowledge_posts ADD COLUMN IF NOT EXISTS org TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS knowledge_experts_tenant_followers_idx ON knowledge_experts (tenant_id, followers DESC);
CREATE INDEX IF NOT EXISTS knowledge_articles_tenant_likes_idx ON knowledge_articles (tenant_id, likes DESC);
CREATE INDEX IF NOT EXISTS knowledge_ideas_tenant_votes_idx ON knowledge_ideas (tenant_id, votes DESC);
CREATE INDEX IF NOT EXISTS knowledge_posts_tenant_created_idx ON knowledge_posts (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS knowledge_expert_follows (
  expert_id TEXT NOT NULL REFERENCES knowledge_experts(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (expert_id, user_id)
);
CREATE INDEX IF NOT EXISTS knowledge_expert_follows_tenant_user_idx ON knowledge_expert_follows (tenant_id, user_id);

CREATE TABLE IF NOT EXISTS knowledge_idea_votes (
  idea_id TEXT NOT NULL REFERENCES knowledge_ideas(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (idea_id, user_id)
);
CREATE INDEX IF NOT EXISTS knowledge_idea_votes_tenant_user_idx ON knowledge_idea_votes (tenant_id, user_id);

UPDATE knowledge_articles SET content = excerpt WHERE content = '';
`)
	return err
}
