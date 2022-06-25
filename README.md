# GoSloppy - Go for prototypes

## Alpha Software

This is a very preliminary version, distributed for the sake of being tested,
it is far from a final product. ~~Hopefully this will change soon~~. Nope, wouldn't change soone

## Goal

There were [many](https://groups.google.com/d/msg/golang-nuts/OBsCksYHPG4/55Cn7nufGXMJ)
[complains](http://uberpython.wordpress.com/2012/09/23/why-im-not-leaving-python-for-go/)
[about](https://plus.google.com/u/0/100144763948435845718/posts/8Bn2gRykVzN)
the unsuitablility of Go (golang) for small prototype projects.

The main issue was, Go will not compile source files with unused imports or unused variables.

This project will give you a quick way to run and test your package without modifying it,
even if it contains unused variables or imports.

This is superior to the `var _ = unused` techinque, since you will never forget to remove
the `var _ =` statements. If you do - the package will simply not compile.

## Quick Intro

    $ go get github.com/elazarl/gosloppy
    $ # make sure $GOPATH/bin is in your path
    $ export PATH=$PATH:${GOPATH//://bin:}/bin
    $ mkdir /tmp/pkg; cd /tmp/pkg
    $ echo 'package main;import "fmt";func main(){i := 1;println("no fmt, yet compiles")}' > a.go
    $ go build
    ./a.go:1: imported and not used: "fmt"
    $ gosloppy build
    $ ./pkg
    no fmt, yet compiles

You can use different executable name:

    $ gosloppy build -o koko
    $ ./koko
    no fmt, yet compiles

Tests also works:

    $ echo 'package main;import "fmt"' > a_test.go
    $ go test
    ./a.go:1: imported and not used: "fmt"
    ./a_test.go:1: imported and not used: "fmt"
    FAIL	_/tmp/pkg [build failed]
    $ gosloppy test
    testing: warning: no tests to run
    PASS
    ok  	_/private/tmp/pkg/__instrument.go555768202	0.019s

Just for the sake of the exposition, let's see unused variable alone.

    $ rm -f *
    $ echo 'package main;func main() { i := 1; println("unused, yet works") }' > a.go
    $ go build
    ./a.go:1: i declared and not used
    $ gosloppy build
    $ ./pkg
    unused, yet works

## Fragmentation of the Go Ecosystem

Would it fragment the Go ecosystem? I think not. GoSloppy, by design, will not be able
to install a package. Only to run it temporarily in your own sandbox.

If you want to publish a library, or get it with `go get`, GoSloppy will refuse to help
you.

The idea is - do whatever you want in the privacy of your sandbox. When you publish your
code - make sure it conform to [Go's spec](http://golang.org/ref/spec).

## How It Works

### Birds Eye View

GoSloppy will parse your package source file, search for unused variables and packages, and insert
`var _ = unused` where appropriate.

GoSloppy would then write the patched file to a temporary directory prefixed with `__gosloppy.go`, and will
run `go build` there. It will never insert a `\n`, so errors reported will still have correct line information.

Finally, it'll copy the resulting file to your current directory.

GoSloppy will try to guess which included packages should be also compiles, and instrument them in a similar
fashion. For example, all relative imports, will also be "sloppified" and compiled when running `gosloppy`.
