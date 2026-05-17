package model

type Location struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type QuicksearchResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
