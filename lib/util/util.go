package util

// Return key for given value in a string map.
// Returns empty string if value is not found.
func ReverseLookup(strMap map[string]string, value string) string {
	for k, v := range strMap {
		if v == value {
			return k
		}
	}

	return ""
}
