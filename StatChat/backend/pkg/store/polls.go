package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"statchat/pkg/model"
)

func CreatePoll(poll model.Poll) (model.Poll, error) {
	poll.ID = uuid.NewString()
	poll.TenantID = normalizedTenantID(poll.TenantID)
	poll.Question = strings.TrimSpace(poll.Question)
	poll.CreatedAt = time.Now().UTC()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return poll, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO polls (id,tenant_id,question,created_by,created_at) VALUES ($1,$2,$3,$4,$5)`, poll.ID, poll.TenantID, poll.Question, poll.CreatedBy, poll.CreatedAt); err != nil {
		return poll, err
	}
	for index := range poll.Options {
		poll.Options[index].ID = uuid.NewString()
		poll.Options[index].Label = strings.TrimSpace(poll.Options[index].Label)
		if _, err = tx.Exec(`INSERT INTO poll_options (id,poll_id,label,position) VALUES ($1,$2,$3,$4)`, poll.Options[index].ID, poll.ID, poll.Options[index].Label, index); err != nil {
			return poll, err
		}
	}
	if err = tx.Commit(); err != nil {
		return poll, err
	}
	return poll, nil
}

func GetPolls(tenantID, userID string) ([]model.Poll, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT p.id,p.tenant_id,p.question,p.created_by,p.created_at,
       o.id,o.label,COUNT(v.option_id),EXISTS(SELECT 1 FROM poll_votes uv WHERE uv.poll_id=p.id AND uv.user_id=$2)
FROM polls p JOIN poll_options o ON o.poll_id=p.id LEFT JOIN poll_votes v ON v.option_id=o.id
WHERE p.tenant_id=$1 GROUP BY p.id,p.tenant_id,p.question,p.created_by,p.created_at,o.id,o.label,o.position
ORDER BY p.created_at DESC,o.position`, normalizedTenantID(tenantID), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	polls := []model.Poll{}
	var current *model.Poll
	for rows.Next() {
		var poll model.Poll
		var option model.PollOption
		if err := rows.Scan(&poll.ID, &poll.TenantID, &poll.Question, &poll.CreatedBy, &poll.CreatedAt, &option.ID, &option.Label, &option.Votes, &poll.Voted); err != nil {
			return nil, err
		}
		if current == nil || current.ID != poll.ID {
			poll.Options = []model.PollOption{}
			polls = append(polls, poll)
			current = &polls[len(polls)-1]
		}
		current.Options = append(current.Options, option)
	}
	return polls, rows.Err()
}

func VotePoll(pollID, optionID, tenantID, userID string) error {
	var valid bool
	err := db.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM poll_options o JOIN polls p ON p.id=o.poll_id WHERE o.id=$1 AND p.id=$2 AND p.tenant_id=$3)`, optionID, pollID, normalizedTenantID(tenantID)).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return sql.ErrNoRows
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO poll_votes (poll_id,option_id,tenant_id,user_id,created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (poll_id,user_id) DO NOTHING`, pollID, optionID, normalizedTenantID(tenantID), userID, time.Now().UTC())
	return err
}
