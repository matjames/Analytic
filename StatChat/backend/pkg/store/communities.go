package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"statchat/pkg/model"
)

var ErrCommunityForbidden = errors.New("community action forbidden")

func ensureCommunitiesSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS communities (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL CHECK (visibility IN ('public','private')),
  created_by TEXT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS communities_tenant_name_uidx ON communities (tenant_id, lower(name));
CREATE INDEX IF NOT EXISTS communities_tenant_created_idx ON communities (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community_members (
  community_id TEXT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('owner','member')),
  joined_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (community_id, user_id)
);
CREATE INDEX IF NOT EXISTS community_members_user_idx ON community_members (tenant_id, user_id);

CREATE TABLE IF NOT EXISTS community_topics (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  community_id TEXT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  author_id TEXT NOT NULL REFERENCES users(id),
  author_name TEXT NOT NULL,
  author_role TEXT NOT NULL DEFAULT '',
  author_org TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS community_topics_community_created_idx ON community_topics (tenant_id, community_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community_replies (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  community_id TEXT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
  topic_id TEXT NOT NULL REFERENCES community_topics(id) ON DELETE CASCADE,
  author_id TEXT NOT NULL REFERENCES users(id),
  author_name TEXT NOT NULL,
  author_role TEXT NOT NULL DEFAULT '',
  author_org TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS community_replies_topic_created_idx ON community_replies (tenant_id, topic_id, created_at);
`)
	return err
}

func CreateCommunity(community model.Community) (model.Community, error) {
	now := time.Now().UTC()
	community.ID = uuid.NewString()
	community.TenantID = normalizedTenantID(community.TenantID)
	community.Name = strings.TrimSpace(community.Name)
	community.Description = strings.TrimSpace(community.Description)
	community.Visibility = strings.TrimSpace(strings.ToLower(community.Visibility))
	community.CreatedAt = now
	community.MemberCount = 1
	community.Joined = true
	community.Role = "owner"
	community.CanPost = true

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return community, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO communities (id,tenant_id,name,description,visibility,created_by,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
		community.ID, community.TenantID, community.Name, community.Description, community.Visibility, community.CreatedBy, now); err != nil {
		return community, err
	}
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO community_members (community_id,tenant_id,user_id,role,joined_at) VALUES ($1,$2,$3,'owner',$4)`,
		community.ID, community.TenantID, community.CreatedBy, now); err != nil {
		return community, err
	}
	return community, tx.Commit()
}

func GetCommunities(tenantID, userID string) ([]model.Community, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT c.id,c.tenant_id,c.name,c.description,c.visibility,c.created_by,c.created_at,
       COUNT(DISTINCT cm_all.user_id) AS member_count,
       COUNT(DISTINCT t.id) AS topic_count,
       COALESCE(cm.role,'') AS role,
       MAX(COALESCE(t.updated_at,c.updated_at)) AS latest_post_at
FROM communities c
LEFT JOIN community_members cm ON cm.community_id=c.id AND cm.user_id=$2
LEFT JOIN community_members cm_all ON cm_all.community_id=c.id
LEFT JOIN community_topics t ON t.community_id=c.id
WHERE c.tenant_id=$1 AND (c.visibility='public' OR cm.user_id IS NOT NULL)
GROUP BY c.id,c.tenant_id,c.name,c.description,c.visibility,c.created_by,c.created_at,cm.role
ORDER BY latest_post_at DESC`, normalizedTenantID(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	communities := []model.Community{}
	for rows.Next() {
		var community model.Community
		if err := rows.Scan(&community.ID, &community.TenantID, &community.Name, &community.Description, &community.Visibility, &community.CreatedBy, &community.CreatedAt, &community.MemberCount, &community.TopicCount, &community.Role, &community.LatestPostAt); err != nil {
			return nil, err
		}
		community.Joined = community.Role != ""
		community.CanPost = community.Joined
		communities = append(communities, community)
	}
	return communities, rows.Err()
}

func JoinCommunity(communityID, tenantID, userID string) (model.Community, error) {
	tenantID = normalizedTenantID(tenantID)
	var visibility string
	if err := db.QueryRowContext(context.Background(), `SELECT visibility FROM communities WHERE id=$1 AND tenant_id=$2`, communityID, tenantID).Scan(&visibility); err != nil {
		return model.Community{}, err
	}
	if visibility != "public" {
		return model.Community{}, sql.ErrNoRows
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO community_members (community_id,tenant_id,user_id,role,joined_at) VALUES ($1,$2,$3,'member',$4) ON CONFLICT DO NOTHING`,
		communityID, tenantID, userID, time.Now().UTC())
	if err != nil {
		return model.Community{}, err
	}
	return GetCommunity(communityID, tenantID, userID)
}

func LeaveCommunity(communityID, tenantID, userID string) error {
	result, err := db.ExecContext(context.Background(), `DELETE FROM community_members WHERE community_id=$1 AND tenant_id=$2 AND user_id=$3 AND role <> 'owner'`, communityID, normalizedTenantID(tenantID), userID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCommunity(communityID, tenantID, userID string) (model.Community, error) {
	communities, err := GetCommunities(tenantID, userID)
	if err != nil {
		return model.Community{}, err
	}
	for _, community := range communities {
		if community.ID == communityID {
			return community, nil
		}
	}
	return model.Community{}, sql.ErrNoRows
}

func CommunityMembership(communityID, tenantID, userID string) (string, error) {
	var role string
	err := db.QueryRowContext(context.Background(), `SELECT role FROM community_members WHERE community_id=$1 AND tenant_id=$2 AND user_id=$3`, communityID, normalizedTenantID(tenantID), userID).Scan(&role)
	return role, err
}

func GetCommunityMembers(communityID, tenantID, userID string) ([]model.CommunityMember, error) {
	if _, err := CommunityMembership(communityID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `
SELECT cm.community_id,cm.user_id,u.name,cm.role,u.roles,u.organization_id,cm.joined_at
FROM community_members cm
JOIN users u ON u.id=cm.user_id
WHERE cm.community_id=$1 AND cm.tenant_id=$2
ORDER BY CASE cm.role WHEN 'owner' THEN 0 ELSE 1 END, u.name`, communityID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []model.CommunityMember{}
	for rows.Next() {
		var member model.CommunityMember
		var rolesJSON []byte
		if err := rows.Scan(&member.CommunityID, &member.UserID, &member.Name, &member.Role, &rolesJSON, &member.Org, &member.JoinedAt); err != nil {
			return nil, err
		}
		roles := []string{}
		_ = json.Unmarshal(rolesJSON, &roles)
		if len(roles) > 0 {
			member.UserRole = roles[0]
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func AddCommunityMember(communityID, tenantID, actorID, targetUserID string) (model.CommunityMember, error) {
	if role, err := CommunityMembership(communityID, tenantID, actorID); err != nil || role != "owner" {
		return model.CommunityMember{}, ErrCommunityForbidden
	}
	tenantID = normalizedTenantID(tenantID)
	var target model.User
	var rolesJSON []byte
	if err := db.QueryRowContext(context.Background(), `SELECT id,name,roles,organization_id FROM users WHERE id=$1 AND organization_id=$2`, targetUserID, tenantID).Scan(&target.ID, &target.Name, &rolesJSON, &target.OrganizationID); err != nil {
		return model.CommunityMember{}, err
	}
	joinedAt := time.Now().UTC()
	if _, err := db.ExecContext(context.Background(), `INSERT INTO community_members (community_id,tenant_id,user_id,role,joined_at) VALUES ($1,$2,$3,'member',$4) ON CONFLICT (community_id,user_id) DO UPDATE SET role=community_members.role`,
		communityID, tenantID, targetUserID, joinedAt); err != nil {
		return model.CommunityMember{}, err
	}
	userRoles := []string{}
	_ = json.Unmarshal(rolesJSON, &userRoles)
	member := model.CommunityMember{CommunityID: communityID, UserID: target.ID, Name: target.Name, Role: "member", Org: target.OrganizationID, JoinedAt: joinedAt}
	if len(userRoles) > 0 {
		member.UserRole = userRoles[0]
	}
	return member, nil
}

func RemoveCommunityMember(communityID, tenantID, actorID, targetUserID string) error {
	if role, err := CommunityMembership(communityID, tenantID, actorID); err != nil || role != "owner" {
		return ErrCommunityForbidden
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM community_members WHERE community_id=$1 AND tenant_id=$2 AND user_id=$3 AND role <> 'owner'`, communityID, normalizedTenantID(tenantID), targetUserID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func TransferCommunityOwnership(communityID, tenantID, actorID, targetUserID string) error {
	if actorID == targetUserID {
		return sql.ErrNoRows
	}
	if role, err := CommunityMembership(communityID, tenantID, actorID); err != nil || role != "owner" {
		return ErrCommunityForbidden
	}
	if targetRole, err := CommunityMembership(communityID, tenantID, targetUserID); err != nil || targetRole == "" {
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `UPDATE community_members SET role='member' WHERE community_id=$1 AND tenant_id=$2 AND user_id=$3 AND role='owner'`, communityID, normalizedTenantID(tenantID), actorID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(context.Background(), `UPDATE community_members SET role='owner' WHERE community_id=$1 AND tenant_id=$2 AND user_id=$3`, communityID, normalizedTenantID(tenantID), targetUserID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(context.Background(), `UPDATE communities SET created_by=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4`, targetUserID, time.Now().UTC(), communityID, normalizedTenantID(tenantID)); err != nil {
		return err
	}
	return tx.Commit()
}

func GetCommunityTopics(communityID, tenantID, userID string) ([]model.CommunityTopic, error) {
	if _, err := CommunityMembership(communityID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `
SELECT t.id,t.tenant_id,t.community_id,t.title,t.body,t.author_id,t.author_name,t.author_role,t.author_org,t.created_at,t.updated_at,
       COUNT(r.id) AS reply_count
FROM community_topics t
LEFT JOIN community_replies r ON r.topic_id=t.id
WHERE t.community_id=$1 AND t.tenant_id=$2
GROUP BY t.id,t.tenant_id,t.community_id,t.title,t.body,t.author_id,t.author_name,t.author_role,t.author_org,t.created_at,t.updated_at
ORDER BY t.updated_at DESC`, communityID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	topics := []model.CommunityTopic{}
	for rows.Next() {
		var topic model.CommunityTopic
		if err := rows.Scan(&topic.ID, &topic.TenantID, &topic.CommunityID, &topic.Title, &topic.Body, &topic.AuthorID, &topic.Author, &topic.Role, &topic.Org, &topic.CreatedAt, &topic.UpdatedAt, &topic.ReplyCount); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

func CreateCommunityTopic(topic model.CommunityTopic) (model.CommunityTopic, error) {
	if _, err := CommunityMembership(topic.CommunityID, topic.TenantID, topic.AuthorID); err != nil {
		return topic, err
	}
	now := time.Now().UTC()
	topic.ID = uuid.NewString()
	topic.TenantID = normalizedTenantID(topic.TenantID)
	topic.Title = strings.TrimSpace(topic.Title)
	topic.Body = strings.TrimSpace(topic.Body)
	topic.CreatedAt = now
	topic.UpdatedAt = now
	_, err := db.ExecContext(context.Background(), `INSERT INTO community_topics (id,tenant_id,community_id,title,body,author_id,author_name,author_role,author_org,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)`,
		topic.ID, topic.TenantID, topic.CommunityID, topic.Title, topic.Body, topic.AuthorID, topic.Author, topic.Role, topic.Org, now)
	return topic, err
}

func DeleteCommunityTopic(communityID, topicID, tenantID, actorID string) error {
	role, roleErr := CommunityMembership(communityID, tenantID, actorID)
	if roleErr != nil {
		return roleErr
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM community_topics WHERE id=$1 AND community_id=$2 AND tenant_id=$3 AND (author_id=$4 OR $5='owner')`, topicID, communityID, normalizedTenantID(tenantID), actorID, role)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func GetCommunityReplies(communityID, topicID, tenantID, userID string) ([]model.CommunityReply, error) {
	if _, err := CommunityMembership(communityID, tenantID, userID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(context.Background(), `
SELECT id,tenant_id,community_id,topic_id,author_id,author_name,author_role,author_org,body,created_at
FROM community_replies
WHERE community_id=$1 AND topic_id=$2 AND tenant_id=$3
ORDER BY created_at`, communityID, topicID, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	replies := []model.CommunityReply{}
	for rows.Next() {
		var reply model.CommunityReply
		if err := rows.Scan(&reply.ID, &reply.TenantID, &reply.CommunityID, &reply.TopicID, &reply.AuthorID, &reply.Author, &reply.Role, &reply.Org, &reply.Body, &reply.CreatedAt); err != nil {
			return nil, err
		}
		replies = append(replies, reply)
	}
	return replies, rows.Err()
}

func CreateCommunityReply(reply model.CommunityReply) (model.CommunityReply, error) {
	if _, err := CommunityMembership(reply.CommunityID, reply.TenantID, reply.AuthorID); err != nil {
		return reply, err
	}
	var topicExists bool
	if err := db.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM community_topics WHERE id=$1 AND community_id=$2 AND tenant_id=$3)`, reply.TopicID, reply.CommunityID, normalizedTenantID(reply.TenantID)).Scan(&topicExists); err != nil || !topicExists {
		if err != nil {
			return reply, err
		}
		return reply, sql.ErrNoRows
	}
	now := time.Now().UTC()
	reply.ID = uuid.NewString()
	reply.TenantID = normalizedTenantID(reply.TenantID)
	reply.Body = strings.TrimSpace(reply.Body)
	reply.CreatedAt = now
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return reply, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(context.Background(), `INSERT INTO community_replies (id,tenant_id,community_id,topic_id,author_id,author_name,author_role,author_org,body,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		reply.ID, reply.TenantID, reply.CommunityID, reply.TopicID, reply.AuthorID, reply.Author, reply.Role, reply.Org, reply.Body, now); err != nil {
		return reply, err
	}
	if _, err = tx.ExecContext(context.Background(), `UPDATE community_topics SET updated_at=$1 WHERE id=$2 AND community_id=$3 AND tenant_id=$4`, now, reply.TopicID, reply.CommunityID, reply.TenantID); err != nil {
		return reply, err
	}
	return reply, tx.Commit()
}

func DeleteCommunityReply(communityID, topicID, replyID, tenantID, actorID string) error {
	role, roleErr := CommunityMembership(communityID, tenantID, actorID)
	if roleErr != nil {
		return roleErr
	}
	result, err := db.ExecContext(context.Background(), `DELETE FROM community_replies WHERE id=$1 AND community_id=$2 AND topic_id=$3 AND tenant_id=$4 AND (author_id=$5 OR $6='owner')`, replyID, communityID, topicID, normalizedTenantID(tenantID), actorID, role)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return sql.ErrNoRows
	}
	_, err = db.ExecContext(context.Background(), `UPDATE community_topics SET updated_at=$1 WHERE id=$2 AND community_id=$3 AND tenant_id=$4`, time.Now().UTC(), topicID, communityID, normalizedTenantID(tenantID))
	return err
}
