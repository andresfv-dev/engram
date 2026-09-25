package dashboard

import "strings"

// noticeMessages is the fixed, server-controlled allowlist of admin-action
// success notices (admin-ui-redesign T7). Handlers redirect (or render) with
// a "notice=<code>" query value; Layout looks the code up here and renders
// the mapped message as a transient toast. A code that is not in this map
// renders nothing — raw query text is never echoed into the page.
var noticeMessages = map[string]string{
	"user_created":    "User created",
	"user_enabled":    "User enabled",
	"user_disabled":   "User disabled",
	"token_created":   "Token created",
	"token_revoked":   "Token revoked",
	"project_granted": "Project access granted",
	"project_revoked": "Project access revoked",
}

// NoticeMessage returns the toast message for code, or "" if code is not on
// the allowlist above.
func NoticeMessage(code string) string {
	return noticeMessages[strings.TrimSpace(code)]
}
