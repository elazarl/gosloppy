package main

import "testing"

func TestMe(t *testing.T) {
	i := 1
	if !f("mustexist") {
		t.Fatal("Expected file 'mustexist', but found none")
	}
}
