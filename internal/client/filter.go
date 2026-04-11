package client

import (
	"path/filepath"
	"regexp"
	"strings"
)

// filterFiles filters a list of file paths based on the filter expression
func (c *Client) filterFiles(files []string, filterExp string, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath bool) []string {
	var filtered []string

	// Compile regex if needed
	var regex *regexp.Regexp
	var err error
	if filterRegex {
		if filterIgnoreCase {
			regex, err = regexp.Compile("(?i)" + filterExp)
		} else {
			regex, err = regexp.Compile(filterExp)
		}
		if err != nil {
			c.logger.Error("Invalid regex filter", "error", err)
			return files // Return original if regex is invalid
		}
	}

	for _, file := range files {
		textToCheck := file
		if !filterMatchPath {
			textToCheck = filepath.Base(file)
		}

		var matches bool
		if filterRegex {
			matches = regex.MatchString(textToCheck)
		} else {
			if filterIgnoreCase {
				matches = strings.Contains(strings.ToLower(textToCheck), strings.ToLower(filterExp))
			} else {
				matches = strings.Contains(textToCheck, filterExp)
			}
		}

		if (matches && !filterExclude) || (!matches && filterExclude) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}
