package gapcategory

import "fmt"

type Category struct {
	value string
}

var (
	MissingContact      = Category{"missing_contact"}
	MissingLocation     = Category{"missing_location"}
	MissingContext      = Category{"missing_context"}
	MissingDependency   = Category{"missing_dependency"}
	MissingDetail       = Category{"missing_detail"}
	MissingDeadline     = Category{"missing_deadline"}
	MissingStakeholder  = Category{"missing_stakeholder"}
	MissingOutcome      = Category{"missing_outcome"}
)

var categories = map[string]Category{
	MissingContact.value:     MissingContact,
	MissingLocation.value:    MissingLocation,
	MissingContext.value:     MissingContext,
	MissingDependency.value:  MissingDependency,
	MissingDetail.value:      MissingDetail,
	MissingDeadline.value:    MissingDeadline,
	MissingStakeholder.value: MissingStakeholder,
	MissingOutcome.value:     MissingOutcome,
}

// AllCategories is the exhaustive list of all valid gap categories.
// Used by codegen (api/tooling/gen-ts-kinds) to generate TypeScript types.
var AllCategories = []Category{
	MissingContact,
	MissingLocation,
	MissingContext,
	MissingDependency,
	MissingDetail,
	MissingDeadline,
	MissingStakeholder,
	MissingOutcome,
}

func Parse(s string) (Category, error) {
	c, ok := categories[s]
	if !ok {
		return Category{}, fmt.Errorf("invalid gap category %q", s)
	}
	return c, nil
}

func MustParse(s string) Category {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

func (c Category) String() string {
	return c.value
}

func (c Category) MarshalText() ([]byte, error) {
	return []byte(c.value), nil
}

func (c *Category) UnmarshalText(data []byte) error {
	cc, err := Parse(string(data))
	if err != nil {
		return err
	}
	*c = cc
	return nil
}

func (c Category) EqualString(v string) bool {
	return c.value == v
}
