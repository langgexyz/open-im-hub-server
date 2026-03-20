package push

import "context"

const (
	PlatformIOS     = 1
	PlatformAndroid = 2
)

type Message struct {
	Token    string
	Platform int8
	Title    string
	Body     string
	Data     map[string]any
}

type Pusher interface {
	Send(ctx context.Context, msg Message) error
}

type NoopPusher struct{}

func (NoopPusher) Send(_ context.Context, _ Message) error { return nil }
