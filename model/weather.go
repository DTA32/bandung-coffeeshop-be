package model

import "time"

// Weather is the resolved current weather for Bandung.
type Weather struct {
	Condition  string    `json:"condition"` // constants.Weather{Clear,Cloudy,Rain}
	TempC      float64   `json:"temp_c"`
	ObservedAt time.Time `json:"observed_at"` // provider's last update, in WIB
}
