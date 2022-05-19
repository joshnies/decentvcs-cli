package util

import (
	"fmt"

	"golang.org/x/exp/maps"
)

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

// Returns a chunked slice.
func ChunkSlice(items []string, chunkSize int) (chunks [][]string) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}

// Returns a chunked map.
func ChunkMap(sourceMap map[string]string, chunkSize int) []map[string]string {
	keyChunks := ChunkSlice(maps.Keys(sourceMap), chunkSize)
	mapChunks := []map[string]string{}

	// Iterate over chunk of keys
	for _, chunk := range keyChunks {
		newMap := make(map[string]string)

		// For each key in chunk, add to new map
		for _, key := range chunk {
			newMap[key] = sourceMap[key]
		}

		// Add new map to map chunks (the result)
		mapChunks = append(mapChunks, newMap)
	}

	return mapChunks
}
