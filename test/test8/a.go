package main

import "fmt"

func main() {
	for i := 0; ; {
		break
	}
	a := 0
	for b := a; ; {
		break
	}
	for i := 1; ; {
		b := 1
		break
	}
	switch a := interface{}(a).(type) {
	case int:
	case string:
		println("WRONG")
	}
	switch a := a; a {
	// comment
	/* comment */
	case 100:
	}
	switch a := a; a {
	// comment
	/* comment */
	}
	if b := a; a != a {
	}
	for a := range []string{} {
	}
	c := make(chan int, 1)
	c <- 1
	select {
	case i, ok := <-c:
	case a = <-c:
	case <-c:
		b := 1
	case s := <-c:
		b := 1
	}
	fmt.Println("SUCCESS")
}
