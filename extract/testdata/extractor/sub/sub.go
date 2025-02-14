package extractor_sub

import (
	"context"
	"fmt"

	"github.com/SLASH2NL/lingua"
)

var (
	tr      *lingua.Container
	message = tr.Message(context.Background(), "sub.translation", nil)
)

func init() {
	fmt.Println(message)
}
