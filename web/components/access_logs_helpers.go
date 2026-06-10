package components

import "strings"

// accessLogTime は RFC3339（"2026-06-10T01:23:45Z"）を "2026-06-10 01:23:45" に整形する。
func accessLogTime(raw string) string {
	t := strings.TrimSuffix(raw, "Z")
	t = strings.Replace(t, "T", " ", 1)
	return t
}

// accessLogUser は user_id（メールアドレス）が空なら "—" を返す。
func accessLogUser(userID string) string {
	if userID == "" {
		return "—"
	}
	return userID
}
