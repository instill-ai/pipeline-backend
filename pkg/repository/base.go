package repository

func transformBoolToDescString(b bool) string {
	if b {
		return " DESC"
	}
	return ""
}
