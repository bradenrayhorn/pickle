package bucket

import "regexp"

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	unsafeRegex     = regexp.MustCompile(`[^0-9a-zA-Z!\-_.*'()/]`)
)

func cleanKeyName(input string) string {
	result := whitespaceRegex.ReplaceAllString(input, "_")
	result = unsafeRegex.ReplaceAllString(result, "")

	return result
}
