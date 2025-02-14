package lingua

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewContainerNoMatches(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/no_matches")

	c, err := ContainerFromFs(fs)
	require.NoError(t, err)
	require.Empty(t, c.messages)
}

func TestNewContainerInvalidLanguage(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/invalid-language")

	_, err := ContainerFromFs(fs)
	t.Log(err)
	require.Error(t, err)
}

func TestNewContainerInvalidLanguageFile(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/invalid-file")

	_, err := ContainerFromFs(fs)
	t.Log(err)
	require.Error(t, err)
}

func TestNewContainerValid(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/valid")

	c, err := ContainerFromFs(fs)
	require.NoError(t, err)

	ctx := WithLanguage(context.Background(), "en")
	require.Equal(t, "Welcome John", c.Message(ctx, "welcome.login", map[string]any{"user": "john"}))

	// Test field and capitalize transformers.
	require.Equal(t, "First name is required", c.Message(ctx, "required", map[string]any{"attribute": "first_name"}))
	t.Log(c.Message(ctx, "required", map[string]any{"attribute": "first_name"}))

	// Test plural.
	require.Equal(t, "There are a few results", c.Message(ctx, "plural.test", map[string]any{"count": 1}))
	require.Equal(t, "There are a few results", c.Message(ctx, "plural.test", map[string]any{"count": 5}))

	require.Equal(t, "There are 500 results", c.Message(ctx, "plural.test", map[string]any{"count": 500}))
	require.Equal(t, "There are no results", c.Message(ctx, "plural.test", map[string]any{"count": 0}))
}

func TestContainerRaw(t *testing.T) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/valid")

	c, err := ContainerFromFs(fs)
	require.NoError(t, err)

	raw := c.Raw()

	require.Contains(t, raw, LanguageID{Language: "nl"})
	require.Contains(t, raw, LanguageID{Language: "en", Region: "US"})

	en := raw[LanguageID{Language: "en", Region: "US"}]
	require.Contains(t, en, "welcome.login")
	require.Equal(t, "Welcome :user|capitalize", en["welcome.login"])
}

func TestContainerMergeSkip(t *testing.T) {
	from, to := writeMergeYaml(t)

	result := Merge(from, to, Skip)

	raw := result.Raw()
	require.Contains(t, raw, LanguageID{Language: "en"})
	require.Contains(t, raw[LanguageID{Language: "en"}], "other")
	require.Contains(t, raw[LanguageID{Language: "en"}], "pizza")
	require.Equal(t, "Welcome :user", raw[LanguageID{Language: "en"}]["welcome.login"])
}

func TestContainerMergeSkipClean(t *testing.T) {
	from, to := writeMergeYaml(t)

	result := Merge(from, to, SkipAndClean)

	raw := result.Raw()
	require.Contains(t, raw, LanguageID{Language: "en"})
	require.Contains(t, raw[LanguageID{Language: "en"}], "other")
	require.NotContains(t, raw[LanguageID{Language: "en"}], "pizza")
	require.Equal(t, "Welcome :user", raw[LanguageID{Language: "en"}]["welcome.login"])
}

func TestContainerMergeOverwrite(t *testing.T) {
	from, to := writeMergeYaml(t)

	result := Merge(from, to, Overwrite)

	raw := result.Raw()
	require.Contains(t, raw, LanguageID{Language: "en"})
	require.Contains(t, raw[LanguageID{Language: "en"}], "other")
	require.Contains(t, raw[LanguageID{Language: "en"}], "pizza")
	require.Equal(t, "Welcome :user|capitalize", raw[LanguageID{Language: "en"}]["welcome.login"])
}

func TestContainerMergeOverwriteAndClean(t *testing.T) {
	from, to := writeMergeYaml(t)

	result := Merge(from, to, OverWriteAndClean)

	raw := result.Raw()
	require.Contains(t, raw, LanguageID{Language: "en"})
	require.Contains(t, raw[LanguageID{Language: "en"}], "other")
	require.NotContains(t, raw[LanguageID{Language: "en"}], "pizza")
	require.Equal(t, "Welcome :user|capitalize", raw[LanguageID{Language: "en"}]["welcome.login"])
}

func writeMergeYaml(t *testing.T) (from, to *Container) {
	fs := afero.NewMemMapFs()
	mustWriteYaml(t, fs, "en.yaml", `
welcome.login: Welcome :user|capitalize
other: Other
`)

	mustWriteYaml(t, fs, "nl.yaml", `
`)

	from, err := ContainerFromFs(fs)
	require.NoError(t, err)

	fs = afero.NewMemMapFs()
	mustWriteYaml(t, fs, "en.yaml", `
welcome.login: Welcome :user
pizza: Pizza
`)

	to, err = ContainerFromFs(fs)
	require.NoError(t, err)

	return from, to
}

func mustWriteYaml(t *testing.T, fs afero.Fs, name string, content string) {
	f, err := fs.Create(name)
	require.NoError(t, err)

	_, err = f.Write([]byte(content))
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)
}

func BenchmarkFormat(b *testing.B) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/valid")

	c, err := ContainerFromFs(fs)
	require.NoError(b, err)

	ctx := WithLanguage(context.Background(), "en")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Message(ctx, "required", map[string]any{"attribute": "first_name"})
	}
}

func BenchmarkFormatPlural(b *testing.B) {
	fs := afero.NewBasePathFs(afero.NewOsFs(), "./testdata/valid")

	c, err := ContainerFromFs(fs)
	require.NoError(b, err)

	ctx := WithLanguage(context.Background(), "en")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Message(ctx, "plural.test", map[string]any{"count": 4})
	}
}
