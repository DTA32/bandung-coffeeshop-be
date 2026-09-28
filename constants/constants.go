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

// Location status (location_status_enum). `deleted` is never exposed.
const (
	LocationStatusActive = "active"
	LocationStatusClosed = "closed"
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

// Weather conditions a cafe can be tagged with (cafe.weather) and filtered by.
// WeatherCurrent is a search-only selector resolved to today's condition.
const (
	WeatherClear   = "clear"
	WeatherCloudy  = "cloudy"
	WeatherRain    = "rain"
	WeatherCurrent = "current"
)

// WeatherValues is the canonical, display-ordered weather set; it must match
// the cafe_weather_valid CHECK in migrations/005_cafe_weather.sql.
var WeatherValues = []string{WeatherClear, WeatherCloudy, WeatherRain}
