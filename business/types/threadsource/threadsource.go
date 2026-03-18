package threadsource

import "fmt"

type Source struct {
	value string
}

var (
	User              = Source{"user"}
	Voice             = Source{"voice"}
	EmailSource       = Source{"email"}
	TransactionSource = Source{"transaction"}
	System            = Source{"system"}
	Claude            = Source{"claude"}
)

var sources = map[string]Source{
	User.value:              User,
	Voice.value:             Voice,
	EmailSource.value:       EmailSource,
	TransactionSource.value: TransactionSource,
	System.value:            System,
	Claude.value:            Claude,
}

func Parse(s string) (Source, error) {
	src, ok := sources[s]
	if !ok {
		return Source{}, fmt.Errorf("invalid thread source %q", s)
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
