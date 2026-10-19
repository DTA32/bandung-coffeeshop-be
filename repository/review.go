package repository

// A review is publicly visible when it is approved and was written either as
// a partnership review (no user) or by a still-active user. Every query that
// serves or aggregates public reviews must use these two fragments together,
// with cafe_review aliased as rv:
//
//	FROM cafe_review rv ` + visibleReviewJoin + `
//	WHERE ... AND ` + visibleReviewCond
const (
	visibleReviewJoin = `LEFT JOIN users u ON u.id = rv.user_id`
	visibleReviewCond = `rv.status = 'approved' AND (rv.user_id IS NULL OR u.status = 'active')`
)
