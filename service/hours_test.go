package service

import (
	"reflect"
	"testing"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
)

func rng(open, close string) model.HourRange { return model.HourRange{Open: open, Close: close} }

// row builds a cafe_hour row; "" leaves a field NULL.
func row(day, open, close, from, until string, closed bool) repository.CafeHourRow {
	opt := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	return repository.CafeHourRow{
		Day: opt(day), OpenHour: opt(open), CloseHour: opt(close),
		ValidFrom: opt(from), ValidUntil: opt(until), Close: closed,
	}
}

// Saturday 2026-10-24 in Jakarta; the window runs sat 24 → fri 30.
var testToday = time.Date(2026, 10, 24, 23, 30, 0, 0, jakarta)

var testWeek = [7][2]string{
	{"2026-10-24", "sat"}, {"2026-10-25", "sun"}, {"2026-10-26", "mon"}, {"2026-10-27", "tue"},
	{"2026-10-28", "wed"}, {"2026-10-29", "thu"}, {"2026-10-30", "fri"},
}

// week returns the 7-day schedule with every day set to def, then applies the
// per-index overrides.
func week(def model.CafeDayHours, overrides map[int]model.CafeDayHours) []model.CafeDayHours {
	out := make([]model.CafeDayHours, 7)
	for i, d := range testWeek {
		h := def
		if o, ok := overrides[i]; ok {
			h = o
		}
		h.Date, h.Day = d[0], d[1]
		if h.Ranges == nil {
			h.Ranges = []model.HourRange{}
		}
		out[i] = h
	}
	return out
}

func open(ranges ...model.HourRange) model.CafeDayHours { return model.CafeDayHours{Ranges: ranges} }

func TestResolveHours(t *testing.T) {
	unknown := model.CafeDayHours{}
	closed := model.CafeDayHours{Closed: true}
	regular := open(rng("08:00", "22:00"))

	tests := []struct {
		name string
		rows []repository.CafeHourRow
		want []model.CafeDayHours
	}{
		{
			name: "no rows is unknown every day",
			rows: nil,
			want: week(unknown, nil),
		},
		{
			name: "every-day row (tier c)",
			rows: []repository.CafeHourRow{row("", "08:00", "22:00", "", "", false)},
			want: week(regular, nil),
		},
		{
			name: "weekday row (tier b) beats every-day row (tier c)",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("mon", "", "", "", "", true),
				row("sat", "10:00", "23:00", "", "", false),
			},
			want: week(regular, map[int]model.CafeDayHours{
				0: open(rng("10:00", "23:00")),
				2: closed,
			}),
		},
		{
			name: "single-day dated row (tier a) beats regular hours",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("sun", "09:00", "21:00", "", "", false),
				row("", "", "", "2026-10-25", "2026-10-25", true),
			},
			want: week(regular, map[int]model.CafeDayHours{
				1: {Closed: true, Special: true},
			}),
		},
		{
			name: "dated row with day beats dated row with null day",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("", "10:00", "16:00", "2026-10-24", "2026-10-26", false),
				row("sun", "", "", "2026-10-24", "2026-10-26", true),
				// Wrong weekday for every date it covers: never applies.
				row("tue", "01:00", "02:00", "2026-10-24", "2026-10-26", false),
			},
			want: week(regular, map[int]model.CafeDayHours{
				0: {Special: true, Ranges: []model.HourRange{rng("10:00", "16:00")}},
				1: {Closed: true, Special: true},
				2: {Special: true, Ranges: []model.HourRange{rng("10:00", "16:00")}},
			}),
		},
		{
			name: "dated row outside the window is ignored",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("", "", "", "2026-10-31", "2026-11-02", true),
				row("", "", "", "2026-10-01", "2026-10-23", true),
			},
			want: week(regular, nil),
		},
		{
			name: "split ranges are sorted by open",
			rows: []repository.CafeHourRow{
				row("sat", "17:00", "22:00", "", "", false),
				row("sat", "08:00", "14:00", "", "", false),
			},
			want: week(unknown, map[int]model.CafeDayHours{
				0: open(rng("08:00", "14:00"), rng("17:00", "22:00")),
			}),
		},
		{
			name: "closed wins over open rows in the same tier",
			rows: []repository.CafeHourRow{
				row("sat", "08:00", "14:00", "", "", false),
				row("sat", "", "", "", "", true),
			},
			want: week(unknown, map[int]model.CafeDayHours{0: closed}),
		},
		{
			name: "null hours in the winning tier are unknown, not regular",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("mon", "", "", "", "", false),
			},
			want: week(regular, map[int]model.CafeDayHours{2: unknown}),
		},
		{
			name: "overnight and 24 hours pass through as stored",
			rows: []repository.CafeHourRow{
				row("", "00:00", "23:59", "", "", false),
				row("fri", "18:00", "02:00", "", "", false),
			},
			want: week(open(rng("00:00", "23:59")), map[int]model.CafeDayHours{
				6: open(rng("18:00", "02:00")),
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveHours(tt.rows, testToday)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveHours()\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveRegularHours(t *testing.T) {
	reg := func(day string, closed bool, ranges ...model.HourRange) model.CafeRegularHours {
		if ranges == nil {
			ranges = []model.HourRange{}
		}
		return model.CafeRegularHours{Day: day, Closed: closed, Ranges: ranges}
	}

	tests := []struct {
		name string
		rows []repository.CafeHourRow
		want []model.CafeRegularHours
	}{
		{
			name: "no rows is seven unknown days",
			rows: nil,
			want: []model.CafeRegularHours{
				reg("mon", false), reg("tue", false), reg("wed", false), reg("thu", false),
				reg("fri", false), reg("sat", false), reg("sun", false),
			},
		},
		{
			name: "tiers b over c, split and closed; dated rows ignored",
			rows: []repository.CafeHourRow{
				row("", "08:00", "22:00", "", "", false),
				row("mon", "", "", "", "", true),
				row("sat", "17:00", "23:00", "", "", false),
				row("sat", "09:00", "14:00", "", "", false),
				row("", "", "", "2026-10-24", "2026-10-30", true),
				row("tue", "10:00", "11:00", "2026-10-27", "2026-10-27", false),
			},
			want: []model.CafeRegularHours{
				reg("mon", true),
				reg("tue", false, rng("08:00", "22:00")),
				reg("wed", false, rng("08:00", "22:00")),
				reg("thu", false, rng("08:00", "22:00")),
				reg("fri", false, rng("08:00", "22:00")),
				reg("sat", false, rng("09:00", "14:00"), rng("17:00", "23:00")),
				reg("sun", false, rng("08:00", "22:00")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRegularHours(tt.rows)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveRegularHours()\n got  %+v\n want %+v", got, tt.want)
			}
		})
	}
}
