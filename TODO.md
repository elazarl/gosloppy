# TODO

## Infra

[ ] Replace the blasphemy ./test/all.test is, with a python/go script

[ ] Move dir("d", file("name", "content")) into its own package.

## `$GOPATH` and subpackages

[V] Make sure subpackages works right. DONE
    a. If you know your import path - instrument packages below you, or directly above you
       (if you're "github.com/foo/bar", then if strings.HasPrefix(x, "github.com/foo/bar"), or
       or strings.HasPrefix("github.com/foo/bar", x), package will be instrumented.
    b. If not, compile only relative subpackages. strings.HasPrefix(x, "./")

## Performance

[V] Package cache - a must before release.

[-] Should a package cache persist itself? Probably not worth it. - WONTFIX

[V] Permanent cache of standard packages.

[ ] Some light performance tests. If you can see by hand it's working reasonably fast, maybe we can skip them.

## Usability

[ ] support gosloppy run file1.go file2.go

[ ] Support gosloppy sloppify

[V] Take into account package namespace.

    $ cat a.go
    package foo
    var io = struct{koko string}
    $ cat b.go
    var x = io.koko // will trigger import of io

[V] Auto import of packages from the standard library. Must before release.

[V] Easy way to panic on error. Must before release. e.g.

    result := must(f())
    // equiv:
    result, __temp := f()
    if __temp := err { panic("filename:linenumber", err)
   
[ ] Easy way to log errors

    orlog(os.Getwd())
    // equiv:
    __temp := os.Getwd()
    if __temp := err { log.Println("filename:linenumber", err)

[ ] Should we support script mode?

    #!/bin/bash -c '$GOPATH/bin/gosloppy'
    fmt.Println
    
[ ] Show warnings for unused variables? (maybe you should just use `go build` for that).
