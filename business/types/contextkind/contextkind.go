package contextkind

import "fmt"

type Kind struct {
	value string
}

var (
	Project = Kind{"project"}
	Area    = Kind{"area"}
)

var kinds = map[string]Kind{
	Project.value: Project,
	Area.value:    Area,
}

func Parse(s string) (Kind, error) {
	k, ok := kinds[s]
	if !ok {
		return Kind{}, fmt.Errorf("invalid context kind %q", s)
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

// EqualString compares the kind to a raw string without parsing.
func (k Kind) EqualString(v string) bool {
	return k.value == v
}
