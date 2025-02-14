# Lingua
This is a simple translation library for golang. It uses yaml files to load translations. Files can be specific to a language or a combination of language and region.

Translations are simple key value pairs. The key is used to identify the translation and the value is the translated message. The value can contain placeholders that will be replaced with the values provided.

```yaml
# Example translation line with a simple replacement.
# Calling this with "user" => "john" will result in "Welcome john!"
welcome.message: "Welcome :user!"
```

## Transformers
Transformers can be used to modify the replacement value before it is inserted into the translation message.
There are 3 built-in transformers:
- capitalize: Capitalizes the replacement value.
- replace: Uses the placeholder to find a translation message. This is usefull for translating generic error messages for validation.
- plural: Uses the replacement value to determine the plural form of the translation message.

### Capitalize
```yaml
# Calling this with "user" => "john" will result in "Welcome John!"
capitalize: "Welcome :user|capitalize"
```

### Replace
```yaml
user.email: "email address"
street: "street"

# Calling this with "field" => "user.email" will result in "Email address is a required field"
# Note the chaining of the transformers. The first transformer is applied to the replacement value and the second transformer is applied to the result of the first transformer.
required: ":field|replace|capitalize is a required field"
```

### Plural
```yaml
# Calling this with "count" => 0 will result in "no items"
# Calling this with "count" => 1 will result in "1 item"
# Calling this with "count" => 5 will result in "a few items"
# Calling this with "count" => 15 will result in "15 items"
items: ":count|plural(=0 {no items} =1 {1 item} =2-10 {a few items} other {# items})"
```

## Message extraction
Users can use the lingua tool to extract translation keys from your go source files. This will collect every value of type github.com/SLASH2NL/lingua.Key from
the src directory.

Install the tool by running:
```bash
$ go install github.com/SLASH2NL/lingua/cmd/lingua@latest
```

```bash
# Extract translation keys from go source files and write them to translation files in dst.
# If there are existing files they will be merged and only new keys will be added.
#
# Note: The tool will only write to language files that exist in the translation directory.
# If you want to add a new language add en empty yaml file like `$ touch en.yaml`.
lingua extract path_to_go_source_files path_to_translation_files
```

What does it extact?
- const values `const translation lingua.Key = "const.translation"`
- var values `var translation lingua.Key = "var.translation"`
- function calls that provide a lingua.Key as argument `myFunc("func.call") where myFunc is defined as func(msg lingua.Key)`

## Usage
Lingua parses translation files from a filesystem(anything that implements afero.Fs). Files should follow the following naming convention to be recognized by lingua:
- en.yaml (language only)
- or en-US.yaml (language and region)

Empty files are allowed and will also be parsed. This can be useful for adding a new language and prefill it with the keys found by `lingua extract`.

```go
// Parse translations from fs (this can be any filesystem that implements the afero.FS interface).
// This can be an embed.FS or a local filesystem.
// Adding a default language to use as fallback.
c, err := lingua.ContainerFromFs(afero.NewBasePathFs(afero.NewOsFs(), "./translations"), lingua.WithDefaultLanguage(lingua.MustParseLanguage("en")))
if err != nil {
    // Handle error...
}

// On each request set the preferred language in the context.
// Parse the language from a user request. This can be from a header or user settings, for example the http Accept-Language header.
ctx := context.Background()
ctx, err := lingua.WithLanguage(ctx, r.Header.Get("Accept-Language"))
if err != nil {
    // Handle error...
}

// Translate the message.
// If the user requested "en-US" but you only have "en" translations available, the translator will use the "en" translations.
msg := c.Message(ctx, "welcome.message", map[string]any{"user": "wvell"})
fmt.Println(msg) // prints: Welcome wvell!
```