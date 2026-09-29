package model

import "time"

type Weather struct {
	Condition  string    `json:"condition"`
	TempC      float64   `json:"temp_c"`
	ObservedAt time.Time `json:"observed_at"`
}

type RandomCafe struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
