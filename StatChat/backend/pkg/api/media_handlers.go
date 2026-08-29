package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/google/uuid"
)

type mediaCatalogItem struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	Tags       []string `json:"tags"`
	PreviewURL string   `json:"previewUrl"`
	Color      string   `json:"-"`
	Accent     string   `json:"-"`
	Animation  string   `json:"-"`
}

var mediaCatalog = []mediaCatalogItem{
	{ID: "celebrate", Kind: "gif", Label: "Celebrate!", Tags: []string{"yay", "success", "party", "congratulations"}, Color: "#f7c948", Accent: "#c05621", Animation: "bounce"},
	{ID: "applause", Kind: "gif", Label: "Applause", Tags: []string{"clap", "great", "thanks", "bravo"}, Color: "#ef8354", Accent: "#7c2d12", Animation: "pulse"},
	{ID: "amazing", Kind: "gif", Label: "Amazing!", Tags: []string{"wow", "excellent", "awesome"}, Color: "#38a169", Accent: "#14532d", Animation: "tilt"},
	{ID: "on-it", Kind: "gif", Label: "On it!", Tags: []string{"working", "task", "acknowledged"}, Color: "#3182ce", Accent: "#1e3a8a", Animation: "slide"},
	{ID: "thank-you", Kind: "gif", Label: "Thank you", Tags: []string{"thanks", "grateful", "appreciation"}, Color: "#d69e2e", Accent: "#713f12", Animation: "pulse"},
	{ID: "good-news", Kind: "gif", Label: "Good news!", Tags: []string{"update", "success", "announcement"}, Color: "#0f766e", Accent: "#134e4a", Animation: "bounce"},
	{ID: "approved", Kind: "sticker", Label: "Approved", Tags: []string{"done", "accepted", "complete"}, Color: "#22c55e", Accent: "#14532d"},
	{ID: "needs-review", Kind: "sticker", Label: "Needs review", Tags: []string{"review", "check", "feedback"}, Color: "#f59e0b", Accent: "#78350f"},
	{ID: "urgent", Kind: "sticker", Label: "Urgent", Tags: []string{"priority", "important", "attention"}, Color: "#ef4444", Accent: "#7f1d1d"},
	{ID: "great-work", Kind: "sticker", Label: "Great work", Tags: []string{"team", "excellent", "thanks"}, Color: "#14b8a6", Accent: "#134e4a"},
	{ID: "received", Kind: "sticker", Label: "Received", Tags: []string{"acknowledged", "noted", "seen"}, Color: "#3b82f6", Accent: "#1e3a8a"},
	{ID: "lets-discuss", Kind: "sticker", Label: "Let's discuss", Tags: []string{"meeting", "talk", "question"}, Color: "#64748b", Accent: "#1e293b"},
}

func init() {
	for index := range mediaCatalog {
		mediaCatalog[index].PreviewURL = mediaSVGDataURL(mediaCatalog[index])
	}
}

func mediaSVGDataURL(item mediaCatalogItem) string {
	animation := ""
	if item.Kind == "gif" {
		animation = fmt.Sprintf(`<style>@keyframes %s{0%%,100%%{transform:translate(0,0) scale(1) rotate(0)}50%%{transform:translate(0,-8px) scale(1.06) rotate(2deg)}}.card{transform-origin:120px 70px;animation:%s 1.1s ease-in-out infinite}</style>`, item.Animation, item.Animation)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="240" height="140" viewBox="0 0 240 140">%s<g class="card"><rect x="4" y="4" width="232" height="132" rx="28" fill="%s"/><path d="M24 104 C72 80 166 126 218 88" fill="none" stroke="%s" stroke-width="8" opacity=".35"/><text x="120" y="78" text-anchor="middle" font-family="Verdana,sans-serif" font-size="24" font-weight="700" fill="#fff">%s</text></g></svg>`, animation, item.Color, item.Accent, escapeSVGText(item.Label))
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

func escapeSVGText(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	return strings.ReplaceAll(value, ">", "&gt;")
}

func findMediaItem(id string) (mediaCatalogItem, bool) {
	for _, item := range mediaCatalog {
		if item.ID == id {
			return item, true
		}
	}
	return mediaCatalogItem{}, false
}

func mediaCatalogHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	if len(query) > 60 || (kind != "" && kind != "gif" && kind != "sticker") {
		writeError(w, http.StatusBadRequest, "invalid media search")
		return
	}
	items := []mediaCatalogItem{}
	for _, item := range mediaCatalog {
		haystack := strings.ToLower(item.Label + " " + strings.Join(item.Tags, " "))
		if (kind == "" || item.Kind == kind) && (query == "" || strings.Contains(haystack, query)) {
			items = append(items, item)
		}
	}
	writeJSON(w, items)
}

func sendCatalogMediaHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConversationID string `json:"conversationId"`
		MediaID        string `json:"mediaId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	item, found := findMediaItem(strings.TrimSpace(req.MediaID))
	if req.ConversationID == "" || !found {
		writeError(w, http.StatusBadRequest, "valid conversationId and mediaId are required")
		return
	}
	if !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	message := model.Message{ID: uuid.NewString(), TenantID: requestTenantID(r), ConversationID: req.ConversationID, SenderID: requestUserID(r), Sender: requestUserName(r), Text: item.Label, CreatedAt: time.Now().UTC(), Status: "active", DeliveryStatus: "sent"}
	attachment := model.MessageAttachment{ID: uuid.NewString(), MessageID: message.ID, FileName: item.Kind + "-" + item.ID + ".svg", FileType: "image/svg+xml", MimeType: "image/svg+xml", URL: item.PreviewURL, CreatedAt: message.CreatedAt}
	message.Attachments = []model.MessageAttachment{attachment}
	if err := store.StoreMessageWithAttachments(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send media")
		return
	}
	store.BroadcastMessage(message)
	_ = store.NotifyConversationMembers(message.ConversationID, requestUserID(r), requestUserName(r), message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), nil)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, message)
}
