package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ErrWeatherDisabled = errors.New("weather provider not configured")

const weatherAPIBaseURL = "https://api.weatherapi.com/v1"

// CurrentWeatherRow is the provider's raw current reading.
type CurrentWeatherRow struct {
	ConditionCode int
	TempC         float64
	UpdatedAt     time.Time
}

// WeatherRepository reads current conditions from weatherapi.com.
type WeatherRepository struct {
	apiKey string
	client *http.Client
}

func NewWeatherRepository(apiKey string) *WeatherRepository {
	return &WeatherRepository{apiKey: apiKey, client: &http.Client{Timeout: 3 * time.Second}}
}

type weatherAPIResponse struct {
	Current struct {
		LastUpdatedEpoch int64   `json:"last_updated_epoch"`
		TempC            float64 `json:"temp_c"`
		Condition        struct {
			Code int `json:"code"`
		} `json:"condition"`
	} `json:"current"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (r *WeatherRepository) Current(ctx context.Context, lat, lng float64) (*CurrentWeatherRow, error) {
	if r.apiKey == "" {
		return nil, ErrWeatherDisabled
	}
	q := url.Values{}
	q.Set("key", r.apiKey)
	q.Set("q", strconv.FormatFloat(lat, 'f', 4, 64)+","+strconv.FormatFloat(lng, 'f', 4, 64))
	q.Set("aqi", "no")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, weatherAPIBaseURL+"/current.json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	res, err := r.client.Do(req)
	if err != nil {
		// *url.Error embeds the request URL, which carries the API key; keep
		// only the underlying cause so the key never reaches the logs.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return nil, fmt.Errorf("weatherapi: request failed: %w", err)
	}
	defer res.Body.Close()

	var body weatherAPIResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("weatherapi: decode (status %d): %w", res.StatusCode, err)
	}
	if res.StatusCode != http.StatusOK || body.Error != nil {
		if body.Error != nil {
			return nil, fmt.Errorf("weatherapi: status %d: %d %s", res.StatusCode, body.Error.Code, body.Error.Message)
		}
		return nil, fmt.Errorf("weatherapi: status %d", res.StatusCode)
	}
	return &CurrentWeatherRow{
		ConditionCode: body.Current.Condition.Code,
		TempC:         body.Current.TempC,
		UpdatedAt:     time.Unix(body.Current.LastUpdatedEpoch, 0),
	}, nil
}
