package instrument

import (
	"fmt"
	"github.com/elazarl/gosloppy/patch"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func OrFail(err error, t *testing.T) {
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

type FsList []*Fs

func (l FsList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l FsList) Less(i, j int) bool {
	return l[i].Name < l[j].Name
}

func (l FsList) Len() int {
	return len(l)
}

func (fs *Fs) IsDir() bool {
	return fs.Children != nil
}

func (fs *Fs) List() (children []*Fs) {
	children = append(children, fs.Children...)
	sort.Sort(FsList(children))
	return
}

func (fs *Fs) AssertEqual(path string, t *testing.T) {
	info, err := os.Stat(path)
	OrFail(err, t)
	fs.recursiveEqual(filepath.Dir(path), info, t)
}

func (fs *Fs) String() string {
	name := fs.Name
	if fs.IsDir() {
		name += "(d)"
	}
	return name
}

func fileinfos(infos []os.FileInfo) string {
	b := make([]byte, 0, 100)
	for _, info := range infos {
		b = append(b, " "+info.Name()...)
		if info.IsDir() {
			b = append(b, "(d)"...)
		}
	}
	return string(b)
}

// Compare returns whether a certain *Fs node is equal to an existing file tree
func (fs *Fs) recursiveEqual(path string, info os.FileInfo, t *testing.T) {
	path = filepath.Join(path, info.Name())
	if fs.Name != info.Name() {
		t.Fatal(path, "expected", fs.Name)
	}
	if fs.IsDir() != info.IsDir() {
		t.Fatal(path, "isDir=", info.IsDir(), "expected", fs.IsDir())
	}
	if fs.IsDir() {
		children, err := ioutil.ReadDir(path)
		OrFail(err, t)
		if len(children) != len(fs.Children) {
			t.Fatal("expected", fs.List(), "got", fileinfos(children))
		}
		for i, child := range fs.List() {
			if child.Name != children[i].Name() {
				t.Fatal("expected", fs.List(), "got", fileinfos(children))
			}
			child.recursiveEqual(path, children[i], t)
		}
	} else {
		content, err := ioutil.ReadFile(path)
		OrFail(err, t)
		if fs.Content != string(content) {
			t.Fatal(path, "expected content", fs.Content, "got", string(content))
		}
	}
}

func (fs *Fs) Remove(path string) error {
	return os.RemoveAll(filepath.Join(path, fs.Name))
}

func (fs *Fs) Build(path string) error {
	name := filepath.Join(path, fs.Name)
	if !fs.IsDir() {
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
	OrFail(err, t)
	if string(b) != content {
		t.Error("File", filename, "Expected", content, "Actually", string(b))
	}
}

func TestDir(t *testing.T) {
	fs := dir(
		"test1",
		file("a.go", "package test1"), file("a_test.go", "package test1"),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(fs.Remove("."), t) }()
	dir, err := ImportDir("test1", "test1")
	OrFail(err, t)
	if fmt.Sprint(dir.Files()) != "[test1/a.go test1/a_test.go]" {
		t.Fatal("Expected [a.go a_test.go] got", dir.Files())
	}
	OrFail(os.Mkdir("temp", 0755), t)
	defer func() { OrFail(os.RemoveAll("temp"), t) }()
	OrFail(dir.Instrument("temp", func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	}), t)
	assertFileIs("temp/a.go", "koko", t)
	assertFileIs("temp/a_test.go", "koko", t)
}

func TestSubDir(t *testing.T) {
	fs := dir(
		"test",
		dir("sub1", file("sub1.go", "package sub1")),
		dir("sub2", file("sub2.go", "package sub2")),
		dir("sub3", file("sub3.go", `package sub3;import "../sub1"`)),
		file("base.go", `package test1;import "./sub1"`), file("a_test.go", `package test1;import "./sub2"`),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(fs.Remove("."), t) }()
	func() {
		pkg, err := ImportDir("test", "test")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/base.go test/a_test.go]" {
			t.Fatal("Expected [test/base.go test/a_test.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		OrFail(pkg.Instrument("temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		}), t)
		dir("temp",
			dir("sub1", file("sub1.go", "koko")),
			dir("sub2", file("sub2.go", "koko")),
			file("base.go", "koko"), file("a_test.go", "koko"),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := ImportDir("test", "test/sub3")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/sub3/sub3.go]" {
			t.Fatal("Expected [test/sub3/sub3.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		OrFail(pkg.Instrument("temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		}), t)
		dir("temp",
			dir("sub1", file("sub1.go", "koko")),
			dir("sub3", file("sub3.go", "koko")),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := ImportDir("test", "test/sub2")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/sub2/sub2.go]" {
			t.Fatal("Expected [test/sub2/sub2.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		OrFail(pkg.Instrument("temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		}), t)
		dir("temp",
			dir("sub2", file("sub2.go", "koko")),
		).AssertEqual("temp", t)
	}()
}
