package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/dta32/bandung-coffeeshop-be/cache"
	"github.com/dta32/bandung-coffeeshop-be/constants"
	"github.com/dta32/bandung-coffeeshop-be/model"
	"github.com/dta32/bandung-coffeeshop-be/repository"
	"golang.org/x/sync/singleflight"
)

// Bandung city centre (Alun-alun); one reading serves the whole city.
const (
	bandungLat = -6.9175
	bandungLng = 107.6191
)

const (
	weatherFreshKey = "bdgcafe:weather:bandung"
	weatherLastKey  = "bdgcafe:weather:bandung:last"
	weatherFreshTTL = 30 * time.Minute
	// weatherLastTTL bounds how stale a fallback reading may get while the
	// provider is failing.
	weatherLastTTL = 24 * time.Hour
)

var wib = time.FixedZone("WIB", 7*3600)

type WeatherService struct {
	repo  *repository.WeatherRepository
	cache *cache.Redis
	group singleflight.Group

	mu       sync.Mutex
	last     *model.Weather // in-process copy: Redis-less mode and stale fallback
	lastSeen time.Time
}

func NewWeatherService(repo *repository.WeatherRepository, redisCache *cache.Redis) *WeatherService {
	return &WeatherService{repo: repo, cache: redisCache}
}

// Current returns Bandung's current weather, refreshed at most every 30
// minutes. Lookup order: Redis fresh key → in-process copy younger than 30
// min → provider → Redis last-known key → in-process last-known copy. Returns an error only when none of them has a
// reading.
func (s *WeatherService) Current(ctx context.Context) (*model.Weather, error) {
	if w := s.fromCache(ctx, weatherFreshKey); w != nil {
		return w, nil
	}
	// Redis missed (expired, disabled, or down): a reading this instance
	// fetched within the window is just as fresh, and keeps a Redis outage
	// from turning every request into a provider call.
	if w := s.memo(weatherFreshTTL); w != nil {
		return w, nil
	}

	// Collapse concurrent misses into one provider call per instance. The
	// fetch is detached from the caller's context so one cancelled request
	// can't fail the others waiting on it.
	v, err, _ := s.group.Do("current", func() (any, error) {
		return s.fetch(context.WithoutCancel(ctx))
	})
	if err == nil {
		return v.(*model.Weather), nil
	}
	if errors.Is(err, repository.ErrWeatherDisabled) {
		return nil, err
	}

	log.Printf("weather: provider lookup failed, using last known reading: %v", err)
	if w := s.fromCache(ctx, weatherLastKey); w != nil {
		return w, nil
	}
	if w := s.memo(weatherLastTTL); w != nil {
		return w, nil
	}
	return nil, err
}

func (s *WeatherService) fetch(ctx context.Context) (*model.Weather, error) {
	row, err := s.repo.Current(ctx, bandungLat, bandungLng)
	if err != nil {
		return nil, err
	}
	w := &model.Weather{
		Condition:  weatherCondition(row.ConditionCode),
		TempC:      row.TempC,
		ObservedAt: row.UpdatedAt.In(wib),
	}

	s.mu.Lock()
	s.last, s.lastSeen = w, time.Now()
	s.mu.Unlock()

	if s.cache != nil {
		if b, err := json.Marshal(w); err == nil {
			if err := s.cache.Set(ctx, weatherFreshKey, b, weatherFreshTTL); err != nil {
				log.Printf("weather: cache set failed: %v", err)
			}
			_ = s.cache.Set(ctx, weatherLastKey, b, weatherLastTTL)
		}
	}
	return w, nil
}

// fromCache reads a reading from Redis; any error or miss yields nil.
func (s *WeatherService) fromCache(ctx context.Context, key string) *model.Weather {
	if s.cache == nil {
		return nil
	}
	b, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil
	}
	var w model.Weather
	if err := json.Unmarshal(b, &w); err != nil {
		return nil
	}
	w.ObservedAt = w.ObservedAt.In(wib)
	return &w
}

// memo returns the in-process reading if it was fetched within maxAge.
func (s *WeatherService) memo(maxAge time.Duration) *model.Weather {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil || time.Since(s.lastSeen) > maxAge {
		return nil
	}
	return s.last
}

// weatherCondition folds a weatherapi.com condition code
// (https://www.weatherapi.com/docs/weather_conditions.json) into the three
// values cafes are tagged with. Anything wet — drizzle, rain, showers,
// thunder, sleet, snow — counts as rain; mist and fog count as cloudy.
func weatherCondition(code int) string {
	switch code {
	case 1000: // Sunny / Clear
		return constants.WeatherClear
	case 1003, 1006, 1009, // Partly cloudy, Cloudy, Overcast
		1030, 1135, 1147: // Mist, Fog, Freezing fog
		return constants.WeatherCloudy
	}
	if code >= 1063 && code <= 1282 {
		return constants.WeatherRain
	}
	log.Printf("weather: unknown condition code %d, treating as cloudy", code)
	return constants.WeatherCloudy
}
