package service

import (
	"slices"
	"strings"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

// hoursDays is how many days GET /v1/cafe/:id resolves, starting today.
const hoursDays = 7

// weekdays is mon→sun, the order of regular_hours; dayNames maps time.Weekday
// to the day_enum value.
var (
	weekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}
	dayNames = [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}
)

// Tiers, most specific first (CLAUDE.md "Opening hours"). Dated rows are one
// tier where a matching day beats a null day.
const (
	tierNone = iota
	tierDatedDay
	tierDatedAny
	tierDay
	tierAny
)

// hourTier returns the tier of row h for a date (YYYY-MM-DD) on weekday day,
// or tierNone when h does not apply to it.
func hourTier(h repository.CafeHourRow, date, day string) int {
	if h.Day != nil && *h.Day != day {
		return tierNone
	}
	if h.ValidFrom == nil && h.ValidUntil == nil {
		if h.Day != nil {
			return tierDay
		}
		return tierAny
	}
	// Dates are YYYY-MM-DD, so they compare as strings. A null bound is open.
	if (h.ValidFrom != nil && *h.ValidFrom > date) || (h.ValidUntil != nil && *h.ValidUntil < date) {
		return tierNone
	}
	if h.Day != nil {
		return tierDatedDay
	}
	return tierDatedAny
}

// resolveDay applies the rows of the most specific tier present for a date.
// dated = false skips the dated tiers (regular hours). Closed wins over any
// open row in the same tier; rows with a NULL open or close are ignored, and
// no usable row means unknown hours (not closed, no ranges).
func resolveDay(rows []repository.CafeHourRow, date, day string, dated bool) (closed, special bool, ranges []model.HourRange) {
	best := tierNone
	for _, h := range rows {
		t := hourTier(h, date, day)
		if t == tierNone || (!dated && t < tierDay) {
			continue
		}
		if best == tierNone || t < best {
			best = t
		}
	}
	ranges = []model.HourRange{}
	if best == tierNone {
		return false, false, ranges
	}
	special = best == tierDatedDay || best == tierDatedAny
	for _, h := range rows {
		if hourTier(h, date, day) != best {
			continue
		}
		if h.Close {
			return true, special, []model.HourRange{}
		}
		if h.OpenHour != nil && h.CloseHour != nil {
			ranges = append(ranges, model.HourRange{Open: *h.OpenHour, Close: *h.CloseHour})
		}
	}
	slices.SortFunc(ranges, func(a, b model.HourRange) int {
		if c := strings.Compare(a.Open, b.Open); c != 0 {
			return c
		}
		return strings.Compare(a.Close, b.Close)
	})
	return false, special, ranges
}

// ResolveHours resolves a cafe's cafe_hour rows into its schedule for the 7
// days starting today (a Jakarta time; only its date is used).
func ResolveHours(rows []repository.CafeHourRow, today time.Time) []model.CafeDayHours {
	out := make([]model.CafeDayHours, 0, hoursDays)
	for i := range hoursDays {
		d := today.AddDate(0, 0, i)
		date, day := d.Format(time.DateOnly), dayNames[d.Weekday()]
		closed, special, ranges := resolveDay(rows, date, day, true)
		out = append(out, model.CafeDayHours{Date: date, Day: day, Closed: closed, Special: special, Ranges: ranges})
	}
	return out
}

// ResolveRegularHours resolves the regular week, mon→sun, from undated rows
// only (dated rows are holidays / special events).
func ResolveRegularHours(rows []repository.CafeHourRow) []model.CafeRegularHours {
	out := make([]model.CafeRegularHours, 0, len(weekdays))
	for _, day := range weekdays {
		closed, _, ranges := resolveDay(rows, "", day, false)
		out = append(out, model.CafeRegularHours{Day: day, Closed: closed, Ranges: ranges})
	}
	return out
}
