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
