package wfm

import "sort"

// SortedKeys returns the keys of a set in alphabetical order
func SortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
