package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

// Callback holds the details for clients to call back the Notifier
// and receive notifications.
type Callback struct {
	NotificationID uuid.UUID `json:"notification_id"`
	Callback       url.URL   `json:"callback,string"`
}

// CbMessage is the actual type used for over the wire. The above's struct tags
// are for documentation only.
type cbMessage struct {
	ID       *uuid.UUID `json:"notification_id"`
	Callback string     `json:"callback"`
}

// MarshalJSON implements json.Marshaler.
func (cb Callback) MarshalJSON() ([]byte, error) {
	// This writes out a block of bytes directly, instead of allocating a
	// cbMessage structure, serializing the URL, then copying it into a []byte.
	b := bytes.NewBufferString(`{"callback":"`)
	b.WriteString(cb.Callback.String())
	b.WriteString(`","notification_id":"`)
	b.WriteString(cb.NotificationID.String())
	b.WriteString(`"}`)
	return b.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (cb *Callback) UnmarshalJSON(b []byte) error {
	var msg cbMessage
	msg.ID = &cb.NotificationID
	if err := json.Unmarshal(b, &msg); err != nil {
		return err
	}
	u, err := url.Parse(msg.Callback)
	if err != nil {
		return fmt.Errorf("json unmarshal failed. malformed callback url: %v", err)
	}
	cb.Callback = *u
	return nil
}
