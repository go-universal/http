package content

import "strings"

// isValidContent checks if the given string `c` starts with any of the valid prefixes provided in `valids`.
// The comparison is case-insensitive and trims any leading or trailing spaces from `c`.
func isValidContent(c string, valids ...string) bool {
	// Normalize the input string by trimming spaces and converting to lowercase.
	c = strings.ToLower(strings.TrimSpace(c))

	// Iterate over the valid prefixes and check if `c` starts with any of them.
	for _, v := range valids {
		if strings.HasPrefix(c, strings.ToLower(v)) {
			return true
		}
	}

	// Return false if no valid prefix matches.
	return false
}
