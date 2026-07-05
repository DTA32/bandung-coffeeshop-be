package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"empty falls back to default", "", constants.DefaultLang},
		{"default is indonesian", "", constants.LangIndonesian},
		{"plain english", "en", constants.LangEnglish},
		{"plain indonesian", "id", constants.LangIndonesian},
		{"region stripped to primary subtag", "en-US", constants.LangEnglish},
		{"weighted list picks first recognised", "en-US,en;q=0.9,id;q=0.8", constants.LangEnglish},
		{"first recognised wins when id leads", "id,en;q=0.9", constants.LangIndonesian},
		{"q-weight stripped", "en;q=0.5", constants.LangEnglish},
		{"surrounding spaces tolerated", "  en  ", constants.LangEnglish},
		{"mixed case lowercased", "EN-GB", constants.LangEnglish},
		{"unknown leading tag skipped to next", "fr-FR,id;q=0.7", constants.LangIndonesian},
		{"all unknown falls back to default", "fr,de,zh", constants.DefaultLang},
		{"uppercase indonesian", "ID", constants.LangIndonesian},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseAcceptLanguage(tt.header))
		})
	}
}

func TestLang(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		header string // empty => header not set at all
		want   string
	}{
		{"no header defaults to indonesian", "", constants.DefaultLang},
		{"english header", "en-US,en;q=0.9", constants.LangEnglish},
		{"indonesian header", "id-ID", constants.LangIndonesian},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				c.Request.Header.Set("Accept-Language", tt.header)
			}
			assert.Equal(t, tt.want, Lang(c))
		})
	}
}
