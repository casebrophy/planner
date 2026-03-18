package order

import (
	"fmt"
	"strings"
)

const (
	ASC  = "ASC"
	DESC = "DESC"
)

type By struct {
	Field     string
	Direction string
}

func NewBy(field, direction string) By {
	if direction == "" {
		direction = ASC
	}
	return By{
		Field:     field,
		Direction: direction,
	}
}

func Parse(fieldMappings map[string]string, orderByStr string, defaultOrder By) (By, error) {
	if orderByStr == "" {
		return defaultOrder, nil
	}

	parts := strings.Split(orderByStr, ",")

	fieldName := strings.TrimSpace(parts[0])
	if _, ok := fieldMappings[fieldName]; !ok {
		return By{}, fmt.Errorf("unknown order field %q", fieldName)
	}

	direction := ASC
	if len(parts) > 1 {
		dir := strings.ToUpper(strings.TrimSpace(parts[1]))
		switch dir {
		case ASC, DESC:
			direction = dir
		default:
			return By{}, fmt.Errorf("unknown order direction %q", dir)
		}
	}

	return By{
		Field:     fieldName,
		Direction: direction,
	}, nil
}
