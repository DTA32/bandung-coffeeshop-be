package helper

import "strings"

// SplitCSV splits a comma-separated value into trimmed, non-empty items.
func SplitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
