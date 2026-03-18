package threadentrykind

import "fmt"

type Kind struct {
	value string
}

var (
	Update          = Kind{"update"}
	Blocker         = Kind{"blocker"}
	BlockerResolved = Kind{"blocker_resolved"}
	Milestone       = Kind{"milestone"}
	ScopeChange     = Kind{"scope_change"}
	TimelineSlip    = Kind{"timeline_slip"}
	ExternalDep     = Kind{"external_dep"}
	Decision        = Kind{"decision"}
	Observation     = Kind{"observation"}
	Email           = Kind{"email"}
	Transaction     = Kind{"transaction"}
)

var kinds = map[string]Kind{
	Update.value:          Update,
	Blocker.value:         Blocker,
	BlockerResolved.value: BlockerResolved,
	Milestone.value:       Milestone,
	ScopeChange.value:     ScopeChange,
	TimelineSlip.value:    TimelineSlip,
	ExternalDep.value:     ExternalDep,
	Decision.value:        Decision,
	Observation.value:     Observation,
	Email.value:           Email,
	Transaction.value:     Transaction,
}

func Parse(s string) (Kind, error) {
	k, ok := kinds[s]
	if !ok {
		return Kind{}, fmt.Errorf("invalid thread entry kind %q", s)
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
