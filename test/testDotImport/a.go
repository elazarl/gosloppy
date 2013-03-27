package main

/* We will not guard against unused dot imports, but
   we will not fail the build either */
import . "fmt"

var _ = Println

func main() { println("SUCCESS") }
