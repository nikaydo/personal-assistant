package tokenizer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

type Tokenizer struct {
}

func M(model string) {
	m := strings.Split(model, "/")
	fmt.Println(m)
	enc, err := tiktoken.EncodingForModel(m[1])
	if err != nil {
		panic(err)
	}
	n := 0
	for _, i := range enc.Encode("Hello world what b fuck", nil, nil) {
		n += Sum(i)
	}
	fmt.Println(n)
	fmt.Println(enc.Encode("Hello world what b fuck", nil, nil))
}

func Sum(numb int) int {
	g := 0
	f := strconv.Itoa(numb)
	for i := range len(f) {
		s, _ := strconv.Atoi(string(f[i]))
		g += s
	}
	return g
}
