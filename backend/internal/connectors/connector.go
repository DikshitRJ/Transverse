package connectors

import "context"

type RawSignal struct {
	Languages     map[string]float64
	ClaimedTopics []string
	Signals       []Signal
}

type Signal struct {
	TopicTag string
	Evidence string
	Strength string // "weak", "moderate", "strong"
}

type Connector interface {
	Fetch(ctx context.Context, ref string) (*RawSignal, error)
}
