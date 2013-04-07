package instrument

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func expectEq(exp, act string, t *testing.T) {
	if exp != act {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			panic("cannot be run alone")
		}
		fmt.Fprintf(os.Stderr, "%s:%d: Expected %s Got %s", file, line, exp, act)
		t.Fail()
	}
}

func TestGoCmdRetarget(t *testing.T) {
	OrFail(dir("pkg", file("p.go", "package main;func main(){}")).Build("."), t)
	defer func() { OrFail(os.RemoveAll("pkg"), t) }()
	cmd, err := NewGoCmd("pkg", "go", "build", "-o", "koko")
	OrFail(err, t)
	cmd, err = cmd.Retarget("pkg/temp")
	OrFail(err, t)
	path, err := filepath.Abs(filepath.Join(cmd.WorkDir, "../koko"))
	OrFail(err, t)
	expectEq(fmt.Sprint("go build -o=", path), cmd.String(), t)
}

func TestGoCmdParsing(t *testing.T) {
	cmd, err := NewGoCmd(".", "go", "build", "-o", "koko", "bobo")
	OrFail(err, t)
	expectEq("[bobo]", fmt.Sprint(cmd.Params), t)
	expectEq("[]", fmt.Sprint(cmd.ExtraFlags), t)
	expectEq("build", fmt.Sprint(cmd.Command), t)
}

func TestGoCmdParsingTest(t *testing.T) {
	cmd, err := NewGoCmd(".", "go", "test", "bobo", "-run", "away")
	OrFail(err, t)
	expectEq("[bobo]", fmt.Sprint(cmd.Params), t)
	expectEq("[-run away]", fmt.Sprint(cmd.ExtraFlags), t)
	expectEq("test", fmt.Sprint(cmd.Command), t)
}

func TestGoCmdParsingTestNoPkg(t *testing.T) {
	cmd, err := NewGoCmd(".", "go", "test", "-run", "away")
	OrFail(err, t)
	expectEq("[]", fmt.Sprint(cmd.Params), t)
	expectEq("run=away", fmt.Sprint(cmd.BuildFlags), t)
	expectEq("test", fmt.Sprint(cmd.Command), t)
}
