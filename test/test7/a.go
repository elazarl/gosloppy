package main

func main() {
	fmt.Fprint(ioutil.Discard, "holalal")
	fmt.Println("SUCCESS")
}

func f(io struct {
	Closer interface {
		Close() error
	}
},) {
	fmt.Println(io.Closer, os.foo)
}
