package main

// note, we should recognize libtest even though its name is pkg
import "./libtest"

func main() {
	pkg.F(1)
}
