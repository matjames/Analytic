package model

import "time"

type User struct {
	ID             string   `json:"id"`
	TenantID       string   `json:"tenantId,omitempty"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	OrganizationID string   `json:"organizationId"`
	Roles          []string `json:"roles"`
	AvatarURL      string   `json:"avatarUrl,omitempty"`
	About          string   `json:"about,omitempty"`
	Presence       string   `json:"presence,omitempty"`
}

type UserSettings struct {
	UserID             string `json:"userId"`
	Theme              string `json:"theme"`
	AccentColor        string `json:"accentColor"`
	FontSize           string `json:"fontSize"`
	EnterToSend        bool   `json:"enterToSend"`
	Language           string `json:"language"`
	LastSeen           string `json:"lastSeen"`
	ProfilePhoto       string `json:"profilePhoto"`
	ReadReceipts       bool   `json:"readReceipts"`
	TypingIndicator    bool   `json:"typingIndicator"`
	VoiceNotes         bool   `json:"voiceNotes"`
	ReadByDefault      bool   `json:"readByDefault"`
	AutoDownload       string `json:"autoDownload"`
	NotifMessages      bool   `json:"notifMessages"`
	NotifGroups        bool   `json:"notifGroups"`
	NotifMentions      bool   `json:"notifMentions"`
	NotifMeetings      bool   `json:"notifMeetings"`
	NotifCollaboration bool   `json:"notifCollaboration"`
	NotifFiles         bool   `json:"notifFiles"`
	NotifKnowledge     bool   `json:"notifKnowledge"`
	NotifWellness      bool   `json:"notifWellness"`
	NotifSound         bool   `json:"notifSound"`
	NotifPreview       bool   `json:"notifPreview"`
	CrossServiceAlerts bool   `json:"crossServiceAlerts"`
	DownloadImages     string `json:"downloadImages"`
	DownloadVideos     string `json:"downloadVideos"`
	DownloadDocuments  string `json:"downloadDocuments"`
	Wallpaper          string `json:"wallpaper,omitempty"`
}

type ConversationType string

const (
	ConversationTypeChannel ConversationType = "channel"
	ConversationTypeDirect  ConversationType = "direct"
	ConversationTypeGroup   ConversationType = "group"
)

type Conversation struct {
	ID              string           `json:"id"`
	TenantID        string           `json:"tenantId,omitempty"`
	ObjectRef       string           `json:"objectRef,omitempty"`
	Name            string           `json:"name"`
	Type            ConversationType `json:"type"`
	MemberIDs       []string         `json:"memberIds"`
	ChannelID       string           `json:"channelId,omitempty"`
	Category        string           `json:"category,omitempty"`
	LatestPreview   string           `json:"latestPreview,omitempty"`
	LatestMessageAt time.Time        `json:"latestMessageAt,omitempty"`
	AttachmentCount int              `json:"attachmentCount,omitempty"`
	UnreadCount     int              `json:"unreadCount,omitempty"`
	Metadata        map[string]any   `json:"metadata,omitempty"`
}

type ConversationCategory string

const (
	CategoryDirect  ConversationCategory = "direct"
	CategoryGroup   ConversationCategory = "group"
	CategoryChannel ConversationCategory = "channel"
)

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	MemberIDs   []string  `json:"memberIds"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GroupCategory struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Icon        string  `json:"icon"`
	Description string  `json:"description"`
	Groups      []Group `json:"groups,omitempty"`
}

type Channel struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Visibility  string    `json:"visibility"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	MemberCount int       `json:"memberCount"`
	Joined      bool      `json:"joined"`
	Archived    bool      `json:"archived"`
}

type Post struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenantId,omitempty"`
	AuthorID    string        `json:"authorId,omitempty"`
	Author      string        `json:"author"`
	Role        string        `json:"role"`
	Org         string        `json:"org"`
	Time        string        `json:"time"`
	Text        string        `json:"text"`
	Likes       int           `json:"likes"`
	Comments    int           `json:"comments"`
	Shares      int           `json:"shares"`
	CreatedAt   time.Time     `json:"createdAt"`
	LikedByMe   bool          `json:"likedByMe,omitempty"`
	CommentList []PostComment `json:"commentList,omitempty"`
}

type PostComment struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId,omitempty"`
	AuthorID  string    `json:"authorId,omitempty"`
	PostID    string    `json:"postId"`
	Author    string    `json:"author"`
	Role      string    `json:"role,omitempty"`
	Org       string    `json:"org,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type PollOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Votes int    `json:"votes"`
}

type Poll struct {
	ID        string       `json:"id"`
	TenantID  string       `json:"tenantId,omitempty"`
	Question  string       `json:"question"`
	Options   []PollOption `json:"options"`
	CreatedBy string       `json:"createdBy"`
	CreatedAt time.Time    `json:"createdAt"`
	Voted     bool         `json:"voted"`
}

type Connection struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	ConnectedToID string    `json:"connectedToId"`
	ConnectedName string    `json:"connectedName"`
	ConnectedRole string    `json:"connectedRole"`
	ConnectedOrg  string    `json:"connectedOrg"`
	ConnectedAt   time.Time `json:"connectedAt"`
}

type Community struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId,omitempty"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Visibility   string    `json:"visibility"`
	CreatedBy    string    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	MemberCount  int       `json:"memberCount"`
	TopicCount   int       `json:"topicCount"`
	Joined       bool      `json:"joined"`
	Role         string    `json:"role,omitempty"`
	CanPost      bool      `json:"canPost"`
	LatestPostAt time.Time `json:"latestPostAt,omitempty"`
}

type CommunityMember struct {
	CommunityID string    `json:"communityId"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	UserRole    string    `json:"userRole,omitempty"`
	Org         string    `json:"org,omitempty"`
	JoinedAt    time.Time `json:"joinedAt"`
}

type CommunityTopic struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	CommunityID string    `json:"communityId"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	AuthorID    string    `json:"authorId"`
	Author      string    `json:"author"`
	Role        string    `json:"role,omitempty"`
	Org         string    `json:"org,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
	ReplyCount  int       `json:"replyCount"`
}

type CommunityReply struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId,omitempty"`
	CommunityID string    `json:"communityId"`
	TopicID     string    `json:"topicId"`
	AuthorID    string    `json:"authorId"`
	Author      string    `json:"author"`
	Role        string    `json:"role,omitempty"`
	Org         string    `json:"org,omitempty"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
}

type CollaborationDocument struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId,omitempty"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"createdBy"`
	Author    string    `json:"author"`
	UpdatedBy string    `json:"updatedBy"`
	Version   int       `json:"version"`
	Role      string    `json:"role"`
	CanEdit   bool      `json:"canEdit"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CollaborationDocumentMember struct {
	DocumentID string    `json:"documentId"`
	UserID     string    `json:"userId"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	Org        string    `json:"org,omitempty"`
	AddedAt    time.Time `json:"addedAt"`
}

type CollaborationDocumentRevision struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"documentId"`
	Version    int       `json:"version"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	EditedBy   string    `json:"editedBy"`
	EditedAt   time.Time `json:"editedAt"`
}

type Opportunity struct {
	ID          string `json:"id"`
	Badge       string `json:"badge"`
	BadgeColor  string `json:"badgeColor"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Job struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Location string `json:"location"`
	Type     string `json:"type"`
	Salary   string `json:"salary"`
	Icon     string `json:"icon"`
}

type Meeting struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Date         string    `json:"date"`
	Time         string    `json:"time"`
	Duration     string    `json:"duration"`
	Participants int       `json:"participants"`
	Status       string    `json:"status"`
	Room         string    `json:"room"`
	Host         string    `json:"host"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MeetingRoom struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Status   string `json:"status"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type MeetingRecording struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Date      string    `json:"date"`
	Duration  string    `json:"duration"`
	Size      string    `json:"size"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type WellnessPost struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Handle    string    `json:"handle"`
	Avatar    string    `json:"avatar"`
	Category  string    `json:"category"`
	Time      string    `json:"time"`
	Text      string    `json:"text"`
	Likes     int       `json:"likes"`
	Comments  int       `json:"comments"`
	Shares    int       `json:"shares"`
	Bookmarks int       `json:"bookmarks"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
}

type KnowledgeExpert struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenantId,omitempty"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Org         string   `json:"org"`
	Specialties []string `json:"specialties"`
	Followers   int      `json:"followers"`
	Articles    int      `json:"articles"`
	Rating      float64  `json:"rating"`
	Avatar      string   `json:"avatar"`
	Following   bool     `json:"following,omitempty"`
}

type KnowledgeArticle struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenantId,omitempty"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Category  string `json:"category"`
	ReadTime  string `json:"readTime"`
	Excerpt   string `json:"excerpt"`
	Content   string `json:"content,omitempty"`
	Likes     int    `json:"likes"`
	Views     int    `json:"views"`
	Published string `json:"published"`
}

type KnowledgeIdea struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenantId,omitempty"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	AuthorID    string `json:"authorId,omitempty"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Votes       int    `json:"votes"`
	Status      string `json:"status"`
	Upvoted     bool   `json:"upvoted,omitempty"`
}

type KnowledgePost struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId,omitempty"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	AuthorID  string    `json:"authorId,omitempty"`
	Role      string    `json:"role,omitempty"`
	Org       string    `json:"org,omitempty"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageAttachment struct {
	ID        string    `json:"id"`
	MessageID string    `json:"messageId,omitempty"`
	FileName  string    `json:"fileName"`
	FileType  string    `json:"fileType"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mimeType"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageLocation struct {
	MessageID      string    `json:"messageId,omitempty"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	AccuracyMeters float64   `json:"accuracyMeters,omitempty"`
	Label          string    `json:"label,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Message struct {
	ID                     string              `json:"id"`
	TenantID               string              `json:"tenantId,omitempty"`
	ConversationID         string              `json:"conversationId,omitempty"`
	ChannelID              string              `json:"channelId,omitempty"`
	SenderID               string              `json:"senderId,omitempty"`
	Sender                 string              `json:"sender"`
	Text                   string              `json:"text"`
	CreatedAt              time.Time           `json:"createdAt"`
	UpdatedAt              time.Time           `json:"updatedAt,omitempty"`
	DeletedAt              time.Time           `json:"deletedAt,omitempty"`
	ParentMessageID        string              `json:"parentMessageId,omitempty"`
	ThreadRootID           string              `json:"threadRootId,omitempty"`
	Status                 string              `json:"status"`
	DeliveryStatus         string              `json:"deliveryStatus,omitempty"`
	Attachments            []MessageAttachment `json:"attachments,omitempty"`
	Reactions              []MessageReaction   `json:"reactions,omitempty"`
	Pinned                 bool                `json:"pinned,omitempty"`
	ReadBy                 []string            `json:"readBy,omitempty"`
	ForwardedFromMessageID string              `json:"forwardedFromMessageId,omitempty"`
	ForwardedFromSender    string              `json:"forwardedFromSender,omitempty"`
	Location               *MessageLocation    `json:"location,omitempty"`
	MentionUserIDs         []string            `json:"mentionUserIds,omitempty"`
	MentionAll             bool                `json:"mentionAll,omitempty"`
	SavedAt                time.Time           `json:"savedAt,omitempty"`
}

type ScheduledMessage struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	ConversationID string    `json:"conversationId"`
	SenderID       string    `json:"senderId"`
	Sender         string    `json:"sender"`
	Text           string    `json:"text"`
	ScheduledFor   time.Time `json:"scheduledFor"`
	Status         string    `json:"status"`
	MessageID      string    `json:"messageId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type MessageReaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"messageId"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName,omitempty"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"createdAt"`
}

type PinnedMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	MessageID      string    `json:"messageId"`
	PinnedBy       string    `json:"pinnedBy"`
	PinnedAt       time.Time `json:"pinnedAt"`
}

type ReadReceipt struct {
	ID        string    `json:"id"`
	MessageID string    `json:"messageId"`
	UserID    string    `json:"userId"`
	ReadAt    time.Time `json:"readAt"`
}

type Task struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	Assignee       string    `json:"assignee,omitempty"`
	Priority       string    `json:"priority,omitempty"`
	DueDate        string    `json:"dueDate,omitempty"`
	Status         string    `json:"status"`
	ConversationID string    `json:"conversationId,omitempty"`
	CreatedBy      string    `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

type CalendarEvent struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	Location       string    `json:"location,omitempty"`
	StartAt        time.Time `json:"startAt"`
	EndAt          time.Time `json:"endAt"`
	AllDay         bool      `json:"allDay"`
	ConversationID string    `json:"conversationId,omitempty"`
	AttendeeIDs    []string  `json:"attendeeIds"`
	CreatedBy      string    `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

type GatewayEnvelope struct {
	Event    string      `json:"event"`
	TenantID string      `json:"tenantId,omitempty"`
	Payload  interface{} `json:"payload"`
}

type RetentionPolicy struct {
	TenantID      string    `json:"tenantId"`
	RetentionDays int       `json:"retentionDays"`
	Enabled       bool      `json:"enabled"`
	UpdatedBy     string    `json:"updatedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type LegalHold struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenantId"`
	ConversationID string    `json:"conversationId,omitempty"`
	Name           string    `json:"name"`
	Reason         string    `json:"reason"`
	Status         string    `json:"status"`
	CreatedBy      string    `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
	ReleasedBy     string    `json:"releasedBy,omitempty"`
	ReleasedAt     time.Time `json:"releasedAt,omitempty"`
}

type ComplianceAuditEvent struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenantId"`
	ActorID    string         `json:"actorId"`
	Action     string         `json:"action"`
	TargetType string         `json:"targetType"`
	TargetID   string         `json:"targetId"`
	Details    map[string]any `json:"details,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type Presence struct {
	UserID    string    `json:"userId"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ── Conferencing (Native WebRTC) ──

type CallKind string

const (
	CallKindVoice CallKind = "voice"
	CallKindVideo CallKind = "video"
)

type CallStatus string

const (
	CallStatusScheduled CallStatus = "scheduled"
	CallStatusLive      CallStatus = "live"
	CallStatusEnded     CallStatus = "ended"
)

type CallSession struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenantId"`
	RoomID       string     `json:"roomId"`
	RoomName     string     `json:"roomName"`
	Kind         CallKind   `json:"kind"`
	HostID       string     `json:"hostId"`
	HostName     string     `json:"hostName"`
	Status       CallStatus `json:"status"`
	Conversation string     `json:"conversationId,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	EndedAt      time.Time  `json:"endedAt,omitempty"`
}

type CallParticipant struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	UserID    string    `json:"userId"`
	UserName  string    `json:"userName"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joinedAt"`
	LeftAt    time.Time `json:"leftAt,omitempty"`
}

type CallRecording struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Title     string    `json:"title"`
	FileName  string    `json:"fileName"`
	URL       string    `json:"url"`
	Size      int64     `json:"size"`
	Duration  string    `json:"duration,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CallQualitySample struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"sessionId"`
	UserID        string    `json:"userId"`
	RTTMs         float64   `json:"rttMs"`
	JitterMs      float64   `json:"jitterMs"`
	PacketLossPct float64   `json:"packetLossPct"`
	BitrateKbps   float64   `json:"bitrateKbps"`
	Quality       string    `json:"quality"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CallSignal struct {
	Type      string `json:"type"` // offer | answer | ice-candidate | screen-share-* | mute-*
	SessionID string `json:"sessionId"`
	From      string `json:"from"`
	FromName  string `json:"fromName,omitempty"`
	To        string `json:"to,omitempty"`
	Payload   string `json:"payload,omitempty"`
}
