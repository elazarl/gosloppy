package main

func f() (string, error) { return "SUCCESS", nil }

var a = must(f())

func main() {
	println(a)
}
