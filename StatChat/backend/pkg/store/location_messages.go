package store

import (
	"context"
	"database/sql"

	"statchat/pkg/model"
)

func StoreLocationMessage(message model.Message, location model.MessageLocation) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO messages (id, conversation_id, channel_id, sender_id, sender, text, created_at, status, tenant_id, delivery_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, message.ID, message.ConversationID, nullString(message.ChannelID), nullString(message.SenderID), message.Sender, message.Text, message.CreatedAt, message.Status, normalizedTenantID(message.TenantID), message.DeliveryStatus); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO message_locations (message_id, latitude, longitude, accuracy_meters, label, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, message.ID, location.Latitude, location.Longitude, nullFloat64(location.AccuracyMeters), location.Label, location.CreatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func GetMessageLocation(messageID string) (*model.MessageLocation, error) {
	var location model.MessageLocation
	var accuracy sql.NullFloat64
	err := db.QueryRowContext(context.Background(), `SELECT message_id, latitude, longitude, accuracy_meters, label, created_at FROM message_locations WHERE message_id = $1`, messageID).Scan(&location.MessageID, &location.Latitude, &location.Longitude, &accuracy, &location.Label, &location.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if accuracy.Valid {
		location.AccuracyMeters = accuracy.Float64
	}
	return &location, nil
}

func nullFloat64(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}
