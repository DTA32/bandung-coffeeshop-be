package model

import "encoding/json"

type Location struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Thumbnail *string `json:"thumbnail"`
}

type LocationImage struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type LocationDetail struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Type            string          `json:"type"`
	Ancestors       []Location      `json:"ancestors"`
	Descendants     []Location      `json:"descendants"`
	Images          []LocationImage `json:"images"`
	ShowWelcomeText bool            `json:"show_welcome_text"`
	ShowMap         bool            `json:"show_map"`
	Polygon         json.RawMessage `json:"polygon"`
}
