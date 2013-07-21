package instrument

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/elazarl/gosloppy/patch"
)

// InstrumentCmd will run the go tool command specified in "args", but will make sure the
// package file's are instrumented with f before running the command.
// For example:
//     // tests the package foo/bar, replace all files' content with string "bobo". Would not compile.
//     InstrumentCmd(func(p *patch.PatchableFile) patch.Patches { return patch.Replace(p.All(), "bobo") },
//         "go", "test", "foo/bar")
//     // Another option to test foo/bar
//     os.Chdir(filepath.Join(os.Getenv("GOPATH"), "src", "foo", "bar"))
//     InstrumentCmd(func(p *patch.PatchableFile) patch.Patches { return patch.Replace(p.All(), "bobo") },
//         "goCommandNameIsIgnored", "test")
//     // You can even instrument pacakges in $GOROOT if you use the -goroot switch
//     InstrumentCmd(f, "go", "test", "-goroot", "net/url")
func InstrumentCmd(f func(*patch.PatchableFile) patch.Patches, args ...string) (err error) {
	var pkg *Instrumentable
	if len(args) > 1 && args[1] == "inline" {
		switch args := args[2:]; {
		case len(args) == 0:
			pkg, err = ImportDir("", ".")
		case len(args) > 1 || strings.HasSuffix(args[0], ".go"):
			pkg = ImportFiles("", args...)
		default:
			pkg, err = Import("", args[0])
		}
		return pkg.InstrumentInline(f)
	}

	fl := flag.NewFlagSet("", flag.ContinueOnError)
	basedir := fl.String("basedir", "", "instrument all packages decendant f basedir")
	goroot := fl.Bool("goroot", false, "Should I instrument packages in $GOROOT/src/pkg? (can take time)")
	gocmd, err := NewGoCmdWithFlags(fl, ".", args...)
	if err != nil {
		return err
	}

	if gocmd.Command == "run" {
		pkg = ImportFiles(*basedir, gocmd.Params...)
	} else if len(gocmd.Params) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
			path = filepath.Join(path, "src")
			if strings.Contains(wd, path) {
				rel, err := filepath.Rel(path, wd)
				if err != nil {
					return err
				}
				gocmd.Params = []string{rel}
				break
			}
		}
		path := filepath.Join(os.Getenv("GOROOT"), "src", "pkg")
		if strings.Contains(wd, path) {
			rel, err := filepath.Rel(path, wd)
			if err != nil {
				return err
			}
			gocmd.Params = []string{rel}
		}
		if len(gocmd.Params) == 0 {
			pkg, err = ImportDir(*basedir, ".")
		}
	}
	if pkg == nil {
		if pkg, err = Import(*basedir, gocmd.Params[0]); err != nil {
			return err
		}
	}
	pkg.InstrumentGoroot = *goroot
	outdir, hasGoroot, err := pkg.Instrument(gocmd.Command == "test", f)
	if gocmd.BuildFlags["work"] == "true" {
		log.Println("Instrumenting to", outdir)
	}
	defer func() {
		if gocmd.BuildFlags["work"] != "true" {
			if err := os.RemoveAll(outdir); err != nil {
				log.Println("Cannot remove temporary dir", outdir, err)
			}
		}
	}()
	if err != nil {
		return err
	}
	newgocmd, err := gocmd.Retarget(outdir)
	if err != nil {
		return err
	}
	if hasGoroot && *goroot {
		newgocmd.Env["GOROOT"] = filepath.Join(outdir, "goroot")
	}
	newgocmd.Executable = "go"
	// TODO(elazarl): Support build gofile.go gofile2.go
	if newgocmd.Command != "run" && !pkg.pkg.Goroot { // goroot package must be in its place
		newgocmd.Params = nil
	}
	// TODO(elazarl): hackish, find better way
	delete(newgocmd.BuildFlags, "basedir")
	delete(newgocmd.BuildFlags, "goroot")
	minusC := newgocmd.BuildFlags["c"] != ""
	if newgocmd.Command == "test" {
		newgocmd.BuildFlags["c"] = "true"
	}
	if fl.Lookup("x").Value.String() == "true" {
		log.Println("In:", newgocmd.WorkDir)
		log.Println("Executing:", newgocmd)
	}
	if err := newgocmd.Runnable().Run(); err != nil {
		return err
	}
	if newgocmd.Command == "test" {
		finalname, _, err := gocmd.OutputFileName()
		workdir, err := filepath.Abs(gocmd.WorkDir)
		if err != nil {
			return err
		}
		finalname = filepath.Join(workdir, finalname)
		if err != nil {
			panic("Should never happen: Cannot find package name, not producing test executable")
		}
		currname, _, err := newgocmd.OutputFileName()
		currname = filepath.Join(outdir, currname)
		if err != nil {
			panic("Should never happen: Cannot find package name, not producing test executable")
		}
		if fl.Lookup("x").Value.String() == "true" {
			log.Println("mv", currname, finalname)
		}
		if err := os.Rename(currname, finalname); err != nil {
			return err
		}
		if !minusC {
			defer os.Remove(finalname)
			for _, flag := range TestFlags {
				v, ok := newgocmd.BuildFlags[flag]
				if !ok {
					v, ok = newgocmd.BuildFlags["test."+flag]
				}
				if ok {
					newgocmd.ExtraFlags = append(newgocmd.ExtraFlags, "-test."+flag+"="+v)
				}
			}
			r := exec.Command(finalname, newgocmd.ExtraFlags...)
			r.Dir = gocmd.WorkDir
			r.Stdin = os.Stdin
			r.Stdout = os.Stdout
			r.Stderr = os.Stderr
			if err := r.Run(); err != nil {
				return err
			}
		}
	}
	return nil
}
