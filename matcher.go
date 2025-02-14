package lingua

import (
	"fmt"
	"regexp"
)

var (
	// defaultMatcher matches a translation file with the format: en.yaml or en-US.yaml.
	defaultMatcher = NewRegexMatcher(regexp.MustCompile(`^([a-z]{2}(?:[-][A-Z]{2})?)\.yaml$`))
)

// FileMatcher is an interface that is used to check if a given file in a directory structure
// is a match, and should be parsed.
// It also provides a way to get the language ID from the file name.
type FileMatcher interface {
	IsMatch(name string) bool
	LanguageID(name string) (LanguageID, error)
}

// NewRegexMatcher creates a new RegexMatcher with the given regex.
func NewRegexMatcher(re *regexp.Regexp) *RegexMatcher {
	return &RegexMatcher{re: re}
}

// RegexMatcher matches all files in the directory that match the regex.
// The first capture group is used to extract the language ID.
type RegexMatcher struct {
	re *regexp.Regexp
}

func (m *RegexMatcher) IsMatch(name string) bool {
	return m.re.MatchString(name)
}

func (m *RegexMatcher) LanguageID(name string) (LanguageID, error) {
	match := m.re.FindStringSubmatch(name)
	if match == nil || len(match) < 2 {
		return LanguageID{}, fmt.Errorf("regex is missing a capture group")
	}

	return ParseLanguage(match[1])
}
