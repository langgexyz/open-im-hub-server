package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FCMPusher 使用 FCM Legacy HTTP API
type FCMPusher struct {
	serverKey string
	http      *http.Client
}

func NewFCMPusher(serverKey string) *FCMPusher {
	return &FCMPusher{serverKey: serverKey, http: &http.Client{Timeout: 10 * time.Second}}
}

func (p *FCMPusher) Send(ctx context.Context, msg Message) error {
	payload, _ := json.Marshal(map[string]any{
		"to":           msg.Token,
		"notification": map[string]any{"title": msg.Title, "body": msg.Body},
		"data":         msg.Data,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fcm.googleapis.com/fcm/send", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+p.serverKey)
	resp, err := p.http.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fcm: HTTP %d", resp.StatusCode)
	}
	return nil
}
