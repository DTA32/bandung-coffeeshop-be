package helper

import (
	"strings"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/gin-gonic/gin"
)

// Lang resolves the request locale from the Accept-Language header, defaulting
// to constants.DefaultLang when absent or unrecognised.
func Lang(c *gin.Context) string {
	return ParseAcceptLanguage(c.GetHeader("Accept-Language"))
}

// ParseAcceptLanguage returns the first supported locale code found in an
// Accept-Language header value, tolerating quality-value lists such as
// "en-US,en;q=0.9,id;q=0.8". Browsers order tags by descending preference, so
// the first recognised primary subtag wins. Falls back to constants.DefaultLang.
func ParseAcceptLanguage(header string) string {
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(part)
		if i := strings.IndexByte(tag, ';'); i >= 0 {
			tag = tag[:i] // drop the ";q=..." weight
		}
		tag = strings.ToLower(strings.TrimSpace(tag))
		if i := strings.IndexByte(tag, '-'); i >= 0 {
			tag = tag[:i] // primary subtag only ("en-US" -> "en")
		}
		switch tag {
		case constants.LangEnglish:
			return constants.LangEnglish
		case constants.LangIndonesian:
			return constants.LangIndonesian
		}
	}
	return constants.DefaultLang
}
