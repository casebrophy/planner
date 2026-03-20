package contextoutcome

import "fmt"

type Outcome struct {
	value string
}

var (
	WentWell      = Outcome{"went_well"}
	Mixed         = Outcome{"mixed"}
	Difficult     = Outcome{"difficult"}
	OngoingIssues = Outcome{"ongoing_issues"}
)

var outcomes = map[string]Outcome{
	WentWell.value:      WentWell,
	Mixed.value:         Mixed,
	Difficult.value:     Difficult,
	OngoingIssues.value: OngoingIssues,
}

func Parse(s string) (Outcome, error) {
	o, ok := outcomes[s]
	if !ok {
		return Outcome{}, fmt.Errorf("invalid context outcome %q", s)
	}
	return o, nil
}

func MustParse(s string) Outcome {
	o, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return o
}

func (o Outcome) String() string {
	return o.value
}

func (o Outcome) MarshalText() ([]byte, error) {
	return []byte(o.value), nil
}

func (o *Outcome) UnmarshalText(data []byte) error {
	out, err := Parse(string(data))
	if err != nil {
		return err
	}
	*o = out
	return nil
}

func (o Outcome) EqualString(v string) bool {
	return o.value == v
}
