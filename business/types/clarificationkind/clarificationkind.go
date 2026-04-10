package clarificationkind

import "fmt"

type Kind struct {
	value string
}

var (
	ContextAssignment   = Kind{"context_assignment"}
	StaleTask           = Kind{"stale_task"}
	AmbiguousDeadline   = Kind{"ambiguous_deadline"}
	NewContext          = Kind{"new_context"}
	OverlappingContexts = Kind{"overlapping_contexts"}
	AmbiguousAction     = Kind{"ambiguous_action"}
	VoiceReference      = Kind{"voice_reference"}
	InactivityPrompt    = Kind{"inactivity_prompt"}
	ContextDebrief      = Kind{"context_debrief"}
	TaskDebrief         = Kind{"task_debrief"}
	EntityLink          = Kind{"entity_link"}
	WeeklyReview        = Kind{"weekly_review"}
	TypeAssignment      = Kind{"type_assignment"}
)

var kinds = map[string]Kind{
	ContextAssignment.value:   ContextAssignment,
	StaleTask.value:           StaleTask,
	AmbiguousDeadline.value:   AmbiguousDeadline,
	NewContext.value:          NewContext,
	OverlappingContexts.value: OverlappingContexts,
	AmbiguousAction.value:     AmbiguousAction,
	VoiceReference.value:      VoiceReference,
	InactivityPrompt.value:    InactivityPrompt,
	ContextDebrief.value:      ContextDebrief,
	TaskDebrief.value:         TaskDebrief,
	EntityLink.value:          EntityLink,
	WeeklyReview.value:        WeeklyReview,
	TypeAssignment.value:      TypeAssignment,
}

// AllKinds is the exhaustive list of all valid clarification kinds.
// Used by codegen (api/tooling/gen-ts-kinds) to generate TypeScript types.
var AllKinds = []Kind{
	ContextAssignment,
	StaleTask,
	AmbiguousDeadline,
	NewContext,
	OverlappingContexts,
	AmbiguousAction,
	VoiceReference,
	InactivityPrompt,
	ContextDebrief,
	TaskDebrief,
	EntityLink,
	WeeklyReview,
	TypeAssignment,
}

// KindWeights maps each kind to its priority weight for scoring.
var KindWeights = map[Kind]float32{
	ContextAssignment:   0.7,
	StaleTask:           0.6,
	AmbiguousDeadline:   0.5,
	NewContext:          0.9,
	OverlappingContexts: 0.6,
	AmbiguousAction:     0.8,
	VoiceReference:      0.7,
	InactivityPrompt:    0.6,
	ContextDebrief:      0.8,
	TaskDebrief:         0.9,
	EntityLink:          0.7,
	WeeklyReview:        0.8,
	TypeAssignment:      0.8,
}

func Parse(s string) (Kind, error) {
	k, ok := kinds[s]
	if !ok {
		return Kind{}, fmt.Errorf("invalid clarification kind %q", s)
	}
	return k, nil
}

func MustParse(s string) Kind {
	k, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return k
}

func (k Kind) String() string {
	return k.value
}

func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.value), nil
}

func (k *Kind) UnmarshalText(data []byte) error {
	kk, err := Parse(string(data))
	if err != nil {
		return err
	}
	*k = kk
	return nil
}

func (k Kind) EqualString(v string) bool {
	return k.value == v
}
