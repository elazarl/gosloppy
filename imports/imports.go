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
	if rv, ok := cache[imp.Path.Value]; ok {
		return rv
	}
	rv := getNameOrGuess(imp)
	cache[imp.Path.Value] = rv
	return rv
}

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
		log.Println("Cannot find package", path, "guessing it's name is", rv)
		return rv
	}
	return pkg.Name
}

/*
# to generate run
(cd $GOROOT/src/pkg; bash -c 'find * -type d' | grep -v testdata | \
python -c 'import sys;import glob
print "var Stdlib = map[string]string{"
for l in sys.stdin.read().split():
    if not glob.glob(l+"/*.go"): continue
    print """\t`"%s"`: "%s",""" % (l, l.split("/")[-1])
print "}"')
*/
// all stdlib precached
var DefaultImportCache = ImportCache(Stdlib)
var Stdlib = map[string]string{
	`"archive/tar"`:         "tar",
	`"archive/zip"`:         "zip",
	`"bufio"`:               "bufio",
	`"builtin"`:             "builtin",
	`"bytes"`:               "bytes",
	`"compress/bzip2"`:      "bzip2",
	`"compress/flate"`:      "flate",
	`"compress/gzip"`:       "gzip",
	`"compress/lzw"`:        "lzw",
	`"compress/zlib"`:       "zlib",
	`"container/heap"`:      "heap",
	`"container/list"`:      "list",
	`"container/ring"`:      "ring",
	`"crypto"`:              "crypto",
	`"crypto/aes"`:          "aes",
	`"crypto/cipher"`:       "cipher",
	`"crypto/des"`:          "des",
	`"crypto/dsa"`:          "dsa",
	`"crypto/ecdsa"`:        "ecdsa",
	`"crypto/elliptic"`:     "elliptic",
	`"crypto/hmac"`:         "hmac",
	`"crypto/md5"`:          "md5",
	`"crypto/rand"`:         "rand",
	`"crypto/rc4"`:          "rc4",
	`"crypto/rsa"`:          "rsa",
	`"crypto/sha1"`:         "sha1",
	`"crypto/sha256"`:       "sha256",
	`"crypto/sha512"`:       "sha512",
	`"crypto/subtle"`:       "subtle",
	`"crypto/tls"`:          "tls",
	`"crypto/x509"`:         "x509",
	`"crypto/x509/pkix"`:    "pkix",
	`"database/sql"`:        "sql",
	`"database/sql/driver"`: "driver",
	`"debug/dwarf"`:         "dwarf",
	`"debug/elf"`:           "elf",
	`"debug/gosym"`:         "gosym",
	`"debug/macho"`:         "macho",
	`"debug/pe"`:            "pe",
	`"encoding/ascii85"`:    "ascii85",
	`"encoding/asn1"`:       "asn1",
	`"encoding/base32"`:     "base32",
	`"encoding/base64"`:     "base64",
	`"encoding/binary"`:     "binary",
	`"encoding/csv"`:        "csv",
	`"encoding/gob"`:        "gob",
	`"encoding/hex"`:        "hex",
	`"encoding/json"`:       "json",
	`"encoding/pem"`:        "pem",
	`"encoding/xml"`:        "xml",
	`"errors"`:              "errors",
	`"expvar"`:              "expvar",
	`"flag"`:                "flag",
	`"fmt"`:                 "fmt",
	`"go/ast"`:              "ast",
	`"go/build"`:            "build",
	`"go/doc"`:              "doc",
	`"go/parser"`:           "parser",
	`"go/printer"`:          "printer",
	`"go/scanner"`:          "scanner",
	`"go/token"`:            "token",
	`"hash"`:                "hash",
	`"hash/adler32"`:        "adler32",
	`"hash/crc32"`:          "crc32",
	`"hash/crc64"`:          "crc64",
	`"hash/fnv"`:            "fnv",
	`"html"`:                "html",
	`"html/template"`:       "template",
	`"image"`:               "image",
	`"image/color"`:         "color",
	`"image/draw"`:          "draw",
	`"image/gif"`:           "gif",
	`"image/jpeg"`:          "jpeg",
	`"image/png"`:           "png",
	`"index/suffixarray"`:   "suffixarray",
	`"io"`:                  "io",
	`"io/ioutil"`:           "ioutil",
	`"log"`:                 "log",
	`"log/syslog"`:          "syslog",
	`"math"`:                "math",
	`"math/big"`:            "big",
	`"math/cmplx"`:          "cmplx",
	`"math/rand"`:           "rand",
	`"mime"`:                "mime",
	`"mime/multipart"`:      "multipart",
	`"net"`:                 "net",
	`"net/http"`:            "http",
	`"net/http/cgi"`:        "cgi",
	`"net/http/fcgi"`:       "fcgi",
	`"net/http/httptest"`:   "httptest",
	`"net/http/httputil"`:   "httputil",
	`"net/http/pprof"`:      "pprof",
	`"net/mail"`:            "mail",
	`"net/rpc"`:             "rpc",
	`"net/rpc/jsonrpc"`:     "jsonrpc",
	`"net/smtp"`:            "smtp",
	`"net/textproto"`:       "textproto",
	`"net/url"`:             "url",
	`"os"`:                  "os",
	`"os/exec"`:             "exec",
	`"os/signal"`:           "signal",
	`"os/user"`:             "user",
	`"path"`:                "path",
	`"path/filepath"`:       "filepath",
	`"reflect"`:             "reflect",
	`"regexp"`:              "regexp",
	`"regexp/syntax"`:       "syntax",
	`"runtime"`:             "runtime",
	`"runtime/cgo"`:         "cgo",
	`"runtime/debug"`:       "debug",
	`"runtime/pprof"`:       "pprof",
	`"sort"`:                "sort",
	`"strconv"`:             "strconv",
	`"strings"`:             "strings",
	`"sync"`:                "sync",
	`"sync/atomic"`:         "atomic",
	`"syscall"`:             "syscall",
	`"testing"`:             "testing",
	`"testing/iotest"`:      "iotest",
	`"testing/quick"`:       "quick",
	`"text/scanner"`:        "scanner",
	`"text/tabwriter"`:      "tabwriter",
	`"text/template"`:       "template",
	`"text/template/parse"`: "parse",
	`"time"`:                "time",
	`"unicode"`:             "unicode",
	`"unicode/utf16"`:       "utf16",
	`"unicode/utf8"`:        "utf8",
	`"unsafe"`:              "unsafe",
}
