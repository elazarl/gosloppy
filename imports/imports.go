// Package imports will find the real package name of a given import path.
//
// Usage:
//
// 	for _, imp := range file.Imports {
// 		println("package name of", imp.Path.Value, "=", imports.GetNameOrGuess(imp))
// 	}
//
// The main motivation of this package is:
//
// Finding out package name of an imort path involves expensive disk access every time.
// Thus,
// (1) Package imports provides a central way to cache package names.
// (2) Package imports precache statically all package names from the Go's standard library, which are commonly used
package imports

import (
	"go/ast"
	"go/build"
	"log"
	"strings"
)

type ImportCache map[string]string

// will get the package name, or guess it if absent
func (cache ImportCache) GetNameOrGuess(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	if rv, ok := cache[imp.Path.Value]; ok {
		return rv
	}
	rv := getNameOrGuess(imp)
	cache[imp.Path.Value] = rv
	return rv
}

// GetNameOrGuess returns the package name of import spec imp
func GetNameOrGuess(imp *ast.ImportSpec) string {
	return DefaultImportCache.GetNameOrGuess(imp)
}

func getNameOrGuess(imp *ast.ImportSpec) string {
	// remove quotes
	path := imp.Path.Value[1 : len(imp.Path.Value)-1]
	pkg, err := build.Import(path, ".", build.AllowBinary)
	if err != nil {
		parts := strings.Split(path, "/")
		rv := parts[len(parts)-1]
		// I don't want to fail if I can't find the package
		// maybe the user is smarter than me, so I guess it's name
		// TODO(elazar): This shouldn't be in general library
		log.Println("Cannot find package", path, "guessing it's name is", rv)
		return rv
	}
	return pkg.Name
}

/*
# to generate run
(cd $GOROOT/src/pkg; bash -c 'find * -type d' | grep -v testdata | \
python -c 'import sys;import glob;from collections import defaultdict
d = defaultdict(list)
lst = sys.stdin.read().split()
print "var Stdlib = map[string]string{"
m = {}
for l in lst:
    d[l.split("/")[-1]].append(l)
    if not glob.glob(l+"/*.go"): continue
    print """\t`"%s"`: "%s",""" % (l, l.split("/")[-1])
print "}"
print "var RevStdlib = map[string][]string{"
for k, v in d.iteritems():
    print """\t"%s": []string{`"%s"`},""" % (k, "\"`,`\"".join(v))
print "}"')
*/
// all stdlib precached, from rev 5b76706b55af (probably Go 1.1)
var DefaultImportCache = ImportCache(Stdlib)

var Stdlib = map[string]string{
	`"archive/tar"`:         "tar",
	`"archive/zip"`:         "zip",
	`"bufio"`:               "bufio",
	`"builtin"`:             "builtin",
	`"bytes"`:               "bytes",
	`"compress/gzip"`:       "gzip",
	`"compress/zlib"`:       "zlib",
	`"compress/lzw"`:        "lzw",
	`"compress/flate"`:      "flate",
	`"compress/bzip2"`:      "bzip2",
	`"container/list"`:      "list",
	`"container/heap"`:      "heap",
	`"container/ring"`:      "ring",
	`"crypto"`:              "crypto",
	`"crypto/dsa"`:          "dsa",
	`"crypto/rand"`:         "rand",
	`"crypto/tls"`:          "tls",
	`"crypto/ecdsa"`:        "ecdsa",
	`"crypto/x509"`:         "x509",
	`"crypto/x509/pkix"`:    "pkix",
	`"crypto/subtle"`:       "subtle",
	`"crypto/md5"`:          "md5",
	`"crypto/elliptic"`:     "elliptic",
	`"crypto/rsa"`:          "rsa",
	`"crypto/hmac"`:         "hmac",
	`"crypto/cipher"`:       "cipher",
	`"crypto/sha512"`:       "sha512",
	`"crypto/sha1"`:         "sha1",
	`"crypto/rc4"`:          "rc4",
	`"crypto/aes"`:          "aes",
	`"crypto/sha256"`:       "sha256",
	`"crypto/des"`:          "des",
	`"database/sql"`:        "sql",
	`"database/sql/driver"`: "driver",
	`"debug/pe"`:            "pe",
	`"debug/dwarf"`:         "dwarf",
	`"debug/macho"`:         "macho",
	`"debug/elf"`:           "elf",
	`"debug/gosym"`:         "gosym",
	`"encoding/pem"`:        "pem",
	`"encoding/ascii85"`:    "ascii85",
	`"encoding/base32"`:     "base32",
	`"encoding/gob"`:        "gob",
	`"encoding/xml"`:        "xml",
	`"encoding/asn1"`:       "asn1",
	`"encoding/base64"`:     "base64",
	`"encoding/json"`:       "json",
	`"encoding/hex"`:        "hex",
	`"encoding/binary"`:     "binary",
	`"encoding/csv"`:        "csv",
	`"errors"`:              "errors",
	`"expvar"`:              "expvar",
	`"flag"`:                "flag",
	`"fmt"`:                 "fmt",
	`"go/parser"`:           "parser",
	`"go/doc"`:              "doc",
	`"go/ast"`:              "ast",
	`"go/types"`:            "types",
	`"go/scanner"`:          "scanner",
	`"go/token"`:            "token",
	`"go/printer"`:          "printer",
	`"go/build"`:            "build",
	`"go/format"`:           "format",
	`"hash"`:                "hash",
	`"hash/adler32"`:        "adler32",
	`"hash/crc32"`:          "crc32",
	`"hash/crc64"`:          "crc64",
	`"hash/fnv"`:            "fnv",
	`"html"`:                "html",
	`"html/template"`:       "template",
	`"image"`:               "image",
	`"image/png"`:           "png",
	`"image/draw"`:          "draw",
	`"image/jpeg"`:          "jpeg",
	`"image/color"`:         "color",
	`"image/gif"`:           "gif",
	`"index/suffixarray"`:   "suffixarray",
	`"io"`:                  "io",
	`"io/ioutil"`:           "ioutil",
	`"log"`:                 "log",
	`"log/syslog"`:          "syslog",
	`"math"`:                "math",
	`"math/rand"`:           "rand",
	`"math/cmplx"`:          "cmplx",
	`"math/big"`:            "big",
	`"mime"`:                "mime",
	`"mime/multipart"`:      "multipart",
	`"net"`:                 "net",
	`"net/mail"`:            "mail",
	`"net/http"`:            "http",
	`"net/http/cgi"`:        "cgi",
	`"net/http/pprof"`:      "pprof",
	`"net/http/fcgi"`:       "fcgi",
	`"net/http/httputil"`:   "httputil",
	`"net/http/cookiejar"`:  "cookiejar",
	`"net/http/httptest"`:   "httptest",
	`"net/rpc"`:             "rpc",
	`"net/rpc/jsonrpc"`:     "jsonrpc",
	`"net/textproto"`:       "textproto",
	`"net/smtp"`:            "smtp",
	`"net/url"`:             "url",
	`"os"`:                  "os",
	`"os/user"`:             "user",
	`"os/exec"`:             "exec",
	`"os/signal"`:           "signal",
	`"path"`:                "path",
	`"path/filepath"`:       "filepath",
	`"reflect"`:             "reflect",
	`"regexp"`:              "regexp",
	`"regexp/syntax"`:       "syntax",
	`"runtime"`:             "runtime",
	`"runtime/debug"`:       "debug",
	`"runtime/race"`:        "race",
	`"runtime/pprof"`:       "pprof",
	`"runtime/cgo"`:         "cgo",
	`"sort"`:                "sort",
	`"strconv"`:             "strconv",
	`"strings"`:             "strings",
	`"sync"`:                "sync",
	`"sync/atomic"`:         "atomic",
	`"syscall"`:             "syscall",
	`"testing"`:             "testing",
	`"testing/iotest"`:      "iotest",
	`"testing/quick"`:       "quick",
	`"text/template"`:       "template",
	`"text/template/parse"`: "parse",
	`"text/scanner"`:        "scanner",
	`"text/tabwriter"`:      "tabwriter",
	`"time"`:                "time",
	`"unicode"`:             "unicode",
	`"unicode/utf8"`:        "utf8",
	`"unicode/utf16"`:       "utf16",
	`"unsafe"`:              "unsafe",
}

var RevStdlib = map[string][]string{
	"text":        []string{`"text"`},
	"jpeg":        []string{`"image/jpeg"`},
	"syntax":      []string{`"regexp/syntax"`},
	"fcgi":        []string{`"net/http/fcgi"`},
	"atomic":      []string{`"sync/atomic"`},
	"unicode":     []string{`"unicode"`},
	"go":          []string{`"go"`},
	"subtle":      []string{`"crypto/subtle"`},
	"xml":         []string{`"encoding/xml"`},
	"base64":      []string{`"encoding/base64"`},
	"elf":         []string{`"debug/elf"`},
	"asn1":        []string{`"encoding/asn1"`},
	"pkix":        []string{`"crypto/x509/pkix"`},
	"cmplx":       []string{`"math/cmplx"`},
	"elliptic":    []string{`"crypto/elliptic"`},
	"mail":        []string{`"net/mail"`},
	"macho":       []string{`"debug/macho"`},
	"format":      []string{`"go/format"`},
	"big":         []string{`"math/big"`},
	"lzw":         []string{`"compress/lzw"`},
	"net":         []string{`"net"`},
	"aes":         []string{`"crypto/aes"`},
	"signal":      []string{`"os/signal"`},
	"ascii85":     []string{`"encoding/ascii85"`},
	"list":        []string{`"container/list"`},
	"crypto":      []string{`"crypto"`},
	"token":       []string{`"go/token"`},
	"race":        []string{`"runtime/race"`},
	"httptest":    []string{`"net/http/httptest"`},
	"bufio":       []string{`"bufio"`},
	"debug":       []string{`"debug"`, `"runtime/debug"`},
	"utf16":       []string{`"unicode/utf16"`},
	"zlib":        []string{`"compress/zlib"`},
	"bytes":       []string{`"bytes"`},
	"testing":     []string{`"testing"`},
	"sync":        []string{`"sync"`},
	"syslog":      []string{`"log/syslog"`},
	"multipart":   []string{`"mime/multipart"`},
	"index":       []string{`"index"`},
	"errors":      []string{`"errors"`},
	"container":   []string{`"container"`},
	"cgo":         []string{`"runtime/cgo"`},
	"gob":         []string{`"encoding/gob"`},
	"pem":         []string{`"encoding/pem"`},
	"template":    []string{`"html/template"`, `"text/template"`},
	"expvar":      []string{`"expvar"`},
	"math":        []string{`"math"`},
	"dsa":         []string{`"crypto/dsa"`},
	"cgi":         []string{`"net/http/cgi"`},
	"gosym":       []string{`"debug/gosym"`},
	"hash":        []string{`"hash"`},
	"dwarf":       []string{`"debug/dwarf"`},
	"ioutil":      []string{`"io/ioutil"`},
	"ast":         []string{`"go/ast"`},
	"compress":    []string{`"compress"`},
	"strconv":     []string{`"strconv"`},
	"quick":       []string{`"testing/quick"`},
	"mime":        []string{`"mime"`},
	"base32":      []string{`"encoding/base32"`},
	"crc32":       []string{`"hash/crc32"`},
	"path":        []string{`"path"`},
	"md5":         []string{`"crypto/md5"`},
	"tls":         []string{`"crypto/tls"`},
	"fnv":         []string{`"hash/fnv"`},
	"jsonrpc":     []string{`"net/rpc/jsonrpc"`},
	"runtime":     []string{`"runtime"`},
	"os":          []string{`"os"`},
	"iotest":      []string{`"testing/iotest"`},
	"rand":        []string{`"crypto/rand"`, `"math/rand"`},
	"encoding":    []string{`"encoding"`},
	"color":       []string{`"image/color"`},
	"image":       []string{`"image"`},
	"rpc":         []string{`"net/rpc"`},
	"regexp":      []string{`"regexp"`},
	"ring":        []string{`"container/ring"`},
	"cookiejar":   []string{`"net/http/cookiejar"`},
	"log":         []string{`"log"`},
	"zip":         []string{`"archive/zip"`},
	"fmt":         []string{`"fmt"`},
	"hex":         []string{`"encoding/hex"`},
	"gif":         []string{`"image/gif"`},
	"json":        []string{`"encoding/json"`},
	"pe":          []string{`"debug/pe"`},
	"sha512":      []string{`"crypto/sha512"`},
	"ecdsa":       []string{`"crypto/ecdsa"`},
	"sort":        []string{`"sort"`},
	"adler32":     []string{`"hash/adler32"`},
	"unsafe":      []string{`"unsafe"`},
	"rsa":         []string{`"crypto/rsa"`},
	"flag":        []string{`"flag"`},
	"heap":        []string{`"container/heap"`},
	"tabwriter":   []string{`"text/tabwriter"`},
	"png":         []string{`"image/png"`},
	"sha256":      []string{`"crypto/sha256"`},
	"rc4":         []string{`"crypto/rc4"`},
	"des":         []string{`"crypto/des"`},
	"flate":       []string{`"compress/flate"`},
	"scanner":     []string{`"go/scanner"`, `"text/scanner"`},
	"tar":         []string{`"archive/tar"`},
	"syscall":     []string{`"syscall"`},
	"parser":      []string{`"go/parser"`},
	"smtp":        []string{`"net/smtp"`},
	"parse":       []string{`"text/template/parse"`},
	"crc64":       []string{`"hash/crc64"`},
	"io":          []string{`"io"`},
	"textproto":   []string{`"net/textproto"`},
	"httputil":    []string{`"net/http/httputil"`},
	"archive":     []string{`"archive"`},
	"binary":      []string{`"encoding/binary"`},
	"bzip2":       []string{`"compress/bzip2"`},
	"filepath":    []string{`"path/filepath"`},
	"pprof":       []string{`"net/http/pprof"`, `"runtime/pprof"`},
	"builtin":     []string{`"builtin"`},
	"html":        []string{`"html"`},
	"build":       []string{`"go/build"`},
	"csv":         []string{`"encoding/csv"`},
	"draw":        []string{`"image/draw"`},
	"printer":     []string{`"go/printer"`},
	"http":        []string{`"net/http"`},
	"exec":        []string{`"os/exec"`},
	"x509":        []string{`"crypto/x509"`},
	"utf8":        []string{`"unicode/utf8"`},
	"driver":      []string{`"database/sql/driver"`},
	"reflect":     []string{`"reflect"`},
	"cipher":      []string{`"crypto/cipher"`},
	"user":        []string{`"os/user"`},
	"sql":         []string{`"database/sql"`},
	"suffixarray": []string{`"index/suffixarray"`},
	"types":       []string{`"go/types"`},
	"sha1":        []string{`"crypto/sha1"`},
	"database":    []string{`"database"`},
	"url":         []string{`"net/url"`},
	"doc":         []string{`"go/doc"`},
	"strings":     []string{`"strings"`},
	"time":        []string{`"time"`},
	"gzip":        []string{`"compress/gzip"`},
	"hmac":        []string{`"crypto/hmac"`},
}
