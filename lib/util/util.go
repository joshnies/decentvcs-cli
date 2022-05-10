package util

import "fmt"

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

// Format size in bytes to human readable format.
func FormatBytesSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	}

	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/1024/1024)
	}

	return fmt.Sprintf("%.2f GB", float64(size)/1024/1024/1024)
}
