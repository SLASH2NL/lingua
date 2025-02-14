package lingua

import (
	"context"
	"fmt"
	"regexp"

	"golang.org/x/text/language"
)

var (
	languageKey = ctxKey("locale")
	langRe      = regexp.MustCompile(`(?i)([a-z]{2,8})([-_][a-z]{4})?([-_][a-z]{2}|\d{3})?`)
)

// WithLanguage parses the given raw language and adds it to the ctx.
// If the language can not be parsed no language is added to the context.
// This will make Container.Message fallback to the default container language if set.
func WithLanguage(ctx context.Context, raw string) context.Context {
	lang, err := ParseLanguage(raw)
	if err != nil {
		return ctx
	}

	return toCtx(ctx, lang)
}

// fromCtx returns the language from the ctx or an empty language if no language is set.
func fromCtx(ctx context.Context) LanguageID {
	l, ok := ctx.Value(languageKey).(LanguageID)
	if ok {
		return l
	}

	// Return an empty language if no language is set.
	return LanguageID{}
}

func toCtx(ctx context.Context, id LanguageID) context.Context {
	return context.WithValue(ctx, languageKey, id)
}

// MustParseLanguage parses the language string into a LanguageID.
// If the language can not be parsed it will panic.
func MustParseLanguage(lang string) LanguageID {
	id, err := ParseLanguage(lang)
	if err != nil {
		panic(err)
	}

	return id
}

// ParseLanguage parses the language string into a LanguageID.
func ParseLanguage(lang string) (LanguageID, error) {
	match := langRe.FindString(lang)
	if match == "" {
		return LanguageID{}, fmt.Errorf("invalid language: %s", lang)
	}
	lang = match

	tag, err := language.Parse(lang)
	if err != nil {
		return LanguageID{}, fmt.Errorf("error parsing %s: %w", lang, err)
	}

	var id LanguageID

	base, baseconf := tag.Base()
	if baseconf != language.Exact {
		return LanguageID{}, fmt.Errorf("error parsing %s: could not parse base language", lang)
	}

	id.Language = base.String()

	region, regionconf := tag.Region()
	if regionconf == language.Exact {
		id.Region = region.String()
	}

	return id, nil
}

// LanguageID holds the language and an optional region.
type LanguageID struct {
	Language string
	Region   string
}

func (l LanguageID) String() string {
	if l.Region != "" {
		return l.Language + "-" + l.Region
	}

	return l.Language
}

func (l LanguageID) Empty() bool {
	return l.Language == "" && l.Region == ""
}

func (l LanguageID) Match(cmp LanguageID) (match bool, strongMatch bool) {
	if l.Language == cmp.Language && l.Region == cmp.Region {
		return true, true
	}

	if l.Language == cmp.Language {
		return true, false
	}

	return false, false
}

type ctxKey string
