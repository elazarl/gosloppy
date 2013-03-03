package instrument

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ripped of $GOROOT/src/cmd/go/vcs.go

// repoRoot represents a version control system, a repo, and a root of
// where to put it on disk.
type repoRoot struct {
	//Removed
	//vcs *vcsCmd

	// repo is the repository URL, including scheme
	repo string

	// root is the import path corresponding to the root of the
	// repository
	root string
}

// A vcsPath describes how to convert an import path into a
// version control system and repository name.
type vcsPath struct {
	prefix string // prefix this description applies to
	re     string // pattern for import path
	repo   string // repository to use (expand with match of re)
	vcs    string // version control system to use (expand with match of re)
	// REMOVED, we do not need the actual VCS
	//check  func(match map[string]string) error // additional checks
	//ping   bool                                // ping for scheme to use to download repo

	regexp *regexp.Regexp // cached compiled form of re
}

func init() {
	// fill in cached regexps.
	// Doing this eagerly discovers invalid regexp syntax
	// without having to run a command that needs that regexp.
	for _, srv := range vcsPaths {
		srv.regexp = regexp.MustCompile(srv.re)
	}
}

// expand rewrites s to replace {k} with match[k] for each key k in match.
func expand(match map[string]string, s string) string {
	for k, v := range match {
		s = strings.Replace(s, "{"+k+"}", v, -1)
	}
	return s
}

// repoRootForImportPathStatic attempts to map importPath to a
// repoRoot using the commonly-used VCS hosting sites in vcsPaths
// (github.com/user/dir), or from a fully-qualified importPath already
// containing its VCS type (foo.com/repo.git/dir)
//
// If scheme is non-empty, that scheme is forced.
func repoRootForImportPathStatic(importPath /*, scheme*/ string) (*repoRoot, error) {
	if strings.Contains(importPath, "://") {
		return nil, fmt.Errorf("invalid import path %q", importPath)
	}
	for _, srv := range vcsPaths {
		if !strings.HasPrefix(importPath, srv.prefix) {
			continue
		}
		m := srv.regexp.FindStringSubmatch(importPath)
		if m == nil {
			if srv.prefix != "" {
				return nil, fmt.Errorf("invalid %s import path %q", srv.prefix, importPath)
			}
			continue
		}

		// Build map of named subexpression matches for expand.
		match := map[string]string{
			"prefix": srv.prefix,
			"import": importPath,
		}
		for i, name := range srv.regexp.SubexpNames() {
			if name != "" && match[name] == "" {
				match[name] = m[i]
			}
		}
		if srv.vcs != "" {
			match["vcs"] = expand(match, srv.vcs)
		}
		if srv.repo != "" {
			match["repo"] = expand(match, srv.repo)
		}
		// Elazar, removed
		//		if srv.check != nil {
		//			if err := srv.check(match); err != nil {
		//				return nil, err
		//			}
		//		}
		//		vcs := vcsByCmd(match["vcs"])
		//		if vcs == nil {
		//			return nil, fmt.Errorf("unknown version control system %q", match["vcs"])
		//		}
		//		if srv.ping {
		//			if scheme != "" {
		//				match["repo"] = scheme + "://" + match["repo"]
		//			} else {
		//				for _, scheme := range vcs.scheme {
		//					if vcs.ping(scheme, match["repo"]) == nil {
		//						match["repo"] = scheme + "://" + match["repo"]
		//						break
		//					}
		//				}
		//			}
		//		}
		rr := &repoRoot{
			// Elazar, removed
			//			vcs:  vcs,
			repo: match["repo"],
			root: match["root"],
		}
		return rr, nil
	}
	return nil, errUnknownSite
}

// vcsPaths lists the known vcs paths.
var vcsPaths = []*vcsPath{
	// Google Code - new syntax
	{
		prefix: "code.google.com/",
		re:     `^(?P<root>code\.google\.com/p/(?P<project>[a-z0-9\-]+)(\.(?P<subrepo>[a-z0-9\-]+))?)(/[A-Za-z0-9_.\-]+)*$`,
		repo:   "https://{root}",
		//check:  googleCodeVCS,
	},

	// Google Code - old syntax
	{
		re: `^(?P<project>[a-z0-9_\-.]+)\.googlecode\.com/(git|hg|svn)(?P<path>/.*)?$`,
		//check: oldGoogleCode,
	},

	// Github
	{
		prefix: "github.com/",
		re:     `^(?P<root>github\.com/[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)(/[A-Za-z0-9_.\-]+)*$`,
		vcs:    "git",
		repo:   "https://{root}",
		//check:  noVCSSuffix,
	},

	// Bitbucket
	{
		prefix: "bitbucket.org/",
		re:     `^(?P<root>bitbucket\.org/(?P<bitname>[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`,
		repo:   "https://{root}",
		//check:  bitbucketVCS,
	},

	// Launchpad
	{
		prefix: "launchpad.net/",
		re:     `^(?P<root>launchpad\.net/((?P<project>[A-Za-z0-9_.\-]+)(?P<series>/[A-Za-z0-9_.\-]+)?|~[A-Za-z0-9_.\-]+/(\+junk|[A-Za-z0-9_.\-]+)/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`,
		vcs:    "bzr",
		repo:   "https://{root}",
		//check:  launchpadVCS,
	},

	// General syntax for any server.
	{
		re: `^(?P<root>(?P<repo>([a-z0-9.\-]+\.)+[a-z0-9.\-]+(:[0-9]+)?/[A-Za-z0-9_.\-/]*?)\.(?P<vcs>bzr|git|hg|svn))(/[A-Za-z0-9_.\-]+)*$`,
		//ping: true,
	},
}

var errUnknownSite = errors.New("dynamic lookup required to find mapping")
