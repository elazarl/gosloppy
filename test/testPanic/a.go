package main

import (
	"crypto/elliptic"
	"fmt"
	"net/url"
)

type ConstWriter byte

func (w ConstWriter) Read(b []byte) (int, err) {
	for i := 0; i < len(b); i++ {
		b[i] = byte(w)
	}
	return len(b), nil
}

func main() {
	u := must(url.Parse("http://SUCCESS"))
	b, x, y, err := elliptic.GenerateKey(elliptic.P224(), ConstWriter(0))
	fmt.Println(u.Host)
}
