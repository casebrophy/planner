// Package recurrence provides a simplified RRULE-like recurrence rule type
// supporting DAILY, WEEKLY, and MONTHLY frequencies.
package recurrence

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Freq represents a recurrence frequency.
type Freq string

const (
	Daily   Freq = "DAILY"
	Weekly  Freq = "WEEKLY"
	Monthly Freq = "MONTHLY"
)

// Weekday constants matching the BYDAY abbreviations.
var weekdayMap = map[string]time.Weekday{
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
	"SU": time.Sunday,
}

var weekdayAbbr = map[time.Weekday]string{
	time.Monday:    "MO",
	time.Tuesday:   "TU",
	time.Wednesday: "WE",
	time.Thursday:  "TH",
	time.Friday:    "FR",
	time.Saturday:  "SA",
	time.Sunday:    "SU",
}

// Rule represents a recurrence rule with frequency, interval, and optional
// day-of-week or day-of-month constraints.
type Rule struct {
	Freq       Freq
	Interval   int
	ByDay      []time.Weekday
	ByMonthDay int // 0 means unset
}

// Parse parses a recurrence rule string like "FREQ=WEEKLY;BYDAY=MO,WE,FR".
func Parse(s string) (Rule, error) {
	if s == "" {
		return Rule{}, fmt.Errorf("empty recurrence rule")
	}

	r := Rule{Interval: 1}
	parts := strings.Split(s, ";")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return Rule{}, fmt.Errorf("invalid recurrence part: %q", part)
		}
		key, val := kv[0], kv[1]

		switch key {
		case "FREQ":
			switch Freq(val) {
			case Daily, Weekly, Monthly:
				r.Freq = Freq(val)
			default:
				return Rule{}, fmt.Errorf("unsupported frequency: %q", val)
			}

		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return Rule{}, fmt.Errorf("invalid interval: %q", val)
			}
			r.Interval = n

		case "BYDAY":
			days := strings.Split(val, ",")
			for _, d := range days {
				wd, ok := weekdayMap[d]
				if !ok {
					return Rule{}, fmt.Errorf("invalid day: %q", d)
				}
				r.ByDay = append(r.ByDay, wd)
			}

		case "BYMONTHDAY":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 || n > 31 {
				return Rule{}, fmt.Errorf("invalid month day: %q", val)
			}
			r.ByMonthDay = n

		default:
			return Rule{}, fmt.Errorf("unsupported recurrence key: %q", key)
		}
	}

	if r.Freq == "" {
		return Rule{}, fmt.Errorf("FREQ is required")
	}

	return r, nil
}

// NextOccurrence computes the next occurrence after the given time.
func (r Rule) NextOccurrence(from time.Time) time.Time {
	switch r.Freq {
	case Daily:
		return from.AddDate(0, 0, r.Interval)

	case Weekly:
		if len(r.ByDay) == 0 {
			return from.AddDate(0, 0, 7*r.Interval)
		}
		// Find the next matching weekday. Walk forward day by day,
		// but respect the interval (every N weeks).
		return r.nextWeekday(from)

	case Monthly:
		if r.ByMonthDay > 0 {
			return r.nextMonthDay(from)
		}
		return from.AddDate(0, r.Interval, 0)

	default:
		return from.AddDate(0, 0, 1)
	}
}

// nextWeekday finds the next matching weekday from the ByDay list.
// For interval=1, it finds the next day in the current or following week.
// For interval>1, after exhausting days in the current week it jumps ahead.
func (r Rule) nextWeekday(from time.Time) time.Time {
	daySet := make(map[time.Weekday]bool, len(r.ByDay))
	for _, d := range r.ByDay {
		daySet[d] = true
	}

	// First, try remaining days in the current week (after from).
	for i := 1; i <= 7; i++ {
		candidate := from.AddDate(0, 0, i)
		if daySet[candidate.Weekday()] {
			// If interval is 1, any match in the next 7 days works.
			if r.Interval == 1 {
				return candidate
			}
			// For interval > 1, only accept if still in the same week as from.
			_, fromWeek := from.ISOWeek()
			_, candWeek := candidate.ISOWeek()
			if candWeek == fromWeek {
				return candidate
			}
		}
	}

	// Jump to the start of the target week (interval weeks ahead).
	// Find Monday of from's week, then add interval weeks.
	daysToMonday := (int(from.Weekday()) - int(time.Monday) + 7) % 7
	monday := from.AddDate(0, 0, -daysToMonday)
	targetMonday := monday.AddDate(0, 0, 7*r.Interval)

	// Find the first matching day in that week.
	for i := 0; i < 7; i++ {
		candidate := targetMonday.AddDate(0, 0, i)
		if daySet[candidate.Weekday()] {
			return candidate
		}
	}

	// Fallback (shouldn't happen if ByDay is non-empty).
	return from.AddDate(0, 0, 7*r.Interval)
}

// nextMonthDay finds the next occurrence of the specified day of month.
func (r Rule) nextMonthDay(from time.Time) time.Time {
	year, month, _ := from.Date()
	loc := from.Location()

	// Try the target day in the current month (if it's still ahead).
	candidate := time.Date(year, month, r.ByMonthDay, from.Hour(), from.Minute(), from.Second(), 0, loc)
	if candidate.After(from) && candidate.Month() == month {
		return candidate
	}

	// Move to the next interval month.
	target := time.Date(year, month+time.Month(r.Interval), r.ByMonthDay, from.Hour(), from.Minute(), from.Second(), 0, loc)

	// Handle months where the day doesn't exist (e.g., Feb 31 -> Mar 3).
	// Clamp to last day of the target month if needed.
	expectedMonth := (month-1+time.Month(r.Interval))%12 + 1
	if r.Interval >= 12 {
		expectedMonth = month
	}
	if target.Month() != expectedMonth {
		// Day overflowed; use last day of the expected month.
		target = time.Date(target.Year(), expectedMonth+1, 0, from.Hour(), from.Minute(), from.Second(), 0, loc)
	}

	return target
}

// String serializes the rule back to the text format.
func (r Rule) String() string {
	var parts []string
	parts = append(parts, "FREQ="+string(r.Freq))

	if r.Interval > 1 {
		parts = append(parts, "INTERVAL="+strconv.Itoa(r.Interval))
	}

	if len(r.ByDay) > 0 {
		days := make([]string, len(r.ByDay))
		for i, d := range r.ByDay {
			days[i] = weekdayAbbr[d]
		}
		parts = append(parts, "BYDAY="+strings.Join(days, ","))
	}

	if r.ByMonthDay > 0 {
		parts = append(parts, "BYMONTHDAY="+strconv.Itoa(r.ByMonthDay))
	}

	return strings.Join(parts, ";")
}

// MarshalText implements encoding.TextMarshaler.
func (r Rule) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (r *Rule) UnmarshalText(data []byte) error {
	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
