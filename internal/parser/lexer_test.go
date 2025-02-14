package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	source := "I have :count|plural(=0 {No apples} =1-2 {Apple} other {# apples}) and will attempt to also get more :fruit|capitalize and :cookie|replace|capitalize."

	tokens, err := runLexer(source)
	require.NoError(t, err)

	require.Len(t, tokens, 26)
	require.Equal(t, tokens[0].TokenType, literal)
	require.Equal(t, tokens[1].TokenType, replacement)
	require.Equal(t, tokens[2].TokenType, transformer)

	for _, token := range tokens {
		t.Logf("Type: %s, Data: %q", token.TokenType, token.Data)
	}
}
