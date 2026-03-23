package guard

import "context"

type Action string

const (
	ActionPass  Action = "pass"
	ActionWarn  Action = "warn"
	ActionBlock Action = "block"
	ActionMask  Action = "mask"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type Content struct {
	Direction Direction
	Body      string
	Messages  []Message
	Metadata  map[string]string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Result struct {
	GuardName string    `json:"guard_name"`
	Action    Action    `json:"action"`
	Blocked   bool      `json:"blocked"`
	Details   string    `json:"details,omitempty"`
	Findings  []Finding `json:"findings,omitempty"`
	Modified  string    `json:"modified,omitempty"`
}

type Finding struct {
	Type       string `json:"type"`
	Value      string `json:"value"`
	Location   int    `json:"location"`
	Length     int    `json:"length"`
	Confidence float64 `json:"confidence"`
}

type Guard interface {
	Name() string
	Check(ctx context.Context, content *Content) (*Result, error)
}
