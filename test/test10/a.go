package main

import "net/http"

type T struct {
	*http.Request
}

func main() { println("SUCCESS") }
