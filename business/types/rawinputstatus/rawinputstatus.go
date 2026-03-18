package rawinputstatus

import "fmt"

type Status struct {
	value string
}

var (
	Pending    = Status{"pending"}
	Processing = Status{"processing"}
	Processed  = Status{"processed"}
	Failed     = Status{"failed"}
)

var statuses = map[string]Status{
	Pending.value:    Pending,
	Processing.value: Processing,
	Processed.value:  Processed,
	Failed.value:     Failed,
}

func Parse(s string) (Status, error) {
	st, ok := statuses[s]
	if !ok {
		return Status{}, fmt.Errorf("invalid raw input status %q", s)
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
