package lingua

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLanguageCtx(t *testing.T) {
	cases := []struct {
		input         string
		expectDefault bool
		language      string
		region        string
	}{
		{
			input:         "en",
			expectDefault: false,
			language:      "en",
		},
		{
			input:         "nl",
			expectDefault: false,
			language:      "nl",
		},
		{
			input:         "en-US",
			expectDefault: false,
			language:      "en",
			region:        "US",
		},
		{
			input:         "en_GB",
			expectDefault: false,
			language:      "en",
			region:        "GB",
		},
		{
			input:         "invalid",
			expectDefault: true,
		},
		{
			// An accept header.
			input:    "en-GB,en;q=0.5",
			language: "en",
			region:   "GB",
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			ctx := WithLanguage(context.Background(), c.input)
			if c.expectDefault {
				require.True(t, fromCtx(ctx).Empty())
				return
			}

			lang := fromCtx(ctx)
			require.Equal(t, c.language, lang.Language)
			require.Equal(t, c.region, lang.Region)
		})
	}
}

func TestToCtx(t *testing.T) {
	cases := []struct {
		input         string
		expectDefault bool
		language      string
		region        string
	}{
		{
			input:         "invalid",
			expectDefault: true,
		},
		{
			// An accept header.
			input:    "en-GB,en;q=0.5",
			language: "en",
			region:   "GB",
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			ctx := WithLanguage(context.Background(), c.input)
			if c.expectDefault {
				require.True(t, fromCtx(ctx).Empty())
				return
			}

			require.False(t, fromCtx(ctx).Empty())
			lang := fromCtx(ctx)
			require.Equal(t, c.language, lang.Language)
			require.Equal(t, c.region, lang.Region)
		})
	}
}
