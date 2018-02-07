package util

import (
	"fmt"
	"sort"
	"strings"
)

const (
	DefaultSuggestionsMinimumDistance = 2
)

func InvalidOptionError(option string, value string, err error) error {
	return InvalidOptionf(option, value, "%s", err)
}

func InvalidOptionf(option string, value string, message string, a ...interface{}) error {
	text := fmt.Sprintf(message, a...)
	return fmt.Errorf("Invalid option: --%s %s\n%s", option, value, text)
}

func MissingOption(name string) error {
	return fmt.Errorf("Missing option: --%s", name)
}

func InvalidOption(name string, value string, values []string) error {
	suggestions := SuggestionsFor(value, values, DefaultSuggestionsMinimumDistance)
	if len(suggestions) > 0 {
		if len(suggestions) == 1 {
			return InvalidOptionf(name, value, "Did you mean:  --%s %s", name, suggestions[0])
		}
		return InvalidOptionf(name, value, "Did you mean one of: %s", strings.Join(suggestions, ", "))
	}
	sort.Strings(values)
	return InvalidOptionf(name, value, "Possible values: %s", strings.Join(values, ", "))
}

func InvalidArg(value string, values []string) error {
	suggestions := SuggestionsFor(value, values, DefaultSuggestionsMinimumDistance)
	if len(suggestions) > 0 {
		if len(suggestions) == 1 {
			return InvalidArgf(value, "Did you mean: %s", suggestions[0])
		}
		return InvalidArgf(value, "Did you mean one of: %s", strings.Join(suggestions, ", "))
	}
	sort.Strings(values)
	return InvalidArgf(value, "Possible values: %s", strings.Join(values, ", "))
}

func InvalidArgError(value string, err error) error {
	return InvalidArgf(value, "%s", err)
}

func InvalidArgf(value string, message string, a ...interface{}) error {
	text := fmt.Sprintf(message, a...)
	return fmt.Errorf("Invalid argument: %s\n%s", value, text)
}

func SuggestionsFor(typedName string, values []string, suggestionsMinimumDistance int, explicitSuggestions ...string) []string {
	suggestions := []string{}
	for _, value := range values {
		levenshteinDistance := ld(typedName, value, true)
		suggestByLevenshtein := levenshteinDistance <= suggestionsMinimumDistance
		suggestByPrefix := strings.HasPrefix(strings.ToLower(value), strings.ToLower(typedName))
		if suggestByLevenshtein || suggestByPrefix {
			suggestions = append(suggestions, value)
		}
		for _, explicitSuggestion := range explicitSuggestions {
			if strings.EqualFold(typedName, explicitSuggestion) {
				suggestions = append(suggestions, value)
			}
		}
	}
	return suggestions
}

// ld compares two strings and returns the levenshtein distance between them.
//
// this was copied from vendor/github.com/spf13/cobra/command.go as its not public
func ld(s, t string, ignoreCase bool) int {
	if ignoreCase {
		s = strings.ToLower(s)
		t = strings.ToLower(t)
	}
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
	}
	for i := range d {
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

func Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
