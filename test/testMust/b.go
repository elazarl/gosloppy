package main

import (
	"crypto/elliptic"
	"fmt"
	"net/url"
)

func init() {
	b[0] = 0
}

var _, _, _ = must(elliptic.GenerateKey(elliptic.P224(), ConstWriter(0)))

func mustStmtExpr() {
	// TODO(elazar): support automatic detection of function's type
	_ = must(fmt.Println("bobo"))
	_ = must(url.Parse("http://example.com"))
}
