package main

import "os"

func f(name string) bool {
	stat, err := os.Stat(name)
	return err != nil
}

func main() {
	println("SUCCESS")
}
