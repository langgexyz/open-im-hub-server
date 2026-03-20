package push

import (
	"context"
	"fmt"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

type APNsPusher struct {
	client   *apns2.Client
	bundleID string
}

func NewAPNsPusher(keyFile, keyID, teamID, bundleID string, sandbox bool) (*APNsPusher, error) {
	authKey, err := token.AuthKeyFromFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("load apns key: %w", err)
	}
	t := &token.Token{AuthKey: authKey, KeyID: keyID, TeamID: teamID}
	client := apns2.NewTokenClient(t)
	if sandbox {
		client = client.Development()
	} else {
		client = client.Production()
	}
	return &APNsPusher{client: client, bundleID: bundleID}, nil
}

func (p *APNsPusher) Send(_ context.Context, msg Message) error {
	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]any{"title": msg.Title, "body": msg.Body},
			"sound": "default",
		},
	}
	for k, v := range msg.Data {
		payload[k] = v
	}
	res, err := p.client.Push(&apns2.Notification{
		DeviceToken: msg.Token,
		Topic:       p.bundleID,
		Payload:     payload,
	})
	if err != nil {
		return fmt.Errorf("apns: %w", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("apns: %s", res.Reason)
	}
	return nil
}
