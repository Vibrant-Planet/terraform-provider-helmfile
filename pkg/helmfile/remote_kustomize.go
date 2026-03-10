package helmfile

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitKustomizeRef represents a parsed remote git reference to a kustomize directory.
type GitKustomizeRef struct {
	// Host is the git host (e.g., "github.com")
	Host string
	// Org is the repository organization/owner
	Org string
	// Repo is the repository name
	Repo string
	// Subpath is the path within the repository
	Subpath string
	// Ref is the git ref (branch, tag, or commit)
	Ref string
}

// CloneURL returns the HTTPS clone URL for the repository.
func (g *GitKustomizeRef) CloneURL() string {
	return fmt.Sprintf("https://%s/%s/%s.git", g.Host, g.Org, g.Repo)
}

// gitURLPattern matches patterns like:
//
//	github.com/org/repo/path/to/dir?ref=v1.0.0
//	gitlab.com/org/repo/subdir?ref=main
var gitURLPattern = regexp.MustCompile(`^(github\.com|gitlab\.com|bitbucket\.org)/([^/]+)/([^/?]+)(/[^?]+)?\?ref=(.+)$`)

// ParseGitKustomizeURL parses a chart reference that looks like a remote git
// kustomize URL. Returns the parsed reference and true if the URL matches the
// expected pattern, or nil and false otherwise.
func ParseGitKustomizeURL(chart string) (*GitKustomizeRef, bool) {
	matches := gitURLPattern.FindStringSubmatch(chart)
	if matches == nil {
		return nil, false
	}

	subpath := strings.TrimPrefix(matches[4], "/")

	return &GitKustomizeRef{
		Host:    matches[1],
		Org:     matches[2],
		Repo:    matches[3],
		Subpath: subpath,
		Ref:     matches[5],
	}, true
}

// ResolveRemoteKustomize clones a remote git repository and checks if the
// referenced subdirectory contains a kustomization.yaml file. If it does,
// it returns the local path to the cloned directory. If not, it returns
// an empty string (indicating the chart reference should be handled normally).
//
// Clones are keyed by repo+ref so that multiple subpaths within the same
// repository share a single clone, avoiding redundant network fetches.
//
// The caller is responsible for cleaning up the returned directory.
func ResolveRemoteKustomize(ref *GitKustomizeRef, baseDir string) (string, error) {
	// Clone directory is keyed by repo+ref only (not subpath), so multiple
	// subpaths within the same repo reuse a single clone.
	repoDirName := fmt.Sprintf("kustomize-%s-%s-%s",
		ref.Org, ref.Repo,
		sanitizePath(ref.Ref),
	)
	cloneDir := filepath.Join(baseDir, repoDirName)

	// Clone only if we haven't already
	if _, err := os.Stat(cloneDir); err != nil {
		cmd := exec.Command("git", "clone",
			"--depth", "1",
			"--branch", ref.Ref,
			"--single-branch",
			ref.CloneURL(),
			cloneDir,
		)
		cmd.Env = os.Environ()
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git clone %s: %w\n%s", ref.CloneURL(), err, string(output))
		}
	}

	// Resolve subpath within the clone
	targetDir := cloneDir
	if ref.Subpath != "" {
		targetDir = filepath.Join(cloneDir, ref.Subpath)
	}

	if !isKustomizeDir(targetDir) {
		return "", nil
	}

	return targetDir, nil
}

// isKustomizeDir checks if a directory contains a kustomization.yaml or
// kustomization.yml or Kustomization file.
func isKustomizeDir(dir string) bool {
	for _, name := range []string{"kustomization.yaml", "kustomization.yml", "Kustomization"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// sanitizePath replaces path separators and special characters with dashes
// for use in directory names.
func sanitizePath(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = url.PathEscape(s)
	// Keep it short and filesystem-safe
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// RewriteHelmfileContent scans helmfile YAML content for chart references
// that match remote git kustomize URLs. For each match, it clones the repo,
// verifies kustomization.yaml exists, and rewrites the chart reference to
// the local path.
//
// Returns the potentially modified content, a list of temp directories to
// clean up, and any error.
func RewriteHelmfileContent(content string, baseDir string) (string, []string, error) {
	var cleanupDirs []string

	// Ensure the base directory for clones exists
	kustomizeDir := filepath.Join(baseDir, ".kustomize-cache")
	if err := os.MkdirAll(kustomizeDir, 0755); err != nil {
		return content, nil, fmt.Errorf("creating kustomize cache dir: %w", err)
	}

	// Find chart references that look like git URLs
	// We look for patterns like:
	//   chart: github.com/org/repo/path?ref=tag
	// in YAML content
	chartPattern := regexp.MustCompile(`(?m)(chart:\s*)((?:github\.com|gitlab\.com|bitbucket\.org)/[^\s#]+\?ref=[^\s#]+)`)

	modified := chartPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := chartPattern.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		prefix := submatches[1] // "chart: " with whitespace
		chartRef := submatches[2]

		ref, ok := ParseGitKustomizeURL(chartRef)
		if !ok {
			return match
		}

		localPath, err := ResolveRemoteKustomize(ref, kustomizeDir)
		if err != nil {
			logf("Warning: failed to resolve remote kustomize %s: %v", chartRef, err)
			return match
		}

		if localPath == "" {
			// Not a kustomize directory, leave as-is
			return match
		}

		repoDir := filepath.Join(kustomizeDir, fmt.Sprintf("kustomize-%s-%s-%s",
			ref.Org, ref.Repo, sanitizePath(ref.Ref)))
		// Only add to cleanup once per repo clone
		found := false
		for _, d := range cleanupDirs {
			if d == repoDir {
				found = true
				break
			}
		}
		if !found {
			cleanupDirs = append(cleanupDirs, repoDir)
		}

		return prefix + localPath
	})

	return modified, cleanupDirs, nil
}
