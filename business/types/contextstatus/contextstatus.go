package contextstatus

import "fmt"

type Status struct {
	value string
}

var (
	Active = Status{"active"}
	Paused = Status{"paused"}
	Closed = Status{"closed"}
)

var statuses = map[string]Status{
	Active.value: Active,
	Paused.value: Paused,
	Closed.value: Closed,
}

func Parse(s string) (Status, error) {
	st, ok := statuses[s]
	if !ok {
		return Status{}, fmt.Errorf("invalid context status %q", s)
	}
	return st, nil
}

func MustParse(s string) Status {
	st, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return st
}

func (s Status) String() string {
	return s.value
}

func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

func (s *Status) UnmarshalText(data []byte) error {
	st, err := Parse(string(data))
	if err != nil {
		return err
	}
	*s = st
	return nil
}

// EqualString compares the status to a raw string without parsing.
func (s Status) EqualString(v string) bool {
	return s.value == v
}
