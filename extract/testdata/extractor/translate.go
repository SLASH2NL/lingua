package extractor

import (
	"context"
	"fmt"

	"github.com/SLASH2NL/lingua"
)

const (
	usedConst              = "used.const"
	unusedConst lingua.Key = "unused.const"
)

var (
	usedVar              = "used.var"
	unusedVar lingua.Key = "unused.var"

	tr *lingua.Container
)

func UseMessagesTranslate() {
	message := tr.Message(context.Background(), "login.welcome", map[string]any{"user": "john"})
	fmt.Println(message)

	// Use zipcode twice.
	tr.Message(context.Background(), "zipcode", map[string]any{"user": "john"})
	tr.Message(context.Background(), "zipcode", map[string]any{"user": "john"})
	fmt.Println(unusedVar)
}

func UseFunc(ctx context.Context) {
	Translate("use.func", nil)
}

func UseFuncWithConst(ctx context.Context) {
	Translate(usedConst, nil)
}

func UseFuncWithVar(ctx context.Context) {
	Translate(lingua.Key(usedVar), nil)
}

func UseFuncWithInlineVar(ctx context.Context) {
	var translation lingua.Key = "inline.var"

	Translate(translation, nil)
}

func Translate(key lingua.Key, replacements map[string]interface{}) string {
	return string(key)
}

func SameSignature(key string, replacements map[string]interface{}) string {
	return key
}
