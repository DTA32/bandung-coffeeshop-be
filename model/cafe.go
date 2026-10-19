package model

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type CafeSearchRequest struct {
	QueryID     string
	QueryType   string
	QueryCoords *Coordinates
	RadiusMax   *int
	RatingIDs   []int // holds the selected rating_category bucket ids (one per category).
	Tags        []string
	OpenHour    string // literal "now" or an "HH:MM" time string; empty = no filter.
	PriceMin    *int
	PriceMax    *int
	IsFeatured  *bool
	Status      string // location status: "active" (default) or "closed".
	Weather     []string
	Lang        string
	Sort        string
	Order       string
	Page        int
	Size        int
}

type CafeDetail struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Coordinates *Coordinates `json:"coordinates"`
	Thumbnail   *string      `json:"thumbnail"`
	Area        *string      `json:"area"`
	PriceRange  *string      `json:"price_range"`
	Distance    *int         `json:"distance"`
	Remark      *string      `json:"remark"`
}

type CafeSearchResponse struct {
	Total                 int          `json:"total"`
	LocationName          string       `json:"location_name"`
	FormattedLocationName string       `json:"formatted_location_name"`
	SearchDescription     string       `json:"search_description"`
	Locations             []Location   `json:"locations"`
	Cafes                 []CafeDetail `json:"cafes"`
	Page                  int          `json:"page"`
	Size                  int          `json:"size"`
	Weather               *Weather     `json:"weather"`
}

type CafeDetailResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	About       *string         `json:"about"`
	Status      string          `json:"status"`
	Images      []LocationImage `json:"images"`
	Instagram   *string         `json:"instagram"`
	Menu        *string         `json:"menu"`
	OpenHour    *string         `json:"open_hour"`
	CloseHour   *string         `json:"close_hour"`
	GmapsID     *string         `json:"gmaps_id"`
	Locations   []Location      `json:"locations"`
	Price       CafePrice       `json:"price"`
	UpdatedAt   string          `json:"updated_at"`
}

type CafePrice struct {
	PriceRangeMin  *int      `json:"price_range_min"`
	PriceRangeMax  *int      `json:"price_range_max"`
	CoffeePriceMin *int      `json:"coffee_price_min"`
	CoffeePriceMax *int      `json:"coffee_price_max"`
	SnackPriceMin  *int      `json:"snack_price_min"`
	SnackPriceMax  *int      `json:"snack_price_max"`
	FoodPriceMin   *int      `json:"food_price_min"`
	FoodPriceMax   *int      `json:"food_price_max"`
	Rank           *CafeRank `json:"rank"`
}

type CafeRank struct {
	Type  int    `json:"type"`
	Label string `json:"label"`
}

// CafeReviewResponse aggregates a cafe's visible reviews.
type CafeReviewResponse struct {
	OverallScore       *float64        `json:"overall_score"`
	ReviewCount        int             `json:"review_count"`
	OverallReviewCount int             `json:"overall_review_count"`
	WFCScore           *float64        `json:"wfc_score"`
	WFCReviewCount     int             `json:"wfc_review_count"`
	Tags               []CafeTag       `json:"tags"`
	Ratings            RatingsResponse `json:"ratings"`
}

type CafeFeedItem struct {
	ID          int      `json:"id"`
	Content     string   `json:"content"`
	Images      []string `json:"images"`
	Author      string   `json:"author"`
	Link        *string  `json:"link"`
	PublishedAt string   `json:"published_at"`
}

type CafeTag struct {
	Name string  `json:"name"`
	Slug *string `json:"slug"`
}

type RatingEntry struct {
	DisplayName string        `json:"display_name"`
	Range       []RatingRange `json:"range"`
	Score       float64       `json:"score"`
	Description string        `json:"description"`
}

type RatingRange struct {
	Name        string  `json:"name"`
	Slug        *string `json:"slug"`
	Description string  `json:"description"`
	LowerBound  float64 `json:"lower_bound"`
	UpperBound  float64 `json:"upper_bound"`
}

type RatingsResponse map[string]RatingEntry
