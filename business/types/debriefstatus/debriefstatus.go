package debriefstatus

import "fmt"

type Status struct {
	value string
}

var (
	Pending = Status{"pending"}
	Done    = Status{"done"}
	Skipped = Status{"skipped"}
)

var statuses = map[string]Status{
	Pending.value: Pending,
	Done.value:    Done,
	Skipped.value: Skipped,
}

func Parse(s string) (Status, error) {
	st, ok := statuses[s]
	if !ok {
		return Status{}, fmt.Errorf("invalid debrief status %q", s)
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

func (s Status) EqualString(v string) bool {
	return s.value == v
}
