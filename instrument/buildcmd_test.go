package instrument

import (
	"os"
	"testing"
)

func expectEq(exp, act string, t *testing.T) {
	if exp != act {
		t.Errorf("Expected %s Got %s", exp, act)
	}
}

func TestGoCmdRetarget(t *testing.T) {
	OrFail(dir("pkg", file("p.go", "package main;func main(){}")).Build("."), t)
	defer func() { OrFail(os.RemoveAll("pkg"), t) }()
	cmd, err := NewGoCmd("pkg", "go", "build", "-o", "koko")
	OrFail(err, t)
	cmd, err = cmd.Retarget("pkg/temp")
	OrFail(err, t)
	expectEq("go build -o=../koko", cmd.String(), t)
}
