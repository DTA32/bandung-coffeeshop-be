package service

import (
	"testing"

	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCoords(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    *model.Coordinates
		wantErr error
	}{
		{"empty returns nil nil", "", nil, nil},
		{"whitespace only returns nil nil", "   ", nil, nil},
		{"three parts", "1,2,3", nil, ErrInvalidCoords},
		{"one part", "1", nil, ErrInvalidCoords},
		{"non-numeric lat", "abc,2", nil, ErrInvalidCoords},
		{"non-numeric lng", "1,xyz", nil, ErrInvalidCoords},
		{"empty lng part", "1,", nil, ErrInvalidCoords},
		{"lat above range", "91,0", nil, ErrInvalidCoords},
		{"lat below range", "-91,0", nil, ErrInvalidCoords},
		{"lng above range", "0,181", nil, ErrInvalidCoords},
		{"lng below range", "0,-181", nil, ErrInvalidCoords},
		{"whitespace trimmed around parts", " 1.5 , 2.5 ", &model.Coordinates{Lat: 1.5, Lng: 2.5}, nil},
		{"lower bound inclusive", "-90,-180", &model.Coordinates{Lat: -90, Lng: -180}, nil},
		{"upper bound inclusive", "90,180", &model.Coordinates{Lat: 90, Lng: 180}, nil},
		{"bandung coords", "-6.9147,107.6098", &model.Coordinates{Lat: -6.9147, Lng: 107.6098}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoords(tt.in)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatThousand(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{25000, "25k"},
		{1000, "1k"},
		{0, "0k"},
		{25500, "25500"},
		{999, "999"},
		{-1000, "-1k"},
		{1500, "1500"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, formatThousand(tt.in))
		})
	}
}

func TestFormatPriceRange(t *testing.T) {
	tests := []struct {
		name string
		lang string
		min  *int
		max  *int
		want *string
	}{
		{"both nil returns nil", "en", nil, nil, nil},
		{"both set ignores locale (en)", "en", intPtr(25000), intPtr(50000), strPtr("Rp. 25k - Rp. 50k")},
		{"both set ignores locale (id)", "id", intPtr(25000), intPtr(50000), strPtr("Rp. 25k - Rp. 50k")},
		{"both set non-round", "en", intPtr(25500), intPtr(50000), strPtr("Rp. 25500 - Rp. 50k")},
		{"min only english", "en", intPtr(25000), nil, strPtr("start from Rp. 25k")},
		{"min only indonesian", "id", intPtr(25000), nil, strPtr("mulai dari Rp. 25k")},
		{"min only unknown lang defaults to id", "fr", intPtr(25000), nil, strPtr("mulai dari Rp. 25k")},
		{"max only english", "en", nil, intPtr(50000), strPtr("up to Rp. 50k")},
		{"max only indonesian", "id", nil, intPtr(50000), strPtr("hingga Rp. 50k")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPriceRange(tt.lang, tt.min, tt.max)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *tt.want, *got)
		})
	}
}

func TestNormLang(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"en", constants.LangEnglish},
		{"id", constants.LangIndonesian},
		{"", constants.LangIndonesian},
		{"EN", constants.LangIndonesian},
		{"fr", constants.LangIndonesian},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, normLang(tt.in))
		})
	}
}

func TestFormatLocationLabel(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		focus     *repository.FocusLocation
		coords    *model.Coordinates
		wantName  string
		wantLabel string
	}{
		{"area english", "en", &repository.FocusLocation{Name: "Dago", Type: constants.LocationTypeArea}, nil, "Dago", "in Dago"},
		{"area indonesian", "id", &repository.FocusLocation{Name: "Dago", Type: constants.LocationTypeArea}, nil, "Dago", "di Dago"},
		{"district english", "en", &repository.FocusLocation{Name: "Coblong", Type: constants.LocationTypeDistrict}, nil, "Coblong", "in Coblong"},
		{"cafe english", "en", &repository.FocusLocation{Name: "Kopi A", Type: constants.LocationTypeCafe}, nil, "Kopi A", "near Kopi A"},
		{"poi indonesian", "id", &repository.FocusLocation{Name: "ITB", Type: constants.LocationTypePOI}, nil, "ITB", "dekat ITB"},
		{"coords only english", "en", nil, &model.Coordinates{Lat: 1, Lng: 2}, "", "near Selected Spot"},
		{"coords only indonesian", "id", nil, &model.Coordinates{Lat: 1, Lng: 2}, "", "dekat Lokasi Terpilih"},
		{"nothing", "en", nil, nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, label := formatLocationLabel(tt.lang, tt.focus, tt.coords)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantLabel, label)
		})
	}
}

func TestBuildBreadcrumb(t *testing.T) {
	t.Run("nil focus returns empty non-nil", func(t *testing.T) {
		got := buildBreadcrumb(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("district focus is itself only", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict})
		assert.Equal(t, []model.Location{
			{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict},
		}, got)
	})

	t.Run("area with district ancestor", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{
			ID: "a1", Name: "Dago", Type: constants.LocationTypeArea,
			DistrictID: strPtr("d1"), DistrictName: strPtr("Coblong"),
		})
		assert.Equal(t, []model.Location{
			{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict},
			{ID: "a1", Name: "Dago", Type: constants.LocationTypeArea},
		}, got)
	})

	t.Run("area without district ancestor", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{ID: "a1", Name: "Dago", Type: constants.LocationTypeArea})
		assert.Equal(t, []model.Location{
			{ID: "a1", Name: "Dago", Type: constants.LocationTypeArea},
		}, got)
	})

	t.Run("poi with district and area ancestors", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{
			ID: "p1", Name: "ITB", Type: constants.LocationTypePOI,
			DistrictID: strPtr("d1"), DistrictName: strPtr("Coblong"),
			AreaID: strPtr("a1"), AreaName: strPtr("Dago"),
		})
		assert.Equal(t, []model.Location{
			{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict},
			{ID: "a1", Name: "Dago", Type: constants.LocationTypeArea},
			{ID: "p1", Name: "ITB", Type: constants.LocationTypePOI},
		}, got)
	})

	t.Run("poi with only district ancestor", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{
			ID: "p1", Name: "ITB", Type: constants.LocationTypePOI,
			DistrictID: strPtr("d1"), DistrictName: strPtr("Coblong"),
		})
		assert.Equal(t, []model.Location{
			{ID: "d1", Name: "Coblong", Type: constants.LocationTypeDistrict},
			{ID: "p1", Name: "ITB", Type: constants.LocationTypePOI},
		}, got)
	})

	t.Run("cafe focus returns empty", func(t *testing.T) {
		got := buildBreadcrumb(&repository.FocusLocation{ID: "c1", Name: "Kopi A", Type: constants.LocationTypeCafe})
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestBuildSearchDescription(t *testing.T) {
	svc := &CafeService{}
	tag := &repository.Tag{Slug: "cozy", Description: "Cozy cafes to relax"}

	t.Run("tag only returns tag description", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		assert.Equal(t, "Cozy cafes to relax", svc.buildSearchDescription(req, nil, tag))
	})

	t.Run("tag present but focus disqualifies tag-only", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		focus := &repository.FocusLocation{Type: constants.LocationTypeArea, Description: "Area desc"}
		assert.Equal(t, "Area desc", svc.buildSearchDescription(req, focus, tag))
	})

	t.Run("tag present but is_featured disqualifies tag-only", func(t *testing.T) {
		req := &model.CafeSearchRequest{IsFeatured: boolPtr(true)}
		assert.Equal(t, "", svc.buildSearchDescription(req, nil, tag))
	})

	t.Run("tag present but price disqualifies tag-only", func(t *testing.T) {
		req := &model.CafeSearchRequest{PriceMin: intPtr(10000)}
		assert.Equal(t, "", svc.buildSearchDescription(req, nil, tag))
	})

	t.Run("focus area returns its description", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		focus := &repository.FocusLocation{Type: constants.LocationTypeDistrict, Description: "District desc"}
		assert.Equal(t, "District desc", svc.buildSearchDescription(req, focus, nil))
	})

	t.Run("focus poi returns its description", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		focus := &repository.FocusLocation{Type: constants.LocationTypePOI, Description: "POI desc"}
		assert.Equal(t, "POI desc", svc.buildSearchDescription(req, focus, nil))
	})

	t.Run("focus cafe returns empty", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		focus := &repository.FocusLocation{Type: constants.LocationTypeCafe, Description: "Cafe desc"}
		assert.Equal(t, "", svc.buildSearchDescription(req, focus, nil))
	})

	t.Run("no tag no focus returns empty", func(t *testing.T) {
		req := &model.CafeSearchRequest{}
		assert.Equal(t, "", svc.buildSearchDescription(req, nil, nil))
	})
}
