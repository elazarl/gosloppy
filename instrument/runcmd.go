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

func InstrumentCmd(f func(*patch.PatchableFile) patch.Patches, args ...string) error {
	fl := flag.NewFlagSet("", flag.ContinueOnError)
	basedir := fl.String("basedir", "", "instrument all packages decendant f basedir")
	goroot := fl.Bool("goroot", false, "Should I instrument packages in $GOROOT/src/pkg? (can take time)")
	gocmd, err := NewGoCmdWithFlags(fl, ".", args...)
	if err != nil {
		return err
	}

	var pkg *Instrumentable
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
		_, _, err := newgocmd.OutputFileName()
		if err != nil {
			panic("Should never happen: Cannot find package name, not producing test executable")
		}
		oldname, _, err := gocmd.OutputFileName()
		if err != nil {
			panic("Should never happen: Cannot find package name, not producing test executable")
		}
		// output name is unclear: http://code.google.com/p/go/issues/detail?id=5230
		testoutput := filepath.Join(outdir, filepath.Base(outdir)+".test")
		if len(newgocmd.Params) > 0 {
			testoutput = filepath.Join(outdir, filepath.Base(newgocmd.Params[0])+".test")
		}
		if minusC {
			if err := os.Rename(testoutput, oldname+".test"); err != nil {
				return err
			}
		} else {
			defer os.Remove(testoutput)
			r := exec.Command(testoutput, newgocmd.ExtraFlags...)
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
