package instrument

import (
	"errors"
	"flag"
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
	Env        map[string]string
	WorkDir    string
	Executable string
	Command    string
	BuildFlags Flags
	Params     []string
	ExtraFlags []string
}

// Flags represents the values of Go's "flag" package command line flags in a certain command line.
//     fs := flag.NewFlagSet("", flag.ContinueOnError)
//     fs.Bool("bool", false, "")
//     fs.Bool("booldefault", true, "")
//     fs.String("string", "bobo", "")
//     fs.Parse([]string{"-bool", "-string", "input", "arg1"})
//     fmt.Println(instrument.FromFlagSet(fs))
// Output:
//     string=input bool=true
type Flags map[string]string

// FromFlagSet serialize flags set in the flagset into Flag
func FromFlagSet(fs *flag.FlagSet) Flags {
	return make(Flags).FromFlagSet(fs)
}

// FromFlagSet adds flags set in flagset into flags. In case of conflict, flags in flagset
// will override flags already set.
func (flags Flags) FromFlagSet(fs *flag.FlagSet) Flags {
	fs.Visit(func(f *flag.Flag) {
		flags[f.Name] = f.Value.String()
	})
	return flags
}

// Clone returns a new Flags instance with the same values set.
func (flags Flags) Clone() Flags {
	clone := make(Flags)
	for k, v := range flags {
		clone[k] = v
	}
	return clone
}

// String writes the flags into a string parsable by the flag package
func (flags Flags) String() string {
	b := make([]byte, 0, 100)
	for k, v := range flags {
		b = append(b, k+"="+v...)
		b = append(b, ' ')
	}
	return string(b[:len(b)-1])
}

// NewGoCmd creates a GoCmd struct from command line arguments and a working diretory
func NewGoCmd(workdir string, args ...string) (*GoCmd, error) {
	return NewGoCmdWithFlags(flag.NewFlagSet("", flag.ContinueOnError), workdir, args...)
}

var TestFlags = []string{
	"bench",
	"benchtime",
	"cpu",
	"cpuprofile",
	"memprofile",
	"memprofilerate",
	"parallel",
	"run",
	"timeout",
}

// NewGoCmdWithFlags like NewGoCmd, but wl also parse flags configured i flagset
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
		for _, testflag := range TestFlags {
			flagset.String(testflag, "", "")
			flagset.String("test."+testflag, "", "")
		}
		flagset.Bool("short", false, "")
		flagset.Bool("test.short", false, "")
		flagset.Bool("test.v", false, "")
	default:
		return nil, errors.New("Currently only build run and test commands supported. Sorry.")
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
	return &GoCmd{make(map[string]string), workdir, args[0], args[1], FromFlagSet(flagset), params, extra}, nil
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

// String returns string representaton of the Go comand, it is not executable by shell, and does
// not necessarily escape arguments correctly. For debugging purpose only.
func (cmd *GoCmd) String() string {
	return strings.Join(append([]string{cmd.Executable}, cmd.Args()...), " ")
}

// OutputFileName returns the output file of the go build tool execution. For example,
//     NewGoCmd("/tmp/foo", "go", "build") // returns foo, by directory name
//     NewGoCmd(".", "go", "test", "-c", "foo/bar") // returns bar.test
// Note: libraries (non-main packages), have no well defined output. The return
// value of OutputFileName is undefined if cmd is "go build a_non-main_package".
func (cmd *GoCmd) OutputFileName() (name string, ismain bool, err error) {
	if len(cmd.Params) > 1 {
		return "", false, errors.New("No support for more than a single package:" + strings.Join(cmd.Params, " "))
	}
	// Output filename of tests depends on path: http://code.google.com/p/go/issues/detail?id=5230
	testsuffix := ""
	if cmd.Command == "test" {
		testsuffix += ".test"
	}
	var d string
	if len(cmd.Params) == 0 {
		d, err = filepath.Abs(cmd.WorkDir)
		if err != nil {
			return "", false, err
		}
		d = filepath.Base(d)
	} else {
		d = filepath.Base(cmd.Params[0])
	}
	return d + testsuffix, false, nil
}

// Retarget will return a new command line to compile the new target, but keep paths
// redirected to the original target.
func (cmd *GoCmd) Retarget(newdir string) (*GoCmd, error) {
	workdir, err := filepath.Abs(cmd.WorkDir)
	if err != nil {
		return nil, err
	}
	buildflags := cmd.BuildFlags.Clone()
	params := cmd.Params
	switch cmd.Command {
	case "run":
		params = []string{}
		for _, p := range cmd.Params {
			params = append(params, filepath.Join(newdir, filepath.Base(p)))
		}
	case "test":
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
	return &GoCmd{make(map[string]string), newdir, cmd.Executable, cmd.Command, buildflags, params, cmd.ExtraFlags}, nil
}

// Runnable returns an exec.Cmd that invoke the go tool, as specified in cmd
func (cmd *GoCmd) Runnable() *exec.Cmd {
	r := exec.Command(cmd.Executable, cmd.Args()...)
	r.Dir = cmd.WorkDir
	r.Stdin = os.Stdin
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr
	// environment inherits parent process environment, cancel environment variable with empty string
	if len(cmd.Env) > 0 {
		for _, env := range os.Environ() {
			kv := strings.SplitN(env, "=", 2)
			if _, ok := cmd.Env[kv[0]]; !ok {
				r.Env = append(r.Env, env)
			}
		}
		for k, v := range cmd.Env {
			r.Env = append(r.Env, k+"="+v)
		}
	}
	return r
}
