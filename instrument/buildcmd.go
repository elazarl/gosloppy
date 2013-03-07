package instrument

import (
	"errors"
	"flag"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// GoCmd is a serialized command line instruction to run the Go tool
// For example
//     $ go run foo.go # equiv GoCmd{"go", "run", "foo.go", []string{}}
//     $ go test -test.run 'A.*' # equiv GoCmd{"go", "test", "", []string{"-test.run", "A.*"}}
type GoCmd struct {
	WorkDir    string
	Executable string
	Command    string
	BuildFlags Flags
	Packages   []string
}

type Flags map[string]string

func FromFlagSet(fs *flag.FlagSet) Flags {
	return make(Flags).FromFlagSet(fs)
}

func (flags Flags) FromFlagSet(fs *flag.FlagSet) Flags {
	fs.Visit(func(f *flag.Flag) {
		flags[f.Name] = f.Value.String()
	})
	return flags
}

func (flags Flags) Clone() Flags {
	clone := make(Flags)
	for k, v := range flags {
		clone[k] = v
	}
	return clone
}

func (flags Flags) String() string {
	b := make([]byte, 0, 100)
	for k, v := range flags {
		b = append(b, k+"="+v...)
	}
	return string(b)
}

func NewGoCmd(workdir string, args ...string) (*GoCmd, error) {
	return NewGoCmdWithFlags(flag.NewFlagSet("", flag.ContinueOnError), workdir, args...)
}

func NewGoCmdWithFlags(flags *flag.FlagSet, workdir string, args ...string) (*GoCmd, error) {
	if len(args) < 2 {
		return nil, errors.New("GoCmd must have at least two arguments (e.g. go build)")
	}
	if sort.SearchStrings([]string{"build", "test"}, args[1]) > -1 {
		flags.Int("p", runtime.NumCPU(), "number or parallel builds")
		for _, f := range []string{"x", "v", "n", "a", "work"} {
			flags.Bool(f, false, "")
		}
		for _, f := range []string{"compiler", "gccgoflags", "gcflags", "ldflags", "tags"} {
			flag.String(f, "", "")
		}
	}
	switch args[1] {
	case "build":
		flags.String("o", "", "output: output file")
		if err := flags.Parse(args[2:]); err != nil {
			return nil, err
		}
	case "test":
		for _, f := range []string{"i", "c"} {
			flags.Bool(f, false, "")
		}
	default:
		return nil, errors.New("Currently only build and test commands supported")
	}
	return &GoCmd{workdir, args[0], args[1], FromFlagSet(flags), flags.Args()}, nil
}

func (cmd *GoCmd) Args() []string {
	l := []string{cmd.Command}
	for k, v := range cmd.BuildFlags {
		l = append(l, "-"+k+"="+v)
	}
	l = append(l, cmd.Packages...)
	return l
}

func (cmd *GoCmd) String() string {
	return strings.Join(append([]string{cmd.Executable}, cmd.Args()...), " ")
}

// Retarget will return a new command line to compile the new target, but keep paths
// redirected to the original target.
func (cmd *GoCmd) Retarget(newdir string) (*GoCmd, error) {
	if len(cmd.Packages) > 1 {
		return nil, errors.New("No support for more than a single package")
	}
	var pkg *build.Package
	var err error
	if len(cmd.Packages) == 0 {
		pkg, err = build.ImportDir(cmd.WorkDir, 0)
	} else {
		pkg, err = build.Import(cmd.Packages[0], "", 0)
	}
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(newdir, cmd.WorkDir)
	if err != nil {
		return nil, err
	}
	defoutput := pkg.Name + ".a"
	if pkg.Name == "main" {
		if len(cmd.Packages) == 0 {
			// hopefully a legal package will contain at list one file...
			// the docs says that we should take the name of the first file, reality however
			// is different: http://code.google.com/p/go/issues/detail?id=5003
			d, err := filepath.Abs(pkg.Dir)
			if err != nil {
				return nil, err
			}
			defoutput = filepath.Base(d)
		} else {
			// not in the docs, but trying to run `go build pkg` gives a `pkg` executable
			defoutput = filepath.Base(cmd.Packages[0])
		}
	}
	buildflags := cmd.BuildFlags.Clone()
	switch cmd.Command {
	case "build":
		v := cmd.BuildFlags["o"]
		if v == "" {
			v = defoutput
		}
		buildflags["o"] = filepath.Join(rel, v)
	default:
		return nil, errors.New("No support for commands other than build test or run")
	}
	return &GoCmd{newdir, cmd.Executable, cmd.Command, buildflags, cmd.Packages}, nil
}

func (cmd *GoCmd) Runnable() *exec.Cmd {
	r := exec.Command("go", cmd.Args()...)
	r.Dir = cmd.WorkDir
	r.Stdin = os.Stdin
	r.Stdout = os.Stdout
	return r
}
