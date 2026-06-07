package model

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type CafeSearchRequest struct {
	QueryID            string
	QueryType          string
	QueryCoords        *Coordinates
	RadiusMax          *int
	RatingCategoryType string
	RatingCategorySlug string
	Tag                string
	IsFeatured         *bool
	Sort               string
	Order              string
	Page               int
	Size               int
}

type CafeDetail struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
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
}

type CafeDetailResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Status      string          `json:"status"`
	Images      []LocationImage `json:"images"`
	Instagram   *string         `json:"instagram"`
	OpenHour    *string         `json:"open_hour"`
	CloseHour   *string         `json:"close_hour"`
	GmapsID     *string         `json:"gmaps_id"`
	Locations   []Location      `json:"locations"`
	Price       CafePrice       `json:"price"`
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

type CafeReviewResponse struct {
	IsSubjective bool            `json:"is_subjective"`
	OverallScore *float64        `json:"overall_score"`
	WFCScore     *float64        `json:"wfc_score"`
	Tags         []CafeTag       `json:"tags"`
	Content      *string         `json:"content"`
	VisitedAt    *string         `json:"visited_at"`
	UpdatedAt    string          `json:"updated_at"`
	Ratings      RatingsResponse `json:"ratings"`
}

type CafeTag struct {
	Name string  `json:"name"`
	Slug *string `json:"slug"`
}

type RatingEntry struct {
	Range       []RatingRange `json:"range"`
	Score       float64       `json:"score"`
	Description string        `json:"description"`
}

type RatingRange struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	LowerBound  float64 `json:"lower_bound"`
	UpperBound  float64 `json:"upper_bound"`
}

type RatingsResponse map[string]RatingEntry
