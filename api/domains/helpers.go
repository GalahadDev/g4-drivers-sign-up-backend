package domains

// GetString safely dereferences a *string. Returns "" if nil.
func GetString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
