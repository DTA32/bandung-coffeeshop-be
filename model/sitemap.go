package model

// SitemapResponse is the raw material for the FE's /sitemap.xml: facts, not
// URLs. The FE owns slugs, canonical segment order and the indexability rule.
type SitemapResponse struct {
	Cafes []SitemapCafe `json:"cafes"`
	SRPs  []SitemapSRP  `json:"srps"`
}

type SitemapCafe struct {
	ID        string `json:"id"`
	Status    string `json:"status"`     // "active" or "closed"
	UpdatedAt string `json:"updated_at"` // ISO-8601 UTC
}

// SitemapSRP is one location × filter combination and how many active cafes
// it matches — the same number GET /v1/search/cafes reports as total.
type SitemapSRP struct {
	Locations    []string       `json:"locations"` // [district, area?, poi?]; empty for a filter-only SRP
	Filter       *SitemapFilter `json:"filter"`    // nil for a location-only SRP
	Count        int            `json:"count"`
	LastModified string         `json:"last_modified"` // newest updated_at among the matched cafes
}

type SitemapFilter struct {
	Type string `json:"type"` // "tag", "price" or "rating"
	Slug string `json:"slug"` // the SRP slug /v1/filters returns
}
