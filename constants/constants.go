package constants

const (
	LocationTypeCafe     = "cafe"
	LocationTypePOI      = "poi"
	LocationTypeArea     = "area"
	LocationTypeDistrict = "district"
)

// Quicksearch `type` selectors. Also accepts specific location types above
const (
	QuicksearchTypeAll      = "all"
	QuicksearchTypeLocation = "location" // all location types
	QuicksearchTypeFilter   = "filter"
)

const (
	RatingCategoryPriceRank = "price-rank"
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
