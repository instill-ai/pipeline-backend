package github

// PageOptions represents the pagination options for a request
type PageOptions struct {
	Page    int `instill:"page"`
	PerPage int `instill:"per-page"`
}

// getString safely dereferences a string pointer and returns an empty string if nil
func getString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// getInt safely dereferences an int pointer and returns 0 if nil
func getInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

// getInt64 safely dereferences an int64 pointer and returns 0 if nil
func getInt64(i *int64) int64 {
	if i != nil {
		return *i
	}
	return 0
}

// getBool safely dereferences a bool pointer and returns false if nil
func getBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}
