package parser

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

type tokenType int8

const (
	eof = -1

	literal tokenType = iota
	replacement
	transformer
	pluralNumeric
	pluralRange
	pluralOther
	pluralTranslationStart
	pluralTranslationEnd
	pluralCount
	errTok

	lowercase = "abcdefghijklmnopqrstuvwxyz"
	uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits    = "0123456789"
	spaces    = " \t\n "
)

func runLexer(input string) ([]Token, error) {
	l := &lexer{
		input:  input,
		tokens: make([]Token, 0),
	}

	for state := lexLiteral; state != nil; {
		state = state(l)
	}

	// Check if the last state was an error.
	if len(l.tokens) > 0 && l.tokens[len(l.tokens)-1].TokenType == errTok {
		return nil, fmt.Errorf("lexer error: %s", l.tokens[len(l.tokens)-1].Data)
	}

	return l.tokens, nil
}

func lexLiteral(l *lexer) lexerStateFn {
	for {
		if l.peek() == eof {
			l.collect(literal)
			return nil
		}

		if l.peek() == ':' {
			l.collect(literal)
			return lexerPlaceholder
		}

		// Check if we are dealing with an escape character that escapes the : sign.
		if l.peek() == '\\' {
			l.next()

			// Check if the next character is a : sign, if so we ignore it.
			if l.peek() == ':' {
				l.next()
			}

			continue
		}

		l.next()
	}
}

func lexerPlaceholder(l *lexer) lexerStateFn {
	l.next() // Collect the ':'

	if !l.accept(lowercase) {
		// We are not dealing with a placeholder but a normal : sign.
		return lexLiteral
	}

	l.backup()
	l.ignore() // Ignore the ':'

	l.acceptRun(lowercase)
	l.collect(replacement)

	// Check if we need to lex a transformer.
	if l.peek() == '|' {
		return lexerTransformer
	}

	return lexLiteral
}

func lexerTransformer(l *lexer) lexerStateFn {
	if l.peek() != '|' {
		return lexLiteral
	}

	l.next() // Collect the '|' and ignore it.
	l.ignore()

	if !l.accept(lowercase) {
		// Lex the invalid character.
		l.next()
		l.error("expected lowercase transformer name")
		return nil
	}

	l.acceptRun(lowercase)

	// Check what kind of transformer we are dealing with.
	transformerType := l.data()

	switch transformerType {
	case "plural":
		if l.peek() != '(' {
			l.error("expected '(' after plural transformer")
			return nil
		}

		l.collect(transformer)
		l.next() // Collect the '('

		return lexerPluralArgs
	case "capitalize", "replace":
		l.collect(transformer)

		// We can chain transformers, so we need to check if there is another transformer.
		return lexerTransformer
	default:
		l.error("unknown transformer")
	}

	return nil
}

func lexerPluralArgs(l *lexer) lexerStateFn {
	// Collect all arguments.
	for {
		// Ignore all whitespace characters between args.
		l.acceptRun(spaces)
		l.ignore()

		// There are no more arguments.
		if l.peek() == ')' {
			l.next() // Collect the ')'
			l.ignore()

			// We can chain transformers, so we need to check if there is another transformer.
			return lexerTransformer
		}

		// Check if we are dealing with a number or with other.
		if l.peek() == '=' {
			return lexerPluralNumericArg
		}

		// Check if we are dealing with 'other'.
		if strings.HasPrefix(l.input[l.pos:], "other") {
			for i := 0; i < 5; i++ {
				l.next()
			}

			l.collect(pluralOther)
			return lexerPluralTranslation
		}

		x := l.next()
		if x == eof {
			l.error("unexpected EOF")
			return nil
		}

	}
}

func lexerPluralNumericArg(l *lexer) lexerStateFn {
	l.next() // Collect the '='
	l.ignore()

	// Expect a number (of at least 1 digit).
	if !l.accept(digits) {
		l.error("expected number after '='")
		return nil
	}
	l.acceptRun(digits) // Collect the rest of the digits.
	l.collect(pluralNumeric)

	// Check if we are dealing with a range.
	if l.peek() == '-' {
		l.next() // Collect the '-'
		l.collect(pluralRange)

		// Expect a number (of at least 1 digit).
		if !l.accept(digits) {
			l.error("expected number after '-'")
			return nil
		}
		l.acceptRun(digits) // Collect the rest of the digits.
		l.collect(pluralNumeric)
	}

	return lexerPluralTranslation
}

func lexerPluralTranslation(l *lexer) lexerStateFn {
	// Collect all whitespace chars and ignore them.
	// Now we expect the translation.
	l.acceptRun(spaces)
	l.ignore()

	if l.peek() != '{' {
		l.error("expected '{' for plural translation start")
		return nil
	}

	l.next() // Collect the '{'
	l.collect(pluralTranslationStart)

	// Collect the translation.
	for {
		// Check if we have reached the end of the translation.
		if l.peek() == '}' {
			// Collect the current buffer as a literal.
			l.collect(literal)

			l.next() // Collect the '}'
			l.collect(pluralTranslationEnd)

			// Continue and try to parse the next plural argument.
			return lexerPluralArgs
		}

		n := l.next()
		if n == eof {
			l.error("unexpected EOF")
			return nil
		}

		if n == '#' {
			l.backup()
			l.collect(literal)

			l.next()
			l.collect(pluralCount)
		}
	}
}

type lexer struct {
	input  string  // the string being scanned
	start  int     // start position of this item
	pos    int     // current position in the input
	width  int     // width of last rune read from input
	tokens []Token // slice of tokens
}

type lexerStateFn func(*lexer) lexerStateFn

// collect the current data as a new token on the tokens slice.
func (l *lexer) collect(t tokenType) {
	if l.start == l.pos {
		return
	}

	l.tokens = append(l.tokens, Token{
		TokenType: t,
		Data:      l.input[l.start:l.pos],
	})
	l.start = l.pos
}

func (l *lexer) data() string {
	return l.input[l.start:l.pos]
}

func (l *lexer) error(msg string) {
	dataSample := l.input[0:l.pos]

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	if isTerminal {
		if len(dataSample) > 0 {
			dataSample = fmt.Sprintf("%s\033[4m\033[1;31m%s\033[0m", dataSample[:l.start], dataSample[l.start:l.pos])
		}
	}

	errorMsg := fmt.Sprintf("%s at position %d (%s)", msg, l.pos, dataSample)

	l.tokens = append(l.tokens, Token{
		TokenType: errTok,
		Data:      errorMsg,
	})
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

// accept consumes the next rune
// if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

type Token struct {
	TokenType tokenType
	Data      string
}

func (t tokenType) String() string {
	switch t {
	case literal:
		return "literal"
	case replacement:
		return "replacement"
	case transformer:
		return "transformer"
	case pluralNumeric:
		return "pluralNumeric"
	case pluralRange:
		return "pluralRange"
	case pluralOther:
		return "pluralOther"
	case pluralTranslationStart:
		return "pluralTranslationStart"
	case pluralTranslationEnd:
		return "pluralTranslationEnd"
	case pluralCount:
		return "pluralCount"
	case errTok:
		return "ERR"
	default:
		return "unknown"
	}
}
