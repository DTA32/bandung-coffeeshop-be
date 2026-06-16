package model

// FiltersResponse powers the explore filter modal: the selectable tags, the
// rating categories (grouped by type, each with its buckets), and the hardcoded
// price tiers. All human-readable text is localized via Accept-Language.
type FiltersResponse struct {
	Tags             []FilterTag            `json:"tags"`
	RatingCategories []FilterRatingCategory `json:"rating_categories"`
	PriceTiers       []FilterPriceTier      `json:"price_tiers"`
}

type FilterTag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type FilterRatingCategory struct {
	Type        string               `json:"type"`         // enum value, e.g. "vibe"
	DisplayName string               `json:"display_name"` // localized type label
	Options     []FilterRatingOption `json:"options"`
}

type FilterRatingOption struct {
	ID          int     `json:"id"`          // FE selects buckets by this id
	Name        string  `json:"name"`        // localized
	Description string  `json:"description"` // localized short_description
	LowerBound  float64 `json:"lower_bound"`
	UpperBound  float64 `json:"upper_bound"`
}

type FilterPriceTier struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   *int   `json:"max"` // nil = open-ended top tier
}
