package observationkind

import "fmt"

type Kind struct {
	value string
}

var (
	DurationAccuracy  = Kind{"duration_accuracy"}
	BlockerProfile    = Kind{"blocker_profile"}
	TimelineProfile   = Kind{"timeline_profile"}
	Lesson            = Kind{"lesson"}
	CompletionPattern = Kind{"completion_pattern"}
	ScopeChange       = Kind{"scope_change"}
	CostProfile       = Kind{"cost_profile"}
	Debrief           = Kind{"debrief"}
)

var kinds = map[string]Kind{
	DurationAccuracy.value:  DurationAccuracy,
	BlockerProfile.value:    BlockerProfile,
	TimelineProfile.value:   TimelineProfile,
	Lesson.value:            Lesson,
	CompletionPattern.value: CompletionPattern,
	ScopeChange.value:       ScopeChange,
	CostProfile.value:       CostProfile,
	Debrief.value:           Debrief,
}

func Parse(s string) (Kind, error) {
	k, ok := kinds[s]
	if !ok {
		return Kind{}, fmt.Errorf("invalid observation kind %q", s)
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
