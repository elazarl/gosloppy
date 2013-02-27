package instrument

import (
	"fmt"
	"github.com/elazarl/gosloppy/patch"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func orFail(err error, t *testing.T) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			t.Fatal("Cannot get caller data")
		}
		t.Fatalf("%s:%d: %v", file, line, err)
	}
}

type Fs struct {
	Name     string
	Content  string
	Children []*Fs
}

func (fs *Fs) Remove(path string) error {
	return os.RemoveAll(filepath.Join(path, fs.Name))
}

func (fs *Fs) Build(path string) error {
	name := filepath.Join(path, fs.Name)
	if fs.Children == nil {
		if err := ioutil.WriteFile(name, []byte(fs.Content), 0644); err != nil {
			return err
		}
	} else {
		if err := os.Mkdir(name, 0755); err != nil {
			return err
		}
		for _, child := range fs.Children {
			if err := child.Build(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func dir(name string, children ...*Fs) *Fs {
	return &Fs{name, "", children}
}

func file(name, content string) *Fs {
	return &Fs{name, content, nil}
}

func assertFileIs(filename, content string, t *testing.T) {
	b, err := ioutil.ReadFile(filename)
	orFail(err, t)
	if string(b) != content {
		t.Error("Expected", content, "got", string(b))
	}
}

func TestDir(t *testing.T) {
	fs := dir(
		"test1",
		file("a.go", "package test1"), file("a_test.go", "package test1"),
	)
	orFail(fs.Build("."), t)
	defer func() { orFail(fs.Remove("."), t) }()
	dir, err := ImportDir("test1", "test1")
	orFail(err, t)
	if fmt.Sprint(dir.Files()) != "[test1/a.go test1/a_test.go]" {
		t.Fatal("Expected [a.go a_test.go] got", dir.Files())
	}
	orFail(os.Mkdir("temp", 0755), t)
	orFail(dir.Instrument("temp", func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	}), t)
	assertFileIs("temp/a.go", "koko", t)
	assertFileIs("temp/a_test.go", "koko", t)
	defer func() { orFail(os.RemoveAll("temp"), t) }()
}
