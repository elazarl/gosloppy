package instrument

import (
	"strings"
	"testing"
)

var repoTestCases = []struct {
	importPath string
	expected   string
}{
	{"github.com/DarthVader/deathstar", "github.com/DarthVader/deathstar"},
	{"github.com/DarthVader/deathstar/canalThatExplodesEverythingIfBombed", "github.com/DarthVader/deathstar"},
	{"bitbucket.org/taruti/pbkdf2.go", "bitbucket.org/taruti/pbkdf2.go"},
	{"launchpad.net/goamz/aws", "launchpad.net/goamz/aws"},
	{"code.google.com/p/digest2/obs", "code.google.com/p/digest2"},
	{"example.org/repo.git/foo/bar", "example.org/repo.git"},
	{"koko/a/b", "error:"},
}

func TestVcsRecognition(t *testing.T) {
	for _, c := range repoTestCases {
		if repo, err := repoRootForImportPathStatic(c.importPath); err != nil {
			if !strings.HasPrefix(c.expected, "error:") {
				t.Errorf("On %s: Expected non error '%s', got error: %s", c.importPath, c.expected, err.Error())
			}
		} else {
			if c.expected != repo.root {
				t.Errorf("On %s: Expected '%s', got '%s'", c.importPath, c.expected, repo.root)
			}
		}
	}
}
