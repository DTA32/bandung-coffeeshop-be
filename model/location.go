package model

type Location struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type QuicksearchResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	// Ancestors lists containing locations, outermost to innermost, excluding
	// the result itself (area -> [district]; poi -> [district, area]).
	Ancestors []Location `json:"ancestors"`
}
