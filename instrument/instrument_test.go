package instrument

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/elazarl/gosloppy/patch"
)

func TestDir(t *testing.T) {
	fs := dir(
		"test1",
		file("a.go", "package test1"), file("a_test.go", "package test1"),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("test1"), t) }()
	pkg, err := ImportDir("test1", "test1")
	OrFail(err, t)
	if fmt.Sprint(pkg.Files()) != "[test1/a.go]" {
		t.Fatal("Expected [a.go] got", pkg.Files())
	}
	outdir, err := pkg.Instrument(true, func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	})

	defer func() { OrFail(os.RemoveAll(outdir), t) }()
	OrFail(err, t)
	dir(outdir,
		file("a.go", "koko"),
		file("a_test.go", "koko"),
	).AssertEqual(outdir, t)
}

func TestGopath(t *testing.T) {
	fs := dir(
		"gopath/src/mypkg",
		file("a.go", "package mypkg"), file("a_test.go", "package mypkg"),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("gopath"), t) }()
	gopath, err := filepath.Abs("gopath")
	OrFail(err, t)
	prevgopath := build.Default.GOPATH
	defer func() { build.Default.GOPATH = prevgopath }()
	build.Default.GOPATH = gopath
	pkg, err := Import("mypkg", "mypkg")
	OrFail(err, t)
	OrFail(os.Mkdir("temp", 0755), t)
	defer func() { OrFail(os.RemoveAll("temp"), t) }()
	err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	})
	OrFail(err, t)
	dir("temp",
		file("a.go", "koko"),
		file("a_test.go", "koko"),
	).AssertEqual("temp", t)
}

func TestGuessSubpackage(t *testing.T) {
	fs := dir(
		"test",
		dir("sub1", file("sub1.go", "package sub1")),
		dir("sub2", file("sub2.go", "package sub2")),
		dir("sub3", file("sub3.go", `package sub3;import "../sub1"`)),
		file("base.go", `package test1;import "./sub1"`), file("a_test.go", `package test1;import "./sub2"`),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("test"), t) }()
	func() {
		pkg, err := ImportDir("", "test/sub3")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/sub3/sub3.go]" {
			t.Fatal("Expected [test/sub3/sub3.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			dir("locals", dir("__", dir("sub1", file("sub1.go", "koko")))),
			file("sub3.go", "koko"),
		).AssertEqual("temp", t)
	}()
}

func TestDontTakeStdLibByDefault(t *testing.T) {
	fs := dir(
		"test",
		file("main.go", `package main;import "fmt";func main() {fmt.Println()}`),
	)
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("test"), t) }()
	pkg, err := ImportDir("", "test")
	OrFail(err, t)
	OrFail(os.Mkdir("temp", 0755), t)
	defer func() { OrFail(os.RemoveAll("temp"), t) }()
	err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	})
	OrFail(err, t)
	dir("temp",
		file("main.go", "koko"),
	).AssertEqual("temp", t)
}

func TestGuessStdlibPkg(t *testing.T) {
	pkg, err := Import("", "io/ioutil")
	OrFail(err, t)
	OrFail(os.Mkdir("temp", 0755), t)
	defer func() { OrFail(os.RemoveAll("temp"), t) }()
	err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
		return patch.Patches{patch.Replace(pf.File, "koko")}
	})
	OrFail(err, t)
	dir("temp",
		dir("gopath", dir("io", file("io.go", "koko"))),
		file("ioutil.go", "koko"),
	).AssertContains("temp", t)
}

func TestGuessSubpkgGopath(t *testing.T) {
	fs := dir(
		"gopath/src/mypkg",
		dir("sub1", file("sub1.go", "package sub1")),
		dir("sub2", file("sub2.go", "package sub2")),
		dir("sub3", dir("subsub3", file("subsub3.go", `package subsub3;import "mypkg/sub1"`))),
		file("base.go", `package test1;import "mypkg/sub1"`), file("a_test.go", `package test1;import "mypkg/sub2"`),
	)
	// TODO(elazar): find a way to use build.Context
	gopath, err := filepath.Abs("gopath")
	OrFail(err, t)
	prevgopath := build.Default.GOPATH
	defer func() { build.Default.GOPATH = prevgopath }()
	build.Default.GOPATH = gopath
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("gopath"), t) }()
	func() {
		pkg, err := Import("", "mypkg/sub3/subsub3")
		OrFail(err, t)
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			dir("gopath", dir("mypkg", dir("sub1", file("sub1.go", "koko")))),
			file("subsub3.go", "koko"),
		).AssertEqual("temp", t)
	}()
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
	defer func() { OrFail(os.RemoveAll("test"), t) }()
	func() {
		pkg, err := ImportDir(".", "test")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/base.go]" {
			t.Fatal("Expected [test/base.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return nil
		})
		OrFail(err, t)
		dir("temp",
			dir("locals",
				dir("sub1", file("sub1.go", "package sub1")),
				dir("sub2", file("sub2.go", "package sub2"))),
			file("base.go", `package test1;import "./locals/sub1"`),
			file("a_test.go", `package test1;import "./locals/sub2"`),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := ImportDir(".", "test/sub3")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/sub3/sub3.go]" {
			t.Fatal("Expected [test/sub3/sub3.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			dir("locals", dir("__", dir("sub1", file("sub1.go", "koko")))),
			file("sub3.go", "koko"),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := ImportDir("./sub3", "test/sub3")
		OrFail(err, t)
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			file("sub3.go", "koko"),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := ImportDir(".", "test/sub2")
		OrFail(err, t)
		if fmt.Sprint(pkg.Files()) != "[test/sub2/sub2.go]" {
			t.Fatal("Expected [test/sub2/sub2.go] got", pkg.Files())
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			file("sub2.go", "koko"),
		).AssertEqual("temp", t)
	}()
}

func TestGopathSubDir(t *testing.T) {
	fs := dir(
		"gopath/src/mypkg",
		dir("sub1", file("sub1.go", "package sub1")),
		dir("sub2", file("sub2.go", "package sub2")),
		dir("sub3", dir("subsub3", file("subsub3.go", `package subsub3;import "mypkg/sub1"`))),
		file("base.go", `package test1;import "mypkg/sub1"`), file("a_test.go", `package test1;import "mypkg/sub2"`),
	)
	// TODO(elazar): find a way to use build.Context
	gopath, err := filepath.Abs("gopath")
	OrFail(err, t)
	prevgopath := build.Default.GOPATH
	defer func() { build.Default.GOPATH = prevgopath }()
	build.Default.GOPATH = gopath
	OrFail(fs.Build("."), t)
	defer func() { OrFail(os.RemoveAll("gopath"), t) }()
	func() {
		pkg, err := Import("mypkg", "mypkg")
		OrFail(err, t)
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp", dir("gopath", dir("mypkg",
			dir("sub1", file("sub1.go", "koko")),
			dir("sub2", file("sub2.go", "koko")))),
			file("base.go", "koko"), file("a_test.go", "koko"),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := Import("mypkg", "mypkg/sub3/subsub3")
		OrFail(err, t)
		if len(pkg.Files()) != 1 || pkg.Files()[0] != filepath.Join(gopath, "src", "mypkg", "sub3", "subsub3", "subsub3.go") {
			t.Fatal("When import \"mypkg/sub3/subsub3\" Expected", filepath.Join(gopath, "src", "mypkg", "sub3", "subsub3", "subsub3.go"))
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return nil
		})
		OrFail(err, t)
		dir("temp", dir("gopath", dir("mypkg",
			dir("sub1", file("sub1.go", "package sub1")))),
			file("subsub3.go", `package subsub3;import "./gopath/mypkg/sub1"`),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := Import("mypkg/sub3", "mypkg/sub3/subsub3")
		OrFail(err, t)
		if len(pkg.Files()) != 1 || pkg.Files()[0] != filepath.Join(gopath, "src", "mypkg", "sub3", "subsub3", "subsub3.go") {
			t.Fatal(`When import "mypkg/sub3/subsub3" Expected`, filepath.Join(gopath, "src", "mypkg", "sub3", "subsub3", "subsub3.go"))
		}
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			file("subsub3.go", "koko"),
		).AssertEqual("temp", t)
	}()
	func() {
		pkg, err := Import("mypkg", "mypkg/sub2")
		OrFail(err, t)
		OrFail(os.Mkdir("temp", 0755), t)
		defer func() { OrFail(os.RemoveAll("temp"), t) }()
		err = pkg.InstrumentTo(true, "temp", func(pf *patch.PatchableFile) patch.Patches {
			return patch.Patches{patch.Replace(pf.File, "koko")}
		})
		OrFail(err, t)
		dir("temp",
			file("sub2.go", "koko"),
		).AssertEqual("temp", t)
	}()
}

func fatalCaller(t *testing.T, depth int, msgs ...interface{}) {
	_, file, line, ok := runtime.Caller(depth + 1) // +1 to go up fatalCaller's stack
	if !ok {
		t.Fatal("Cannot get caller data")
	}
	t.Fatalf("%s:%d: %v", file, line, fmt.Sprintln(msgs...))
}

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

func (fs *Fs) AssertContains(path string, t *testing.T) {
	info, err := os.Stat(path)
	OrFail(err, t)
	fs.recursiveCheck(false, filepath.Dir(path), info, 2, t)
}

func (fs *Fs) AssertEqual(path string, t *testing.T) {
	info, err := os.Stat(path)
	OrFail(err, t)
	fs.recursiveCheck(true, filepath.Dir(path), info, 2, t)
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
// if compFolder is not set, it would not compare directories' content
func (fs *Fs) recursiveCheck(compFolder bool, path string, info os.FileInfo, depth int, t *testing.T) {
	path = filepath.Join(path, info.Name())
	if fs.Name != info.Name() {
		fatalCaller(t, depth, path, "expected", fs.Name)
	}
	if fs.IsDir() != info.IsDir() {
		fatalCaller(t, depth, path, "isDir=", info.IsDir(), "expected", fs.IsDir())
	}
	if fs.IsDir() {
		children, err := ioutil.ReadDir(path)
		OrFail(err, t)
		childrenMap := make(map[string]os.FileInfo)
		for _, child := range children {
			childrenMap[child.Name()] = child
		}
		if compFolder && len(children) != len(fs.Children) {
			fatalCaller(t, depth, "expected", fs.List(), "got", fileinfos(children))
		}
		for _, fsChild := range fs.Children {
			if child, ok := childrenMap[fsChild.Name]; ok {
				fsChild.recursiveCheck(compFolder, path, child, depth+1, t)
			} else {
				fatalCaller(t, depth, "no", fsChild.Name, "in", fs.Name, "expected", fs.List(), "got", fileinfos(children))
			}
		}
	} else {
		content, err := ioutil.ReadFile(path)
		OrFail(err, t)
		if fs.Content != string(content) {
			fatalCaller(t, depth, path, "expected content", fs.Content, "got", string(content))
		}
	}
}

func (fs *Fs) Build(path string) error {
	name := filepath.Join(path, fs.Name)
	if !fs.IsDir() {
		if err := ioutil.WriteFile(name, []byte(fs.Content), 0644); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(name, 0755); err != nil {
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
