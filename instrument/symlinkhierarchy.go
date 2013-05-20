package instrument

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func symlinkGoroot(dst string) error {
	goroot := runtime.GOROOT()
	for _, name := range [...]string{"src", "pkg"} {
		dir := filepath.Join(goroot, name)
		if err := symlinkHierarchy(filepath.Join(dst, name), dir); err != nil {
			return err
		}
	}
	return nil
}

// adapted from: https://github.com/axw/gocov/blob/master/gocov/main.go#L142
func symlinkHierarchy(dst, src string) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if _, err = os.Stat(target); err == nil {
			return nil
		}

		// Walk directory symlinks. Check for target
		// existence above and os.MkdirAll below guards
		// against infinite recursion.
		mode := info.Mode()
		if mode&os.ModeSymlink == os.ModeSymlink {
			realpath, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if !filepath.IsAbs(realpath) {
				dir := filepath.Dir(path)
				realpath = filepath.Join(dir, realpath)
			}
			info, err := os.Stat(realpath)
			if err != nil {
				return err
			}
			if info.IsDir() {
				err = os.MkdirAll(target, 0700)
				if err != nil {
					return err
				}
				// Symlink contents, as the MkdirAll above
				// and the initial existence check will work
				// against each other.
				f, err := os.Open(realpath)
				if err != nil {
					return err
				}
				names, err := f.Readdirnames(-1)
				f.Close()
				if err != nil {
					return err
				}
				for _, name := range names {
					realpath := filepath.Join(realpath, name)
					target := filepath.Join(target, name)
					err = symlinkHierarchy(realpath, target)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}

		if mode.IsDir() {
			return os.MkdirAll(target, 0700)
		} else {
			err = os.Symlink(path, target)
			if err != nil {
				srcfile, err := os.Open(path)
				if err != nil {
					return err
				}
				defer srcfile.Close()
				dstfile, err := os.OpenFile(
					target, os.O_RDWR|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
				defer dstfile.Close()
				_, err = io.Copy(dstfile, srcfile)
				return err
			}
		}
		return nil
	}
	return filepath.Walk(src, fn)
}
