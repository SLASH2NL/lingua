package lingua

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/SLASH2NL/lingua/internal/parser"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// Key is a unique identifier for a translation message.
type Key string

// ContainerFromFs reads all translation files that match the default FileMatcher from the fs.
func ContainerFromFs(fs afero.Fs, opts ...ContainerOpt) (*Container, error) {
	return ContainerFromFsAndMatcher(fs, defaultMatcher, opts...)
}

func ContainerFromFsAndMatcher(fs afero.Fs, matcher FileMatcher, opts ...ContainerOpt) (*Container, error) {
	c := &Container{
		messages: make(map[LanguageID]map[Key]*parser.Message),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Read all files from the directory.
	entries, err := afero.ReadDir(fs, ".")
	if err != nil {
		return nil, fmt.Errorf("unable to read fs: %w", err)
	}

	files := make(map[LanguageID]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !matcher.IsMatch(entry.Name()) {
			continue
		}

		langID, err := matcher.LanguageID(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("unable to parse language %q: %w", entry.Name(), err)
		}

		if _, ok := files[langID]; ok {
			return nil, fmt.Errorf("duplicate language file %q for language %s", entry.Name(), langID.String())
		}

		files[langID] = entry.Name()
	}

	for langID, file := range files {
		f, err := fs.Open(file)
		if err != nil {
			return nil, fmt.Errorf("unable to open file %q: %w", file, err)
		}
		defer f.Close()

		err = c.addFile(langID, f)
		if err != nil {
			return nil, fmt.Errorf("unable to add file %q: %w", file, err)
		}
	}

	return c, nil
}

type Container struct {
	messages map[LanguageID]map[Key]*parser.Message

	defaultLanguage LanguageID
}

func (c *Container) Message(ctx context.Context, key Key, replacements map[string]any) string {
	lang := c.ScopedLanguage(ctx)
	if lang.Empty() {
		return string(key)
	}

	scope := c.messages[lang]

	msg, ok := scope[key]
	if !ok {
		return string(key)
	}

	formattedReplacements := make(map[string]string)
	for key, value := range replacements {
		formattedReplacements[key] = formatReplacement(value)
	}

	return c.format(msg, formattedReplacements, scope)
}

func (c *Container) format(msg *parser.Message, replacements map[string]string, messages map[Key]*parser.Message) string {
	var b strings.Builder

	// Simple pre-allocate the buffer.
	// This does not take into account transformers.
	length := 0
	for _, t := range msg.Ops {
		switch v := t.(type) {
		case parser.ReplacementOp:
			// Check if a replacement is provided.
			if rep, ok := replacements[v.Key]; ok {
				length += len(rep)
			} else {
				// If no replacement provided, leave the placeholder as-is.
				length += len(v.Key) + 1
			}
		case parser.LiteralOp:
			length += len(v.Value)
		}
	}

	b.Grow(length)

	var replacementB strings.Builder
	for _, t := range msg.Ops {
		switch v := t.(type) {
		case parser.LiteralOp:
			b.WriteString(v.Value)
		case parser.ReplacementOp:
			value, ok := replacements[v.Key]
			if !ok {
				// If no replacement provided, leave the placeholder as-is.
				b.WriteString(":" + v.Key)
				continue
			}

			for _, transformer := range v.Transformers {
				switch t := transformer.(type) {
				case parser.CapitalizeTransformer:
					r, size := utf8.DecodeRuneInString(value)

					value = string(unicode.ToUpper(r)) + value[size:]
				case parser.ReplaceTransformer:
					if rep, ok := messages[Key(value)]; ok {
						// Only allow literals as replacements.
						value = rep.Raw()
					}
				case parser.PluralTransformer:
					// Convert value to int. If that fails we assume 0.
					count, err := strconv.Atoi(value)
					if err != nil {
						count = 0
					}

					var ops []any
					for _, c := range t.Cases {
						if c.Match(count) {
							ops = c.Ops
							break
						}
					}

					if len(ops) == 0 {
						ops = t.Other
					}

					replacementB.Reset()
					for _, c := range ops {
						switch c := c.(type) {
						case parser.LiteralOp:
							replacementB.WriteString(c.Value)
						case parser.PluralCountOp:
							replacementB.WriteString(strconv.Itoa(count))
						}
					}
					value = replacementB.String()
				}
			}

			b.WriteString(value)
		}
	}
	return b.String()
}

// Scope returns a container type with the ctx embedded.
func (c *Container) Scope(ctx context.Context) *ScopedContainer {
	return &ScopedContainer{
		ctx: ctx,
		c:   c,
	}
}

// ScopedLanguage returns the language in the context or falls back to no strict matches or the default lang.
// Returns and empty LanguageID{} if no language can be detected.
func (c *Container) ScopedLanguage(ctx context.Context) LanguageID {
	// Get the language from the context.
	// Fallback to the defaultLanguage. If no language can be detected return the translation key.
	lang := FromCtx(ctx)
	if lang.Empty() {
		if c.defaultLanguage.Empty() {
			return LanguageID{}
		}

		lang = c.defaultLanguage
	}

	var firstMatch LanguageID
	for scoped := range c.messages {
		isMatch, isExactMatch := scoped.Match(lang)
		if isMatch && isExactMatch {
			return scoped
		} else if isMatch {
			firstMatch = scoped
		}
	}

	if !firstMatch.Empty() {
		return firstMatch
	}

	if !c.defaultLanguage.Empty() {
		return c.defaultLanguage
	}

	return LanguageID{}
}

// Raw returns the raw messages from the container.
func (c *Container) Raw() map[LanguageID]map[string]string {
	raw := make(map[LanguageID]map[string]string, len(c.messages))
	for lang, messages := range c.messages {
		raw[lang] = make(map[string]string)

		for key, message := range messages {
			raw[lang][string(key)] = message.Raw()
		}
	}

	return raw
}

func (c *Container) Messages(lang LanguageID) map[Key]*parser.Message {
	return c.messages[lang]
}

// Merge merges the messages from the from container to the to container based on the strategy.
func Merge(from *Container, to *Container, strategy MergeStrategy) *Container {
	// Add messages to the to container that are in from.
	for language, messages := range from.messages {
		for key, msg := range messages {
			switch strategy {
			case Skip, SkipAndClean:
				// If the message exists in the to container we skip it.
				if _, ok := to.messages[language][key]; ok {
					continue
				}
			}

			if _, ok := to.messages[language]; !ok {
				to.messages[language] = make(map[Key]*parser.Message)
			}

			to.messages[language][key] = msg
		}
	}

	// If the strategy is to clean and skip we remove all messages from to that are not in from.
	switch strategy {
	case OverWriteAndClean, SkipAndClean:
		for language, messages := range to.messages {
			for key := range messages {
				// If the whole language does not exist.
				if _, ok := from.messages[language]; !ok {
					delete(to.messages[language], key)
					continue
				}

				// Or the message does not exist.
				if _, ok := from.messages[language][key]; !ok {
					delete(to.messages[language], key)
					continue
				}
			}
		}
	}

	return to
}

func (c *Container) addFile(language LanguageID, content io.Reader) error {
	var rawMessages map[string]string

	err := yaml.NewDecoder(content).Decode(&rawMessages)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("unable to decode yaml: %w", err)
	}

	c.messages[language] = make(map[Key]*parser.Message)

	for key, raw := range rawMessages {
		c.messages[language][Key(key)], err = parser.Parse(raw)
		if err != nil {
			return fmt.Errorf("unable to parse message %q: %w", key, err)
		}
	}

	return nil
}

type ScopedContainer struct {
	ctx context.Context
	c   *Container
}

func (s *ScopedContainer) Message(key Key, replacements map[string]any) string {
	return s.c.Message(s.ctx, key, replacements)
}

func WithDefaultLanguage(lang LanguageID) ContainerOpt {
	return func(c *Container) {
		c.defaultLanguage = lang
	}
}

func formatReplacement(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	}

	valueOf := reflect.ValueOf(value)

	if valueOf.Kind() == reflect.String {
		return reflect.ValueOf(value).String()
	} else if valueOf.Kind() == reflect.Slice {
		var strSlice []string

		// Iterate through the slice elements and convert each to string
		for i := 0; i < valueOf.Len(); i++ {
			strSlice = append(strSlice, formatReplacement(valueOf.Index(i).Interface()))
		}

		return strings.Join(strSlice, ", ")
	} else if valueOf.Kind() == reflect.Map {
		var strSlice []string

		for _, key := range valueOf.MapKeys() {
			// Get the key and value as strings
			keyStr := formatReplacement(key.Interface())
			valueStr := formatReplacement(valueOf.MapIndex(key).Interface())
			strSlice = append(strSlice, fmt.Sprintf("%s: %s", keyStr, valueStr))
		}

		return strings.Join(strSlice, ", ")
	}

	return ""
}

type ContainerOpt func(c *Container)

type MergeStrategy int

const (
	// SkipAndClean will skip existing messages and remove the messages from to that are not in from.
	SkipAndClean MergeStrategy = iota
	// Skip will skip the merge for messages that already exist in to.
	Skip
	// Overwrite will overwrite the messages in to with the messages from from.
	Overwrite
	// OverWriteAndClean will overwrite the messages in to with the messages from from and remove the messages from to that are not in from.
	OverWriteAndClean
)
