package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func ensureCollaborationSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS posts (
  id TEXT PRIMARY KEY,
  author TEXT NOT NULL,
  role TEXT NOT NULL,
  org TEXT NOT NULL,
  time_label TEXT NOT NULL,
  text TEXT NOT NULL,
  likes INT NOT NULL DEFAULT 0,
  comments INT NOT NULL DEFAULT 0,
  shares INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS connections (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  connected_to_id TEXT NOT NULL REFERENCES users(id),
  connected_at TIMESTAMPTZ NOT NULL,
  UNIQUE (user_id, connected_to_id)
);

CREATE TABLE IF NOT EXISTS post_comments (
  id TEXT PRIMARY KEY,
  post_id TEXT NOT NULL REFERENCES posts(id),
  author TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  org TEXT NOT NULL DEFAULT '',
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS post_likes (
  id TEXT PRIMARY KEY,
  post_id TEXT NOT NULL REFERENCES posts(id),
  user_id TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (post_id, user_id)
);

CREATE TABLE IF NOT EXISTS opportunities (
  id TEXT PRIMARY KEY,
  badge TEXT NOT NULL,
  badge_color TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  company TEXT NOT NULL,
  location TEXT NOT NULL,
  type TEXT NOT NULL,
  salary TEXT NOT NULL,
  icon TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS meetings (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  date TEXT NOT NULL,
  time TEXT NOT NULL,
  duration TEXT NOT NULL,
  participants INT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'upcoming',
  room TEXT NOT NULL,
  host TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS meeting_rooms (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  capacity INT NOT NULL,
  status TEXT NOT NULL DEFAULT 'available',
  password TEXT NOT NULL,
  url TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS meeting_recordings (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  date TEXT NOT NULL,
  duration TEXT NOT NULL,
  size TEXT NOT NULL,
  url TEXT,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS wellness_posts (
  id TEXT PRIMARY KEY,
  author TEXT NOT NULL,
  handle TEXT NOT NULL,
  avatar TEXT NOT NULL,
  category TEXT NOT NULL,
  time_label TEXT NOT NULL,
  text TEXT NOT NULL,
  likes INT NOT NULL DEFAULT 0,
  comments INT NOT NULL DEFAULT 0,
  shares INT NOT NULL DEFAULT 0,
  bookmarks INT NOT NULL DEFAULT 0,
  tags JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS knowledge_experts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  org TEXT NOT NULL,
  specialties JSONB NOT NULL DEFAULT '[]',
  followers INT NOT NULL DEFAULT 0,
  articles INT NOT NULL DEFAULT 0,
  rating DOUBLE PRECISION NOT NULL DEFAULT 0,
  avatar TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS knowledge_articles (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category TEXT NOT NULL,
  read_time TEXT NOT NULL,
  excerpt TEXT NOT NULL,
  likes INT NOT NULL DEFAULT 0,
  views INT NOT NULL DEFAULT 0,
  published TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS knowledge_ideas (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category TEXT NOT NULL,
  description TEXT NOT NULL,
  votes INT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'discussing'
);

CREATE TABLE IF NOT EXISTS knowledge_posts (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  category TEXT NOT NULL,
  content TEXT NOT NULL,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);
`)
	return err
}

func seedCollaborationDefaults(ctx context.Context) error {
	// Seed wellness posts independently (not gated by posts table check)
	var wellnessCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM wellness_posts`).Scan(&wellnessCount); err != nil {
		return err
	}
	if wellnessCount == 0 {
		now := time.Now().UTC()
		wellnessPosts := []model.WellnessPost{
			{ID: "w1", Author: "Dr. Sarah Chen", Handle: "@drsarah", Avatar: "🧠", Category: "Mental Health", Time: "2h", Text: "Reminder: Your mental health is not a luxury. It's a necessity. Taking a break isn't giving up — it's giving yourself the space to grow stronger. 💚 #MentalHealthMatters", Likes: 1240, Comments: 89, Shares: 234, Bookmarks: 156, Tags: []string{"#MentalHealth", "#SelfCare", "#Wellness"}, CreatedAt: now.Add(-2 * time.Hour)},
			{ID: "w2", Author: "Mindful Marcus", Handle: "@mindfulmarcus", Avatar: "🧘", Category: "Mindfulness", Time: "4h", Text: "Try this 60-second breathing exercise right now:\n\n1. Breathe in for 4 seconds\n2. Hold for 4 seconds\n3. Breathe out for 4 seconds\n4. Hold for 4 seconds\n\nRepeat 4 times. Notice the difference. 🌿 #Mindfulness #AnxietyRelief", Likes: 890, Comments: 45, Shares: 312, Bookmarks: 478, Tags: []string{"#Mindfulness", "#Breathing", "#AnxietyRelief"}, CreatedAt: now.Add(-4 * time.Hour)},
			{ID: "w3", Author: "Growth Guide", Handle: "@growthguide", Avatar: "🌱", Category: "Development", Time: "6h", Text: "5 habits that changed my life:\n\n1. Journaling for 5 minutes every morning\n2. Reading 10 pages before bed\n3. Walking 20 minutes daily\n4. Gratitude practice before sleep\n5. One meaningful conversation per day\n\nSmall steps. Big change. 📚 #PersonalDevelopment", Likes: 2150, Comments: 167, Shares: 543, Bookmarks: 892, Tags: []string{"#PersonalDevelopment", "#Habits", "#Growth"}, CreatedAt: now.Add(-6 * time.Hour)},
			{ID: "w4", Author: "Anxiety Ally", Handle: "@anxietyally", Avatar: "💙", Category: "Anxiety", Time: "8h", Text: "Having an anxious day? That's okay. You're not broken. Anxiety is your brain trying to protect you — it's just being a little too protective.\n\nYou've survived 100% of your bad days so far. That's a pretty good track record. 💪 #AnxietySupport", Likes: 3450, Comments: 234, Shares: 678, Bookmarks: 1234, Tags: []string{"#Anxiety", "#MentalHealth", "#YouAreNotAlone"}, CreatedAt: now.Add(-8 * time.Hour)},
			{ID: "w5", Author: "Self-Care Sam", Handle: "@selfcaresam", Avatar: "🛁", Category: "Self-Care", Time: "12h", Text: "Self-care isn't selfish. It's maintenance.\n\nYou can't pour from an empty cup. Take the nap. Eat the meal. Go for the walk. Say no. Rest is productive too. 🧖 #SelfCare #RestIsProductive", Likes: 1890, Comments: 78, Shares: 445, Bookmarks: 678, Tags: []string{"#SelfCare", "#Rest", "#Boundaries"}, CreatedAt: now.Add(-12 * time.Hour)},
			{ID: "w6", Author: "Motivation Mike", Handle: "@motivatemike", Avatar: "🔥", Category: "Motivation", Time: "1d", Text: "The version of you that you're becoming is going to be so proud of the version of you that didn't give up.\n\nKeep going. Even on the hard days. Especially on the hard days. 🚀 #Motivation #KeepGoing", Likes: 4520, Comments: 312, Shares: 890, Bookmarks: 1567, Tags: []string{"#Motivation", "#KeepGoing", "#MentalHealth"}, CreatedAt: now.Add(-24 * time.Hour)},
		}
		for _, wp := range wellnessPosts {
			tagsJSON, _ := json.Marshal(wp.Tags)
			if _, err := db.ExecContext(ctx, `INSERT INTO wellness_posts (id, author, handle, avatar, category, time_label, text, likes, comments, shares, bookmarks, tags, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT DO NOTHING`, wp.ID, wp.Author, wp.Handle, wp.Avatar, wp.Category, wp.Time, wp.Text, wp.Likes, wp.Comments, wp.Shares, wp.Bookmarks, tagsJSON, wp.CreatedAt); err != nil {
				return err
			}
		}
	}

	var postCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM posts`).Scan(&postCount); err != nil {
		return err
	}
	if postCount > 0 {
		return nil
	}

	now := time.Now().UTC()
	posts := []model.Post{
		{ID: uuid.NewString(), Author: "Jonas Gonzalez", Role: "Senior Researcher", Org: "Ministry of Finance", Time: "2h", Text: "Excited to share that our team has published the quarterly statistical bulletin. The data shows a 12% improvement in data quality across all districts. Proud of the team effort! 📊", Likes: 47, Comments: 12, Shares: 8, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: uuid.NewString(), Author: "Claire Wang", Role: "Lead Data Scientist", Org: "Digital Services", Time: "5h", Text: "Looking for collaborators on a machine learning project for anomaly detection in health surveillance data. If you have experience with time-series analysis, let's connect! 🤖", Likes: 89, Comments: 23, Shares: 15, CreatedAt: now.Add(-5 * time.Hour)},
		{ID: uuid.NewString(), Author: "Rahul Kumar", Role: "Research Manager", Org: "Research Operations", Time: "1d", Text: "Our research on maternal health indicators has been accepted for publication. Thanks to the StatGate platform for making cross-district collaboration seamless. The full report will be available next week. 📄", Likes: 156, Comments: 34, Shares: 42, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: uuid.NewString(), Author: "Maya Smith", Role: "Enterprise Architect", Org: "Enterprise Architecture", Time: "2d", Text: "Just completed the data governance framework for 2026. Key highlights: automated metadata management, role-based access controls, and real-time data lineage tracking. Happy to discuss with anyone interested! 🏛️", Likes: 72, Comments: 18, Shares: 11, CreatedAt: now.Add(-48 * time.Hour)},
	}
	for _, p := range posts {
		if _, err := db.ExecContext(ctx, `INSERT INTO posts (id, author, role, org, time_label, text, likes, comments, shares, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, p.ID, p.Author, p.Role, p.Org, p.Time, p.Text, p.Likes, p.Comments, p.Shares, p.CreatedAt); err != nil {
			return err
		}
	}

	opps := []model.Opportunity{
		{ID: "o1", Badge: "COLLABORATION", BadgeColor: "#0b5fff", Title: "National Census 2030 — Data Collection Team", Description: "Seeking 50 data collectors and 10 supervisors for the national census. Training provided. Apply by August 15."},
		{ID: "o2", Badge: "RESEARCH", BadgeColor: "#7c3aed", Title: "Malaria Surveillance Study — Principal Investigator", Description: "Lead a multi-district malaria surveillance study. Requires PhD in epidemiology or public health. 2-year contract."},
		{ID: "o3", Badge: "TRAINING", BadgeColor: "#16a34a", Title: "Advanced R Programming Workshop", Description: "3-day intensive workshop on advanced R programming for statistical analysis. Limited to 30 participants."},
		{ID: "o4", Badge: "PUBLICATION", BadgeColor: "#d97706", Title: "Call for Papers — Health Informatics Journal", Description: "Submit your research on health information systems and digital health. Deadline: September 30."},
	}
	for _, o := range opps {
		if _, err := db.ExecContext(ctx, `INSERT INTO opportunities (id, badge, badge_color, title, description, created_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, o.ID, o.Badge, o.BadgeColor, o.Title, o.Description, now); err != nil {
			return err
		}
	}

	jobs := []model.Job{
		{ID: "j1", Title: "Senior Biostatistician", Company: "Ministry of Health", Location: "Nairobi, Kenya", Type: "Full-time", Salary: "KES 250K-350K", Icon: "📊"},
		{ID: "j2", Title: "Data Engineer", Company: "Analytics Lab", Location: "Remote", Type: "Full-time", Salary: "KES 200K-300K", Icon: "⚙️"},
		{ID: "j3", Title: "GIS Specialist", Company: "Digital Services", Location: "Kampala, Uganda", Type: "Contract", Salary: "KES 180K-250K", Icon: "🗺️"},
		{ID: "j4", Title: "AI Research Lead", Company: "AI Initiative", Location: "Remote", Type: "Full-time", Salary: "KES 350K-500K", Icon: "🤖"},
	}
	for _, j := range jobs {
		if _, err := db.ExecContext(ctx, `INSERT INTO jobs (id, title, company, location, type, salary, icon, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, j.ID, j.Title, j.Company, j.Location, j.Type, j.Salary, j.Icon, now); err != nil {
			return err
		}
	}

	meetings := []model.Meeting{
		{ID: "m1", Title: "Weekly Team Sync", Date: "2026-08-03", Time: "10:00 AM", Duration: "45 min", Participants: 12, Status: "upcoming", Room: "StatGate Room 1", Host: "Amina Patel", CreatedAt: now},
		{ID: "m2", Title: "Research Review Board", Date: "2026-08-03", Time: "2:00 PM", Duration: "90 min", Participants: 8, Status: "upcoming", Room: "Statistics Hall", Host: "Jonas Gonzalez", CreatedAt: now},
		{ID: "m3", Title: "Malaria Surveillance Sync", Date: "2026-08-04", Time: "11:00 AM", Duration: "60 min", Participants: 25, Status: "upcoming", Room: "Public Health Room", Host: "Claire Wang", CreatedAt: now},
		{ID: "m4", Title: "Data Governance Board", Date: "2026-08-05", Time: "9:00 AM", Duration: "45 min", Participants: 6, Status: "upcoming", Room: "Board Room", Host: "Maya Smith", CreatedAt: now},
		{ID: "m5", Title: "District Health Officers Briefing", Date: "2026-08-02", Time: "3:00 PM", Duration: "120 min", Participants: 45, Status: "live", Room: "StatGate Room 1", Host: "Rahul Kumar", CreatedAt: now},
	}
	for _, m := range meetings {
		if _, err := db.ExecContext(ctx, `INSERT INTO meetings (id, title, date, time, duration, participants, status, room, host, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, m.ID, m.Title, m.Date, m.Time, m.Duration, m.Participants, m.Status, m.Room, m.Host, m.CreatedAt); err != nil {
			return err
		}
	}

	rooms := []model.MeetingRoom{
		{ID: "r1", Name: "StatGate Room 1", Capacity: 50, Status: "available", Password: "SG-4821", URL: "statchat.local/room/sg-4821"},
		{ID: "r2", Name: "Statistics Hall", Capacity: 30, Status: "available", Password: "SH-9374", URL: "statchat.local/room/sh-9374"},
		{ID: "r3", Name: "Public Health Room", Capacity: 40, Status: "available", Password: "PH-2058", URL: "statchat.local/room/ph-2058"},
		{ID: "r4", Name: "Board Room", Capacity: 20, Status: "in-use", Password: "BR-6642", URL: "statchat.local/room/br-6642"},
		{ID: "r5", Name: "AI Lab Room", Capacity: 25, Status: "available", Password: "AI-1197", URL: "statchat.local/room/ai-1197"},
	}
	for _, r := range rooms {
		if _, err := db.ExecContext(ctx, `INSERT INTO meeting_rooms (id, name, capacity, status, password, url, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, r.ID, r.Name, r.Capacity, r.Status, r.Password, r.URL, now); err != nil {
			return err
		}
	}

	recordings := []model.MeetingRecording{
		{ID: "rec1", Title: "Weekly Team Sync — July 26", Date: "2026-07-26", Duration: "42 min", Size: "28 MB", CreatedAt: now.Add(-7 * 24 * time.Hour)},
		{ID: "rec2", Title: "Data Quality Workshop", Date: "2026-07-24", Duration: "1h 15m", Size: "62 MB", CreatedAt: now.Add(-9 * 24 * time.Hour)},
		{ID: "rec3", Title: "GIS Training Session", Date: "2026-07-22", Duration: "55 min", Size: "41 MB", CreatedAt: now.Add(-11 * 24 * time.Hour)},
	}
	for _, rec := range recordings {
		if _, err := db.ExecContext(ctx, `INSERT INTO meeting_recordings (id, title, date, duration, size, url, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, rec.ID, rec.Title, rec.Date, rec.Duration, rec.Size, rec.URL, rec.CreatedAt); err != nil {
			return err
		}
	}

	return nil
}

func GetPosts(userID string) ([]model.Post, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, author, role, org, time_label, text, likes, comments, shares, created_at FROM posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []model.Post{}
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.Author, &p.Role, &p.Org, &p.Time, &p.Text, &p.Likes, &p.Comments, &p.Shares, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach comments and likedByMe for each post
	for i := range posts {
		comments, err := GetPostComments(posts[i].ID)
		if err != nil {
			return nil, err
		}
		posts[i].CommentList = comments
		posts[i].Comments = len(comments)
		liked, err := IsPostLikedByUser(posts[i].ID, userID)
		if err != nil {
			return nil, err
		}
		posts[i].LikedByMe = liked
	}
	return posts, nil
}

// ── Post Comments ──

func GetPostComments(postID string) ([]model.PostComment, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, post_id, author, role, org, text, created_at FROM post_comments WHERE post_id = $1 ORDER BY created_at ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []model.PostComment{}
	for rows.Next() {
		var c model.PostComment
		if err := rows.Scan(&c.ID, &c.PostID, &c.Author, &c.Role, &c.Org, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func AddPostComment(req model.PostComment) (model.PostComment, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO post_comments (id, post_id, author, role, org, text, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		req.ID, req.PostID, req.Author, req.Role, req.Org, req.Text, req.CreatedAt)
	if err != nil {
		return model.PostComment{}, err
	}
	// Update the count on the post
	_, err = db.ExecContext(context.Background(), `UPDATE posts SET comments = (SELECT COUNT(1) FROM post_comments WHERE post_id = $1) WHERE id = $1`, req.PostID)
	return req, err
}

// ── Post Likes ──

func IsPostLikedByUser(postID, userID string) (bool, error) {
	var count int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(1) FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func TogglePostLike(postID, userID string) (bool, error) {
	liked, err := IsPostLikedByUser(postID, userID)
	if err != nil {
		return false, err
	}
	if liked {
		_, err = db.ExecContext(context.Background(), `DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
		if err != nil {
			return false, err
		}
	} else {
		_, err = db.ExecContext(context.Background(), `INSERT INTO post_likes (id, post_id, user_id, created_at) VALUES ($1,$2,$3,$4)`,
			uuid.NewString(), postID, userID, time.Now().UTC())
		if err != nil {
			return false, err
		}
	}
	// Update the count on the post
	_, err = db.ExecContext(context.Background(), `UPDATE posts SET likes = (SELECT COUNT(1) FROM post_likes WHERE post_id = $1) WHERE id = $1`, postID)
	return !liked, err
}

func SharePost(postID string) (int, error) {
	_, err := db.ExecContext(context.Background(), `UPDATE posts SET shares = shares + 1 WHERE id = $1`, postID)
	if err != nil {
		return 0, err
	}
	var shares int
	err = db.QueryRowContext(context.Background(), `SELECT shares FROM posts WHERE id = $1`, postID).Scan(&shares)
	return shares, err
}

func CreatePost(req model.Post) (model.Post, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	if req.Time == "" {
		req.Time = "now"
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO posts (id, author, role, org, time_label, text, likes, comments, shares, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.Author, req.Role, req.Org, req.Time, req.Text, req.Likes, req.Comments, req.Shares, req.CreatedAt)
	return req, err
}

func GetConnections(userID string) ([]model.Connection, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT c.id, c.user_id, c.connected_to_id, u.name, u.roles, u.organization_id, c.connected_at
FROM connections c
JOIN users u ON u.id = c.connected_to_id
WHERE c.user_id = $1
ORDER BY c.connected_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conns := []model.Connection{}
	for rows.Next() {
		var conn model.Connection
		var rolesJSON []byte
		if err := rows.Scan(&conn.ID, &conn.UserID, &conn.ConnectedToID, &conn.ConnectedName, &rolesJSON, &conn.ConnectedOrg, &conn.ConnectedAt); err != nil {
			return nil, err
		}
		var roles []string
		json.Unmarshal(rolesJSON, &roles)
		if len(roles) > 0 {
			conn.ConnectedRole = roles[0]
		}
		conns = append(conns, conn)
	}
	return conns, rows.Err()
}

func CreateConnection(userID string, connectedToID string) (model.Connection, error) {
	var target model.User
	var rolesJSON []byte
	err := db.QueryRowContext(context.Background(), `SELECT id, name, roles, organization_id FROM users WHERE id = $1`, connectedToID).Scan(&target.ID, &target.Name, &rolesJSON, &target.OrganizationID)
	if err != nil {
		return model.Connection{}, err
	}
	var roles []string
	json.Unmarshal(rolesJSON, &roles)

	conn := model.Connection{
		ID:            uuid.NewString(),
		UserID:        userID,
		ConnectedToID: connectedToID,
		ConnectedName: target.Name,
		ConnectedOrg:  target.OrganizationID,
		ConnectedAt:   time.Now().UTC(),
	}
	if len(roles) > 0 {
		conn.ConnectedRole = roles[0]
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO connections (id, user_id, connected_to_id, connected_at) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
		conn.ID, userID, connectedToID, conn.ConnectedAt)
	return conn, err
}

func RemoveConnection(userID string, connectedToID string) error {
	_, err := db.ExecContext(context.Background(), `DELETE FROM connections WHERE user_id = $1 AND connected_to_id = $2`, userID, connectedToID)
	return err
}

func GetOpportunities() ([]model.Opportunity, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, badge, badge_color, title, description FROM opportunities ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	opps := []model.Opportunity{}
	for rows.Next() {
		var o model.Opportunity
		if err := rows.Scan(&o.ID, &o.Badge, &o.BadgeColor, &o.Title, &o.Description); err != nil {
			return nil, err
		}
		opps = append(opps, o)
	}
	return opps, rows.Err()
}

func GetJobs() ([]model.Job, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, company, location, type, salary, icon FROM jobs ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []model.Job{}
	for rows.Next() {
		var j model.Job
		if err := rows.Scan(&j.ID, &j.Title, &j.Company, &j.Location, &j.Type, &j.Salary, &j.Icon); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func GetMeetings() ([]model.Meeting, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, date, time, duration, participants, status, room, host, created_at FROM meetings ORDER BY date, time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meetings := []model.Meeting{}
	for rows.Next() {
		var m model.Meeting
		if err := rows.Scan(&m.ID, &m.Title, &m.Date, &m.Time, &m.Duration, &m.Participants, &m.Status, &m.Room, &m.Host, &m.CreatedAt); err != nil {
			return nil, err
		}
		meetings = append(meetings, m)
	}
	return meetings, rows.Err()
}

func CreateMeeting(req model.Meeting) (model.Meeting, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.Status == "" {
		req.Status = "upcoming"
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO meetings (id, title, date, time, duration, participants, status, room, host, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.Title, req.Date, req.Time, req.Duration, req.Participants, req.Status, req.Room, req.Host, req.CreatedAt)
	return req, err
}

func GetMeetingRooms() ([]model.MeetingRoom, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, name, capacity, status, password, url FROM meeting_rooms ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := []model.MeetingRoom{}
	for rows.Next() {
		var r model.MeetingRoom
		if err := rows.Scan(&r.ID, &r.Name, &r.Capacity, &r.Status, &r.Password, &r.URL); err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}

func GetMeetingRecordings() ([]model.MeetingRecording, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, date, duration, size, url, created_at FROM meeting_recordings ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recs := []model.MeetingRecording{}
	for rows.Next() {
		var rec model.MeetingRecording
		var url sql.NullString
		if err := rows.Scan(&rec.ID, &rec.Title, &rec.Date, &rec.Duration, &rec.Size, &url, &rec.CreatedAt); err != nil {
			return nil, err
		}
		rec.URL = url.String
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func GetWellnessPosts() ([]model.WellnessPost, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, author, handle, avatar, category, time_label, text, likes, comments, shares, bookmarks, tags, created_at FROM wellness_posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []model.WellnessPost{}
	for rows.Next() {
		var p model.WellnessPost
		var tagsJSON []byte
		if err := rows.Scan(&p.ID, &p.Author, &p.Handle, &p.Avatar, &p.Category, &p.Time, &p.Text, &p.Likes, &p.Comments, &p.Shares, &p.Bookmarks, &tagsJSON, &p.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsJSON, &p.Tags)
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func CreateWellnessPost(req model.WellnessPost) (model.WellnessPost, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	if req.Time == "" {
		req.Time = "now"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	tagsJSON, _ := json.Marshal(req.Tags)
	_, err := db.ExecContext(context.Background(), `INSERT INTO wellness_posts (id, author, handle, avatar, category, time_label, text, likes, comments, shares, bookmarks, tags, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		req.ID, req.Author, req.Handle, req.Avatar, req.Category, req.Time, req.Text, req.Likes, req.Comments, req.Shares, req.Bookmarks, tagsJSON, req.CreatedAt)
	return req, err
}

// Knowledge store functions
func seedKnowledgeDefaults(ctx context.Context) error {
	// Seed knowledge experts
	experts := []model.KnowledgeExpert{
		{ID: "e1", Name: "Dr. Sarah Chen", Role: "Senior Biostatistician", Org: "Ministry of Health", Specialties: []string{"Biostatistics", "Clinical Trials"}, Followers: 3420, Articles: 48, Rating: 4.9, Avatar: "🔬"},
		{ID: "e2", Name: "Prof. James Okello", Role: "Data Science Professor", Org: "University of Nairobi", Specialties: []string{"Machine Learning", "Statistical Modeling"}, Followers: 5670, Articles: 89, Rating: 4.8, Avatar: "📊"},
		{ID: "e3", Name: "Dr. Maria Rodriguez", Role: "Research Director", Org: "National Statistics Office", Specialties: []string{"Survey Methodology", "Census Design"}, Followers: 2890, Articles: 35, Rating: 4.7, Avatar: "📋"},
		{ID: "e4", Name: "Kevin Mwangi", Role: "Full Stack Developer", Org: "StatGate Labs", Specialties: []string{"React", "Go", "Data Engineering"}, Followers: 4120, Articles: 67, Rating: 4.9, Avatar: "💻"},
		{ID: "e5", Name: "Dr. Amina Hassan", Role: "Epidemiologist", Org: "CDC Africa", Specialties: []string{"Disease Surveillance", "Outbreak Analysis"}, Followers: 3780, Articles: 52, Rating: 4.6, Avatar: "🦠"},
		{ID: "e6", Name: "Prof. David Kim", Role: "AI Research Lead", Org: "AI Initiative", Specialties: []string{"Deep Learning", "NLP"}, Followers: 6280, Articles: 94, Rating: 5.0, Avatar: "🤖"},
	}
	for _, e := range experts {
		specJSON, _ := json.Marshal(e.Specialties)
		_, err := db.ExecContext(ctx, `INSERT INTO knowledge_experts (id, name, role, org, specialties, followers, articles, rating, avatar) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			e.ID, e.Name, e.Role, e.Org, specJSON, e.Followers, e.Articles, e.Rating, e.Avatar)
		if err != nil {
			return err
		}
	}

	// Seed knowledge articles
	articles := []model.KnowledgeArticle{
		{ID: "a1", Title: "Introduction to Bayesian Statistics for Health Data", Author: "Dr. Sarah Chen", Category: "Statistics", ReadTime: "12 min", Excerpt: "Learn how Bayesian methods can transform your health data analysis with practical examples from real surveillance programs.", Likes: 234, Views: 12450, Published: "2d ago"},
		{ID: "a2", Title: "Building Scalable Data Pipelines with Go and PostgreSQL", Author: "Kevin Mwangi", Category: "Development", ReadTime: "18 min", Excerpt: "A practical guide to designing data pipelines that handle millions of records without breaking a sweat.", Likes: 456, Views: 23400, Published: "3d ago"},
		{ID: "a3", Title: "Machine Learning for Disease Outbreak Prediction", Author: "Prof. David Kim", Category: "Machine Learning", ReadTime: "15 min", Excerpt: "How deep learning models are being used to predict disease outbreaks before they spread — with case studies from East Africa.", Likes: 678, Views: 34500, Published: "1d ago"},
		{ID: "a4", Title: "Designing Effective Household Surveys: Lessons from Census 2030", Author: "Dr. Maria Rodriguez", Category: "Research", ReadTime: "10 min", Excerpt: "Key principles for designing household surveys that minimize bias and maximize data quality.", Likes: 189, Views: 8930, Published: "4d ago"},
		{ID: "a5", Title: "Time Series Analysis for Disease Surveillance Data", Author: "Dr. Amina Hassan", Category: "Analysis", ReadTime: "14 min", Excerpt: "Master the art of detecting anomalies in time-series disease surveillance data using R and Python.", Likes: 312, Views: 15670, Published: "5d ago"},
		{ID: "a6", Title: "Teaching Statistics to Non-Statisticians", Author: "Prof. James Okello", Category: "Education", ReadTime: "8 min", Excerpt: "Practical techniques for making complex statistical concepts accessible to public health professionals.", Likes: 445, Views: 19800, Published: "6d ago"},
	}
	for _, a := range articles {
		_, err := db.ExecContext(ctx, `INSERT INTO knowledge_articles (id, title, author, category, read_time, excerpt, likes, views, published) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			a.ID, a.Title, a.Author, a.Category, a.ReadTime, a.Excerpt, a.Likes, a.Views, a.Published)
		if err != nil {
			return err
		}
	}

	// Seed knowledge ideas
	ideas := []model.KnowledgeIdea{
		{ID: "i1", Title: "Open Health Data Repository", Author: "Dr. Amina Hassan", Category: "Research", Description: "A centralized, anonymized health data repository where researchers can share datasets.", Votes: 156, Status: "discussing"},
		{ID: "i2", Title: "Real-time Disease Surveillance Dashboard", Author: "Prof. David Kim", Category: "Development", Description: "Building a real-time dashboard that integrates data from multiple health facilities.", Votes: 234, Status: "developing"},
		{ID: "i3", Title: "Statistical Literacy Program", Author: "Prof. James Okello", Category: "Education", Description: "A free, structured program to teach statistical literacy to health workers.", Votes: 189, Status: "launched"},
		{ID: "i4", Title: "AI-Assisted Data Cleaning Toolkit", Author: "Kevin Mwangi", Category: "Development", Description: "An open-source toolkit that uses machine learning to automatically detect and fix data quality issues.", Votes: 167, Status: "discussing"},
	}
	for _, i := range ideas {
		_, err := db.ExecContext(ctx, `INSERT INTO knowledge_ideas (id, title, author, category, description, votes, status) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			i.ID, i.Title, i.Author, i.Category, i.Description, i.Votes, i.Status)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetKnowledgeExperts() ([]model.KnowledgeExpert, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, name, role, org, specialties, followers, articles, rating, avatar FROM knowledge_experts ORDER BY followers DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	experts := []model.KnowledgeExpert{}
	for rows.Next() {
		var e model.KnowledgeExpert
		var specJSON []byte
		if err := rows.Scan(&e.ID, &e.Name, &e.Role, &e.Org, &specJSON, &e.Followers, &e.Articles, &e.Rating, &e.Avatar); err != nil {
			return nil, err
		}
		json.Unmarshal(specJSON, &e.Specialties)
		experts = append(experts, e)
	}
	return experts, rows.Err()
}

func GetKnowledgeArticles() ([]model.KnowledgeArticle, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, author, category, read_time, excerpt, likes, views, published FROM knowledge_articles ORDER BY likes DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	articles := []model.KnowledgeArticle{}
	for rows.Next() {
		var a model.KnowledgeArticle
		if err := rows.Scan(&a.ID, &a.Title, &a.Author, &a.Category, &a.ReadTime, &a.Excerpt, &a.Likes, &a.Views, &a.Published); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, rows.Err()
}

func GetKnowledgeIdeas() ([]model.KnowledgeIdea, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, author, category, description, votes, status FROM knowledge_ideas ORDER BY votes DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ideas := []model.KnowledgeIdea{}
	for rows.Next() {
		var i model.KnowledgeIdea
		if err := rows.Scan(&i.ID, &i.Title, &i.Author, &i.Category, &i.Description, &i.Votes, &i.Status); err != nil {
			return nil, err
		}
		ideas = append(ideas, i)
	}
	return ideas, rows.Err()
}

func CreateKnowledgePost(req model.KnowledgePost) (model.KnowledgePost, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now().UTC()
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO knowledge_posts (id, title, author, category, content, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		req.ID, req.Title, req.Author, req.Category, req.Content, req.CreatedBy, req.CreatedAt)
	return req, err
}

func UpvoteKnowledgeIdea(ideaID string) (int, error) {
	_, err := db.ExecContext(context.Background(), `UPDATE knowledge_ideas SET votes = votes + 1 WHERE id = $1`, ideaID)
	if err != nil {
		return 0, err
	}
	var votes int
	err = db.QueryRowContext(context.Background(), `SELECT votes FROM knowledge_ideas WHERE id = $1`, ideaID).Scan(&votes)
	return votes, err
}

func FollowKnowledgeExpert(expertID string) (int, error) {
	_, err := db.ExecContext(context.Background(), `UPDATE knowledge_experts SET followers = followers + 1 WHERE id = $1`, expertID)
	if err != nil {
		return 0, err
	}
	var followers int
	err = db.QueryRowContext(context.Background(), `SELECT followers FROM knowledge_experts WHERE id = $1`, expertID).Scan(&followers)
	return followers, err
}

func GetKnowledgePosts() ([]model.KnowledgePost, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, title, author, category, content, created_by, created_at FROM knowledge_posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := []model.KnowledgePost{}
	for rows.Next() {
		var p model.KnowledgePost
		if err := rows.Scan(&p.ID, &p.Title, &p.Author, &p.Category, &p.Content, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}
