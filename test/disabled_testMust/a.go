package main

import (
	"crypto/elliptic"
	"fmt"
	"net/url"
)

type ConstWriter byte

func (w ConstWriter) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		b[i] = byte(w)
	}
	return len(b), nil
}

var b, x, y = must(elliptic.GenerateKey(elliptic.P224(), ConstWriter(0)))

func main() {
	u := must(url.Parse("http://SUCCESS"))
	b, x, y := must(elliptic.GenerateKey(elliptic.P224(), ConstWriter(0)))
	if must(url.Parse("http://example.com/a/b")).IsAbs() {
		fmt.Println(u.Host)
	}
}
