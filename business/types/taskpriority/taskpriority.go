package taskpriority

import "fmt"

type Priority struct {
	value string
}

var (
	Low    = Priority{"low"}
	Medium = Priority{"medium"}
	High   = Priority{"high"}
	Urgent = Priority{"urgent"}
)

var priorities = map[string]Priority{
	Low.value:    Low,
	Medium.value: Medium,
	High.value:   High,
	Urgent.value: Urgent,
}

func Parse(s string) (Priority, error) {
	p, ok := priorities[s]
	if !ok {
		return Priority{}, fmt.Errorf("invalid task priority %q", s)
	}
	return p, nil
}

func MustParse(s string) Priority {
	p, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Priority) String() string {
	return p.value
}

func (p Priority) MarshalText() ([]byte, error) {
	return []byte(p.value), nil
}

func (p *Priority) UnmarshalText(data []byte) error {
	pr, err := Parse(string(data))
	if err != nil {
		return err
	}
	*p = pr
	return nil
}

// EqualString compares the priority to a raw string without parsing.
func (p Priority) EqualString(v string) bool {
	return p.value == v
}
