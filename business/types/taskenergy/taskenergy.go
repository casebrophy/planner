package taskenergy

import "fmt"

type Energy struct {
	value string
}

var (
	Low    = Energy{"low"}
	Medium = Energy{"medium"}
	High   = Energy{"high"}
)

var energies = map[string]Energy{
	Low.value:    Low,
	Medium.value: Medium,
	High.value:   High,
}

func Parse(s string) (Energy, error) {
	e, ok := energies[s]
	if !ok {
		return Energy{}, fmt.Errorf("invalid task energy %q", s)
	}
	return e, nil
}

func MustParse(s string) Energy {
	e, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return e
}

func (e Energy) String() string {
	return e.value
}

func (e Energy) MarshalText() ([]byte, error) {
	return []byte(e.value), nil
}

func (e *Energy) UnmarshalText(data []byte) error {
	en, err := Parse(string(data))
	if err != nil {
		return err
	}
	*e = en
	return nil
}

// EqualString compares the energy to a raw string without parsing.
func (e Energy) EqualString(v string) bool {
	return e.value == v
}
