package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	source := "I have :count|plural(=0 {No apples} =1-2 {Apple} other {# apples (# that is)}) and will attempt to also get more :fruit|capitalize and :cookie|replace|capitalize."

	message, err := Parse(source)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for _, op := range message.Ops {
		t.Logf("Op: %T %+v", op, op)
	}

	require.Len(t, message.Ops, 7)

	require.Equal(t, message.Ops[0], LiteralOp{Value: "I have "})

	replacement, ok := message.Ops[1].(ReplacementOp)
	require.True(t, ok)
	require.Equal(t, replacement.Key, "count")

	plural, ok := replacement.Transformers[0].(PluralTransformer)
	require.True(t, ok)
	require.Len(t, plural.Cases, 2)
	require.Len(t, plural.Other, 4)
}
