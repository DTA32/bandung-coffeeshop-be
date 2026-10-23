package constants

// Auth (BCF-25): login providers, user statuses, cookie names and the
// auth_error codes the callback redirects back with.
const (
	AuthProviderGoogle = "google"

	UserStatusActive  = "active"
	UserStatusDeleted = "deleted"

	SessionCookie = "bdg_session"
	OAuthCookie   = "bdg_oauth"

	AuthErrorDeactivated = "deactivated"
	AuthErrorFailed      = "failed"
)
