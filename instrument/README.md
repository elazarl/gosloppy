# `gosloppy` instrumentaton framework

## Alpha warning

This repository exists to get early feedback on the design, API *will* change.

## Quickstart

Ever wondered what are the most called functions when testing `net/url`?

    ❯ cat printcalls.go
    package main

    import (
      "go/ast"
      "os"

      "github.com/elazarl/gosloppy/instrument"
      "github.com/elazarl/gosloppy/patch"
    )

    func main() {
      err := instrument.InstrumentCmd(func(p *patch.PatchableFile) (patches patch.Patches) {
        for _, dec := range p.File.Decls {
          if fun, ok := dec.(*ast.FuncDecl); ok && fun.Name != nil && fun.Body != nil {
            patches = append(patches, patch.Insert(fun.Body.Lbrace + 1, "println(`"+fun.Name.Name+"`);"))
          }
        }
        return patches
      }, os.Args...)
      if err != nil {
        println(err.Error())
        os.Exit(-1)
      }
    }
    ❯ go run printcalls.go test -goroot net/url 2>&1 |sort|uniq -c|sort -nr|head
     521 shouldEscape
     385 split
     213 unescape
     149 parse
     147 getscheme
     144 Parse
     115 ishex
     112 unhex
      93 escape
      90 parseAuthority

## Goal and Motivation

Many times one would like to instrument a big piece of software to gain
insight about it. For example:

  * What is the maximal number of concurrent TCP connections my program uses?
  * I want to get an alert if a certain system call is being used.
  * I want a log of all files open during execution of `Foo`.
  * If the integration test writes non-ASCII characters to any file descriptor
    it should fail immediately.

You can answer some of those questions with `strace` and similar tools, but sometimes
instrumentation is the correct answer. A visible benefit for instance is,
 the instrumented program runs cross platform, no need to run `strace` on Linux
`dtruss` on mac and `Process Monitor` on Windows..

The `gosloppy` instrumentation framework allows you to easily instrument and run
any Go program. You can choose to instrument a single Go progran, all code used
by this program in your `$GOPATH`, and even all code in `$GOROOT`.
