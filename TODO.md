# TODO

## `$GOPATH` and subpackages - DONE

  1. Make sure subpackages works right.
    a. Is an import path corresponds to you - BAM you found the base path.
    b. If not, does an import path import a package name with identical name to a subpackage?
       i. Issue a warning if it's compilable (ie package exists).
       ii. treat as a base path if it isn't.

## Performance

Package cache - a must before release. - DONE

Should a package cache persist itself? Probably not worth it. - WONTFIX

Permanent cache of standard packages.

Some light performance tests. If you can see by hand it's working reasonably fast, maybe we can skip them.

## Usability

Auto import of packages from the standard library. Must before release.

Easy way to panic on error. Must before release. e.g.

    result, panic := f()
    // equiv:
    result, __temp := f()
    if __temp := err { panic("filename:linenumber", err)
    
    log.Println := os.Getwd()
    // equiv:
    __temp := os.Getwd()
    if __temp := err { log.Println("filename:linenumber", err)

Should we support script mode?

    #!/bin/bash -c '$GOPATH/bin/gosloppy'
    fmt.Println("hello world")
