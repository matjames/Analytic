package store

import (
	"context"
	"database/sql"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func ensureConferencingSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS call_sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL DEFAULT 'default',
  room_id TEXT NOT NULL,
  room_name TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'video',
  host_id TEXT NOT NULL,
  host_name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'live',
  conversation_id TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ
);
ALTER TABLE call_sessions ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT 'default';
CREATE INDEX IF NOT EXISTS idx_call_sessions_status ON call_sessions(status);
CREATE INDEX IF NOT EXISTS idx_call_sessions_room ON call_sessions(room_id);

CREATE TABLE IF NOT EXISTS call_participants (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES call_sessions(id),
  user_id TEXT NOT NULL,
  user_name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'participant',
  joined_at TIMESTAMPTZ NOT NULL,
  left_at TIMESTAMPTZ,
  UNIQUE (session_id, user_id, left_at)
);
CREATE INDEX IF NOT EXISTS idx_call_participants_session ON call_participants(session_id);
-- Partial unique index: a user may only be an active participant once per session.
DROP INDEX IF EXISTS idx_call_participants_active_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_call_participants_active_unique
  ON call_participants(session_id, user_id) WHERE left_at IS NULL;
-- Migrate away from the legacy composite unique constraint that allowed
-- duplicate active participants (NULLs are distinct in PostgreSQL).
ALTER TABLE call_participants DROP CONSTRAINT IF EXISTS call_participants_session_id_user_id_left_at_key;

CREATE TABLE IF NOT EXISTS call_recordings (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES call_sessions(id),
  title TEXT NOT NULL,
  file_name TEXT NOT NULL,
  url TEXT NOT NULL,
  size BIGINT NOT NULL DEFAULT 0,
  duration TEXT,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_call_recordings_session ON call_recordings(session_id);

CREATE TABLE IF NOT EXISTS call_quality_samples (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES call_sessions(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  rtt_ms DOUBLE PRECISION NOT NULL,
  jitter_ms DOUBLE PRECISION NOT NULL,
  packet_loss_pct DOUBLE PRECISION NOT NULL,
  bitrate_kbps DOUBLE PRECISION NOT NULL,
  quality TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_call_quality_session_created ON call_quality_samples(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_call_quality_created ON call_quality_samples(created_at);
`)
	return err
}

func CreateCallSession(session model.CallSession) (model.CallSession, error) {
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	if session.RoomID == "" {
		session.RoomID = uuid.NewString()
	}
	if session.Status == "" {
		session.Status = model.CallStatusLive
	}
	if session.Kind == "" {
		session.Kind = model.CallKindVideo
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `
INSERT INTO call_sessions (id, tenant_id, room_id, room_name, kind, host_id, host_name, status, conversation_id, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		session.ID, normalizedTenantID(session.TenantID), session.RoomID, session.RoomName, session.Kind, session.HostID, session.HostName, session.Status, nullString(session.Conversation), session.CreatedAt)
	return session, err
}

func GetCallSession(sessionID string) (model.CallSession, error) {
	var s model.CallSession
	var conversationID sql.NullString
	var endedAt sql.NullTime
	err := db.QueryRowContext(context.Background(), `
SELECT id, tenant_id, room_id, room_name, kind, host_id, host_name, status, conversation_id, created_at, ended_at
FROM call_sessions WHERE id = $1`, sessionID).Scan(
		&s.ID, &s.TenantID, &s.RoomID, &s.RoomName, &s.Kind, &s.HostID, &s.HostName, &s.Status, &conversationID, &s.CreatedAt, &endedAt)
	if err != nil {
		return s, err
	}
	s.Conversation = conversationID.String
	if endedAt.Valid {
		s.EndedAt = endedAt.Time
	}
	return s, nil
}

func FindCallSessionByRoom(roomID, tenantID string) (model.CallSession, error) {
	var s model.CallSession
	var conversationID sql.NullString
	var endedAt sql.NullTime
	err := db.QueryRowContext(context.Background(), `
SELECT id, tenant_id, room_id, room_name, kind, host_id, host_name, status, conversation_id, created_at, ended_at
FROM call_sessions WHERE room_id = $1 AND tenant_id = $2 AND status = 'live' ORDER BY created_at DESC LIMIT 1`, roomID, normalizedTenantID(tenantID)).Scan(
		&s.ID, &s.TenantID, &s.RoomID, &s.RoomName, &s.Kind, &s.HostID, &s.HostName, &s.Status, &conversationID, &s.CreatedAt, &endedAt)
	if err != nil {
		return s, err
	}
	s.Conversation = conversationID.String
	if endedAt.Valid {
		s.EndedAt = endedAt.Time
	}
	return s, nil
}

func ListActiveCallSessions(tenantID string) ([]model.CallSession, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, tenant_id, room_id, room_name, kind, host_id, host_name, status, conversation_id, created_at, ended_at
FROM call_sessions WHERE tenant_id = $1 AND status = 'live' ORDER BY created_at DESC`, normalizedTenantID(tenantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []model.CallSession{}
	for rows.Next() {
		var s model.CallSession
		var conversationID sql.NullString
		var endedAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.TenantID, &s.RoomID, &s.RoomName, &s.Kind, &s.HostID, &s.HostName, &s.Status, &conversationID, &s.CreatedAt, &endedAt); err != nil {
			return nil, err
		}
		s.Conversation = conversationID.String
		if endedAt.Valid {
			s.EndedAt = endedAt.Time
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func EndCallSession(sessionID string) (model.CallSession, error) {
	_, err := db.ExecContext(context.Background(), `
UPDATE call_sessions SET status = 'ended', ended_at = $1 WHERE id = $2`, time.Now().UTC(), sessionID)
	if err != nil {
		return model.CallSession{}, err
	}
	// Mark all active participants as left
	_, err = db.ExecContext(context.Background(), `
UPDATE call_participants SET left_at = $1 WHERE session_id = $2 AND left_at IS NULL`, time.Now().UTC(), sessionID)
	if err != nil {
		return model.CallSession{}, err
	}
	syncMeetingParticipantCount(sessionID)
	return GetCallSession(sessionID)
}

func JoinCallSession(sessionID string, userID string, userName string) (model.CallParticipant, error) {
	p := model.CallParticipant{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		UserID:    userID,
		UserName:  userName,
		Role:      "participant",
		JoinedAt:  time.Now().UTC(),
	}
	if userName == "" {
		p.UserName = userID
	}
	var hostID string
	if err := db.QueryRowContext(context.Background(), `SELECT host_id FROM call_sessions WHERE id = $1`, sessionID).Scan(&hostID); err != nil {
		return model.CallParticipant{}, err
	}
	if userID == hostID {
		p.Role = "host"
	}
	_, err := db.ExecContext(context.Background(), `
INSERT INTO call_participants (id, session_id, user_id, user_name, role, joined_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (session_id, user_id) WHERE left_at IS NULL DO UPDATE SET joined_at = EXCLUDED.joined_at, left_at = NULL`,
		p.ID, sessionID, userID, p.UserName, p.Role, p.JoinedAt)
	if err != nil {
		return model.CallParticipant{}, err
	}
	// Update meeting participant count if this session maps to a meeting room.
	// The session's room_id holds the meeting room token, so use that as the
	// match key rather than an empty string.
	syncMeetingParticipantCount(sessionID)
	return p, nil
}

func LeaveCallSession(sessionID string, userID string) error {
	_, err := db.ExecContext(context.Background(), `
UPDATE call_participants SET left_at = $1 WHERE session_id = $2 AND user_id = $3 AND left_at IS NULL`,
		time.Now().UTC(), sessionID, userID)
	if err == nil {
		syncMeetingParticipantCount(sessionID)
	}
	return err
}

func syncMeetingParticipantCount(sessionID string) {
	var roomID string
	if err := db.QueryRowContext(context.Background(), `SELECT room_id FROM call_sessions WHERE id = $1`, sessionID).Scan(&roomID); err != nil || roomID == "" {
		return
	}
	_, _ = db.ExecContext(context.Background(), `
UPDATE meetings SET participants = (SELECT COUNT(1) FROM call_participants WHERE session_id = $1 AND left_at IS NULL)
WHERE room = $2`, sessionID, roomID)
}

func GetCallParticipants(sessionID string) ([]model.CallParticipant, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, session_id, user_id, user_name, role, joined_at, left_at
FROM call_participants WHERE session_id = $1 ORDER BY joined_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := []model.CallParticipant{}
	for rows.Next() {
		var p model.CallParticipant
		var leftAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.SessionID, &p.UserID, &p.UserName, &p.Role, &p.JoinedAt, &leftAt); err != nil {
			return nil, err
		}
		if leftAt.Valid {
			p.LeftAt = leftAt.Time
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func GetActiveParticipants(sessionID string) ([]model.CallParticipant, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, session_id, user_id, user_name, role, joined_at
FROM call_participants WHERE session_id = $1 AND left_at IS NULL ORDER BY joined_at`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := []model.CallParticipant{}
	for rows.Next() {
		var p model.CallParticipant
		if err := rows.Scan(&p.ID, &p.SessionID, &p.UserID, &p.UserName, &p.Role, &p.JoinedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func IsActiveCallParticipant(sessionID, userID string) (bool, error) {
	var active bool
	err := db.QueryRowContext(context.Background(), `SELECT EXISTS (SELECT 1 FROM call_participants WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL)`, sessionID, userID).Scan(&active)
	return active, err
}

func GetActiveCallParticipantRole(sessionID, userID string) (string, error) {
	var role string
	err := db.QueryRowContext(context.Background(), `SELECT role FROM call_participants WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`, sessionID, userID).Scan(&role)
	return role, err
}

func UpdateCallParticipantRole(sessionID, userID, role string) (bool, error) {
	result, err := db.ExecContext(context.Background(), `UPDATE call_participants SET role = $1 WHERE session_id = $2 AND user_id = $3 AND left_at IS NULL`, role, sessionID, userID)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func RemoveCallParticipant(sessionID, userID string) (bool, error) {
	result, err := db.ExecContext(context.Background(), `UPDATE call_participants SET left_at = NOW() WHERE session_id = $1 AND user_id = $2 AND left_at IS NULL`, sessionID, userID)
	if err != nil {
		return false, err
	}
	removed, err := result.RowsAffected()
	if err == nil && removed == 1 {
		syncMeetingParticipantCount(sessionID)
	}
	return removed == 1, err
}

func SaveCallRecording(rec model.CallRecording) (model.CallRecording, error) {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	if rec.Title == "" {
		rec.Title = "Meeting recording"
	}
	_, err := db.ExecContext(context.Background(), `
INSERT INTO call_recordings (id, session_id, title, file_name, url, size, duration, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		rec.ID, rec.SessionID, rec.Title, rec.FileName, rec.URL, rec.Size, nullString(rec.Duration), rec.CreatedAt)
	return rec, err
}

func GetCallRecordings(sessionID string) ([]model.CallRecording, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, session_id, title, file_name, url, size, COALESCE(duration,''), created_at
FROM call_recordings WHERE session_id = $1 ORDER BY created_at DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recs := []model.CallRecording{}
	for rows.Next() {
		var r model.CallRecording
		var duration sql.NullString
		if err := rows.Scan(&r.ID, &r.SessionID, &r.Title, &r.FileName, &r.URL, &r.Size, &duration, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Duration = duration.String
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

func SaveCallQualitySample(sample model.CallQualitySample) (model.CallQualitySample, error) {
	if sample.ID == "" {
		sample.ID = uuid.NewString()
	}
	if sample.CreatedAt.IsZero() {
		sample.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO call_quality_samples (id, session_id, user_id, rtt_ms, jitter_ms, packet_loss_pct, bitrate_kbps, quality, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, sample.ID, sample.SessionID, sample.UserID, sample.RTTMs, sample.JitterMs, sample.PacketLossPct, sample.BitrateKbps, sample.Quality, sample.CreatedAt)
	if err == nil {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM call_quality_samples WHERE created_at < NOW() - INTERVAL '30 days'`)
	}
	return sample, err
}

func GetCallQualitySamples(sessionID string, limit int) ([]model.CallQualitySample, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := db.QueryContext(context.Background(), `SELECT id, session_id, user_id, rtt_ms, jitter_ms, packet_loss_pct, bitrate_kbps, quality, created_at FROM call_quality_samples WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	samples := []model.CallQualitySample{}
	for rows.Next() {
		var sample model.CallQualitySample
		if err := rows.Scan(&sample.ID, &sample.SessionID, &sample.UserID, &sample.RTTMs, &sample.JitterMs, &sample.PacketLossPct, &sample.BitrateKbps, &sample.Quality, &sample.CreatedAt); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}
