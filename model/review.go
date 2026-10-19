package model

// ReviewItem is one review as shown on the cafe page.
type ReviewItem struct {
	ID               int                `json:"id"`
	ReviewerName     string             `json:"reviewer_name"`
	OverallScore     *float64           `json:"overall_score"`
	WFCScore         *float64           `json:"wfc_score"`
	Content          *string            `json:"content"`
	VisitedAt        *string            `json:"visited_at"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	ExternalURLTitle *string            `json:"external_url_title"`
	ExternalURLLink  *string            `json:"external_url_link"`
	Ratings          map[string]float64 `json:"ratings"`
}

type ReviewListResponse struct {
	Items      []ReviewItem `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}
