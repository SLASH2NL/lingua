package parser

import (
	"fmt"
	"strconv"
	"strings"
)

func Parse(input string) (*Message, error) {
	tokens, err := runLexer(input)
	if err != nil {
		return nil, err
	}

	it := newIterator(tokens)

	msg := &Message{
		Ops: make([]any, 0),
	}

	for it.HasNext() {
		token, ok := it.Next()
		if !ok {
			break
		}

		switch token.TokenType {
		case literal:
			msg.Ops = append(msg.Ops, LiteralOp{Value: token.Data})
		case replacement:
			transformers, err := parseTransformers(it)
			if err != nil {
				return nil, fmt.Errorf("unable to parse transformers: %w", err)
			}

			replacementOp := ReplacementOp{
				Key:          token.Data,
				Transformers: transformers,
			}

			msg.Ops = append(msg.Ops, replacementOp)
		}

	}

	return msg, nil
}

func parseTransformers(it *iterator[Token]) (transformers []any, err error) {
	for it.HasNext() {
		token, ok := it.Peek()
		if !ok {
			return transformers, nil
		}

		if token.TokenType != transformer {
			break
		}

		token, _ = it.Next()

		switch token.Data {
		case "capitalize":
			transformers = append(transformers, CapitalizeTransformer{})
		case "replace":
			transformers = append(transformers, ReplaceTransformer{})
		case "plural":
			plural := PluralTransformer{
				Cases: make([]PluralCase, 0),
			}

			for it.HasNext() {
				// Parse all cases.
				pcase, err := parsePluralCase(it)
				if err != nil {
					return nil, fmt.Errorf("unable to parse plural case: %w", err)
				}

				if pcase == nil {
					break
				}

				if pcase.Type == OpPluralCaseOther {
					plural.Other = pcase.Ops
					continue
				}

				plural.Cases = append(plural.Cases, *pcase)
			}

			if len(plural.Other) == 0 {
				return nil, fmt.Errorf("missing 'other' case for plural transformer")
			}

			transformers = append(transformers, plural)
		}
	}

	return transformers, nil
}

func parsePluralCase(it *iterator[Token]) (*PluralCase, error) {
	pcase := &PluralCase{
		Type: OpPluralCaseTypeExact,
		Ops:  make([]any, 0),
	}

	// Check if there is a case to parse.
	peek, ok := it.Peek()
	if !ok {
		return nil, nil
	}

	if peek.TokenType != pluralNumeric && peek.TokenType != pluralOther {
		return nil, nil
	}

	// First grab the case type and args.
	for it.HasNext() {
		token, ok := it.Peek()
		if !ok {
			return nil, nil
		}

		if token.TokenType == pluralTranslationStart {
			break
		}

		// The first token could be a numeric or the other keyword.
		switch token.TokenType {
		case pluralNumeric:
			// If the type of pcase is range then we have already got the A value, if not we need to set it.
			var err error

			if pcase.Type != OpPluralCaseTypeRange {
				pcase.A, err = strconv.Atoi(token.Data)
				if err != nil {
					return nil, fmt.Errorf("unable to convert A value %q to int: %w", token.Data, err)
				}
			} else {
				pcase.B, err = strconv.Atoi(token.Data)
				if err != nil {
					return nil, fmt.Errorf("unable to convert B value %q to int: %w", token.Data, err)
				}
			}
		case pluralRange:
			pcase.Type = OpPluralCaseTypeRange
		case pluralOther:
			pcase.Type = OpPluralCaseOther
		default:
			return nil, fmt.Errorf("unexpected end of plural case with type %s", token.TokenType) // Some unknown token, we should stop parsing the case.
		}

		// Make sure we continue the iterator.
		it.Next()
	}

	// We expect the following token to be an translation opening.
	token, ok := it.Next()
	if !ok {
		return nil, nil
	}

	if token.TokenType != pluralTranslationStart {
		return nil, fmt.Errorf("expected translation start token, got %s", token.TokenType)
	}

	// Collect all operations until we reach the end of the translation.
	for it.HasNext() {
		token, ok := it.Next()
		if !ok {
			return nil, fmt.Errorf("unexpected end of plural case")
		}

		if token.TokenType == pluralTranslationEnd {
			break
		}

		switch token.TokenType {
		case literal:
			pcase.Ops = append(pcase.Ops, LiteralOp{Value: token.Data})
		case pluralCount:
			pcase.Ops = append(pcase.Ops, PluralCountOp{})
		}
	}

	return pcase, nil
}

type Message struct {
	Ops []any
}

func (m Message) Raw() string {
	var b strings.Builder
	for _, op := range m.Ops {
		switch v := op.(type) {
		case LiteralOp:
			b.WriteString(v.Value)
		case ReplacementOp:
			b.WriteString(":" + v.Key)

			for _, transformer := range v.Transformers {
				b.WriteRune('|')

				switch t := transformer.(type) {
				case CapitalizeTransformer:
					b.WriteString("capitalize")
				case ReplaceTransformer:
					b.WriteString("replace")
				case PluralTransformer:
					b.WriteString("plural")

					b.WriteRune('(')
					for _, c := range t.Cases {
						b.WriteRune('=')

						if c.Type == OpPluralCaseTypeRange {
							b.WriteString(strconv.Itoa(c.A))
							b.WriteRune('-')
							b.WriteString(strconv.Itoa(c.B))
						} else {
							b.WriteString(strconv.Itoa(c.A))
						}

						b.WriteRune(' ')

						b.WriteRune('{')
						for _, c := range c.Ops {
							switch c := c.(type) {
							case LiteralOp:
								b.WriteString(c.Value)
							case PluralCountOp:
								b.WriteRune('#')
							}
						}
						b.WriteRune('}')
					}

					if len(t.Other) > 0 {
						b.WriteString(" other {")
						for _, c := range t.Other {
							switch c := c.(type) {
							case LiteralOp:
								b.WriteString(c.Value)
							case PluralCountOp:
								b.WriteRune('#')
							}
						}
						b.WriteRune('}')
					}

					b.WriteRune(')')
				}
			}
		}
	}

	return b.String()
}

type LiteralOp struct {
	Value string
}

type ReplacementOp struct {
	Key          string
	Transformers []any
}

type CapitalizeTransformer struct{}

type ReplaceTransformer struct{}

type PluralTransformer struct {
	Cases []PluralCase
	Other []any
}

type PluralCase struct {
	Type OpPluralCaseType
	A    int
	B    int

	// Ops is a list of operations that should be applied if the case is true.
	// This can be a list of LiteralOp and PluralCountOp.
	Ops []any
}

func (c PluralCase) Match(count int) bool {
	switch c.Type {
	case OpPluralCaseTypeRange:
		return count >= c.A && count <= c.B
	case OpPluralCaseTypeExact:
		return count == c.A
	}

	return false
}

type PluralCountOp struct{}

type OpPluralCaseType int

const (
	OpPluralCaseTypeRange OpPluralCaseType = iota
	OpPluralCaseTypeExact
	OpPluralCaseOther
)
