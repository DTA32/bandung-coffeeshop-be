package constants

const (
	LocationTypeCafe     = "cafe"
	LocationTypePOI      = "poi"
	LocationTypeArea     = "area"
	LocationTypeDistrict = "district"
)

const (
	RatingCategoryPriceRank  = "price-rank"
	RatingCategoryVibe       = "vibe"
	RatingCategoryNoise      = "noise"
	RatingCategoryWifi       = "wifi"
	RatingCategoryMeals      = "meals"
	RatingCategoryAtmosphere = "atmosphere"
	RatingCategoryParking    = "parking"
)

const (
	SortDefault    = "default"
	SortUpdatedAt  = "updated_at"
	SortDistance   = "distance"
	SortRating     = "rating"
	SortPriceRange = "price_range"
)

const (
	OrderAsc  = "asc"
	OrderDesc = "desc"
)

const (
	LangIndonesian = "id"
	LangEnglish    = "en"
	// DefaultLang is used when the client sends no (recognised) Accept-Language.
	DefaultLang = LangIndonesian
)
