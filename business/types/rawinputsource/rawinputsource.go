package rawinputsource

import "fmt"

type Source struct {
	value string
}

var (
	Email       = Source{"email"}
	Transaction = Source{"transaction"}
	Voice       = Source{"voice"}
	File        = Source{"file"}
)

var sources = map[string]Source{
	Email.value:       Email,
	Transaction.value: Transaction,
	Voice.value:       Voice,
	File.value:        File,
}

func Parse(s string) (Source, error) {
	src, ok := sources[s]
	if !ok {
		return Source{}, fmt.Errorf("invalid raw input source %q", s)
	}
	return src, nil
}

func MustParse(s string) Source {
	src, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return src
}

func (s Source) String() string {
	return s.value
}

func (s Source) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

func (s *Source) UnmarshalText(data []byte) error {
	src, err := Parse(string(data))
	if err != nil {
		return err
	}
	*s = src
	return nil
}

func (s Source) EqualString(v string) bool {
	return s.value == v
}
