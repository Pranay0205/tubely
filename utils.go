package main

import "strings"

func containsMediaType(s string, substring []string) bool {
	for _, sub := range substring {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}
