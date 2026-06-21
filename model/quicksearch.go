package model

// QuicksearchResult is one typeahead hit — a location or an SRP filter.
type QuicksearchResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "cafe"|"poi"|"area"|"district"|"filter"
	// Slug is the ready-to-use navigation target the frontend links to:
	//   - filter           -> the SRP filter slug ("quiet-noise")
	//   - district/area/poi -> the canonical /explore splat ("bandung-tengah/dago/<poi>"),
	//                          empty when the ancestor chain is incomplete (frontend falls back)
	//   - cafe             -> empty (routed by id to /cafe/<id>)
	Slug string `json:"slug,omitempty"`
}
