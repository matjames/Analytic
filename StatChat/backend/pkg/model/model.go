package model

import "time"

type User struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	OrganizationID string   `json:"organizationId"`
	Roles          []string `json:"roles"`
	AvatarURL      string   `json:"avatarUrl,omitempty"`
	About          string   `json:"about,omitempty"`
	Presence       string   `json:"presence,omitempty"`
}

type UserSettings struct {
	UserID            string `json:"userId"`
	Theme             string `json:"theme"`
	AccentColor       string `json:"accentColor"`
	FontSize          string `json:"fontSize"`
	EnterToSend       bool   `json:"enterToSend"`
	Language          string `json:"language"`
	LastSeen          string `json:"lastSeen"`
	ProfilePhoto      string `json:"profilePhoto"`
	ReadReceipts      bool   `json:"readReceipts"`
	TypingIndicator   bool   `json:"typingIndicator"`
	VoiceNotes        bool   `json:"voiceNotes"`
	ReadByDefault     bool   `json:"readByDefault"`
	AutoDownload      string `json:"autoDownload"`
	NotifMessages     bool   `json:"notifMessages"`
	NotifGroups       bool   `json:"notifGroups"`
	NotifMentions     bool   `json:"notifMentions"`
	NotifMeetings     bool   `json:"notifMeetings"`
	NotifSound        bool   `json:"notifSound"`
	NotifPreview      bool   `json:"notifPreview"`
	DownloadImages    string `json:"downloadImages"`
	DownloadVideos    string `json:"downloadVideos"`
	DownloadDocuments string `json:"downloadDocuments"`
}

type ConversationType string

const (
	ConversationTypeChannel ConversationType = "channel"
	ConversationTypeDirect  ConversationType = "direct"
	ConversationTypeGroup   ConversationType = "group"
)

type Conversation struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Type            ConversationType `json:"type"`
	MemberIDs       []string         `json:"memberIds"`
	ChannelID       string           `json:"channelId,omitempty"`
	Category        string           `json:"category,omitempty"`
	LatestPreview   string           `json:"latestPreview,omitempty"`
	LatestMessageAt time.Time        `json:"latestMessageAt,omitempty"`
	AttachmentCount int              `json:"attachmentCount,omitempty"`
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
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Post struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Role      string    `json:"role"`
	Org       string    `json:"org"`
	Time      string    `json:"time"`
	Text      string    `json:"text"`
	Likes     int       `json:"likes"`
	Comments  int       `json:"comments"`
	Shares    int       `json:"shares"`
	CreatedAt time.Time `json:"createdAt"`
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
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Org         string   `json:"org"`
	Specialties []string `json:"specialties"`
	Followers   int      `json:"followers"`
	Articles    int      `json:"articles"`
	Rating      float64  `json:"rating"`
	Avatar      string   `json:"avatar"`
}

type KnowledgeArticle struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Category  string `json:"category"`
	ReadTime  string `json:"readTime"`
	Excerpt   string `json:"excerpt"`
	Likes     int    `json:"likes"`
	Views     int    `json:"views"`
	Published string `json:"published"`
}

type KnowledgeIdea struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Votes       int    `json:"votes"`
	Status      string `json:"status"`
}

type KnowledgePost struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Category  string    `json:"category"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type Message struct {
	ID              string    `json:"id"`
	ConversationID  string    `json:"conversationId,omitempty"`
	ChannelID       string    `json:"channelId,omitempty"`
	Sender          string    `json:"sender"`
	Text            string    `json:"text"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty"`
	DeletedAt       time.Time `json:"deletedAt,omitempty"`
	ParentMessageID string    `json:"parentMessageId,omitempty"`
	ThreadRootID    string    `json:"threadRootId,omitempty"`
	Status          string    `json:"status"`
}
