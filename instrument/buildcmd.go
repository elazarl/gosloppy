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
	Params     []string
	ExtraFlags []string
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

func NewGoCmdWithFlags(flagset *flag.FlagSet, workdir string, args ...string) (*GoCmd, error) {
	if len(args) < 2 {
		return nil, errors.New("GoCmd must have at least two arguments (e.g. go build)")
	}
	if sort.SearchStrings([]string{"build", "run", "test"}, args[1]) > -1 {
		flagset.Int("p", runtime.NumCPU(), "number or parallel builds")
		for _, f := range []string{"x", "v", "n", "a", "work"} {
			flagset.Bool(f, false, "")
		}
		for _, f := range []string{"compiler", "gccgoflags", "gcflags", "ldflags", "tags"} {
			flagset.String(f, "", "")
		}
	}
	switch args[1] {
	case "run":
	case "build":
		flagset.String("o", "", "output: output file")
	case "test":
		for _, f := range []string{"i", "c"} {
			flagset.Bool(f, false, "")
		}
	default:
		return nil, errors.New("Currently only build run and test commands supported")
	}
	if err := flagset.Parse(args[2:]); err != nil {
		return nil, err
	}
	var params, extra []string
	switch args[1] {
	case "build":
		params = flagset.Args()
	case "run":
		for i, param := range flagset.Args() {
			if !strings.HasSuffix(param, ".go") {
				extra = flagset.Args()[i:]
				break
			}
			params = append(params, param)
		}
	case "test":
		for i, param := range flagset.Args() {
			if strings.HasPrefix(param, "-") {
				extra = flagset.Args()[i:]
				break
			}
			params = append(params, param)
		}
	default:
		return nil, errors.New("Currently only build run and test commands supported")
	}
	return &GoCmd{workdir, args[0], args[1], FromFlagSet(flagset), params, extra}, nil
}

func (cmd *GoCmd) Args() []string {
	l := []string{cmd.Command}
	for k, v := range cmd.BuildFlags {
		l = append(l, "-"+k+"="+v)
	}
	l = append(l, cmd.Params...)
	l = append(l, cmd.ExtraFlags...)
	return l
}

func (cmd *GoCmd) String() string {
	return strings.Join(append([]string{cmd.Executable}, cmd.Args()...), " ")
}

func (cmd *GoCmd) OutputFileName() (name string, ismain bool, err error) {
	if len(cmd.Params) > 1 {
		return "", false, errors.New("No support for more than a single package")
	}
	// TODO(elazar): use previous build.Package, or make build.Package cache. no reason to duplicate code
	var pkg *build.Package
	if len(cmd.Params) == 0 {
		pkg, err = build.ImportDir(cmd.WorkDir, 0)
	} else {
		pkg, err = build.Import(cmd.Params[0], "", 0)
	}
	if err != nil {
		return "", false, err
	}
	if pkg.Name != "main" {
		return pkg.Name, true, nil
	}
	d, err := filepath.Abs(pkg.Dir)
	if err != nil {
		return "", false, err
	}
	return filepath.Base(d), false, nil
}

// Retarget will return a new command line to compile the new target, but keep paths
// redirected to the original target.
func (cmd *GoCmd) Retarget(newdir string) (*GoCmd, error) {
	workdir, err := filepath.Abs(cmd.WorkDir)
	if err != nil {
		return nil, err
	}
	buildflags := cmd.BuildFlags.Clone()
	switch cmd.Command {
	case "run", "test":
	case "build":
		v := cmd.BuildFlags["o"]
		if v == "" {
			name, ismain, err := cmd.OutputFileName()
			if ismain {
				return nil, errors.New("gosloppy won't build non-main package, just for testing packages or producing executables")
			}
			if err != nil {
				return nil, err
			}
			v = name
		}
		buildflags["o"] = filepath.Join(workdir, v)
	default:
		return nil, errors.New("No support for commands other than build test or run")
	}
	return &GoCmd{newdir, cmd.Executable, cmd.Command, buildflags, cmd.Params, cmd.ExtraFlags}, nil
}

func (cmd *GoCmd) Runnable() *exec.Cmd {
	r := exec.Command(cmd.Executable, cmd.Args()...)
	r.Dir = cmd.WorkDir
	r.Stdin = os.Stdin
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr
	return r
}
