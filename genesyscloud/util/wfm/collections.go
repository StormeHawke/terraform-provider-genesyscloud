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

// StringValue returns the string a pointer refers to, or an empty string for nil
func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
