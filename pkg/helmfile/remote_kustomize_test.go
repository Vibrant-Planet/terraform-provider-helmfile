package helmfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitKustomizeURL(t *testing.T) {
	tests := []struct {
		name     string
		chart    string
		wantOK   bool
		wantHost string
		wantOrg  string
		wantRepo string
		wantSub  string
		wantRef  string
	}{
		{
			name:     "GitHub URL with subpath and ref",
			chart:    "github.com/kubeflow/manifests/apps/katib/upstream/installs/katib-with-kubeflow?ref=v1.9.1",
			wantOK:   true,
			wantHost: "github.com",
			wantOrg:  "kubeflow",
			wantRepo: "manifests",
			wantSub:  "apps/katib/upstream/installs/katib-with-kubeflow",
			wantRef:  "v1.9.1",
		},
		{
			name:     "GitHub URL with root path",
			chart:    "github.com/org/repo?ref=main",
			wantOK:   true,
			wantHost: "github.com",
			wantOrg:  "org",
			wantRepo: "repo",
			wantSub:  "",
			wantRef:  "main",
		},
		{
			name:     "GitLab URL",
			chart:    "gitlab.com/myorg/myrepo/deploy/base?ref=v2.0.0",
			wantOK:   true,
			wantHost: "gitlab.com",
			wantOrg:  "myorg",
			wantRepo: "myrepo",
			wantSub:  "deploy/base",
			wantRef:  "v2.0.0",
		},
		{
			name:     "Bitbucket URL",
			chart:    "bitbucket.org/team/project/kustomize?ref=release-1.0",
			wantOK:   true,
			wantHost: "bitbucket.org",
			wantOrg:  "team",
			wantRepo: "project",
			wantSub:  "kustomize",
			wantRef:  "release-1.0",
		},
		{
			name:   "Standard helm chart reference",
			chart:  "stable/nginx",
			wantOK: false,
		},
		{
			name:   "OCI reference",
			chart:  "oci://ghcr.io/org/chart",
			wantOK: false,
		},
		{
			name:   "Local path",
			chart:  "./charts/myapp",
			wantOK: false,
		},
		{
			name:   "Missing ref parameter",
			chart:  "github.com/org/repo/path",
			wantOK: false,
		},
		{
			name:   "Unsupported host",
			chart:  "example.com/org/repo/path?ref=main",
			wantOK: false,
		},
		{
			name:   "Empty string",
			chart:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, ok := ParseGitKustomizeURL(tt.chart)
			if ok != tt.wantOK {
				t.Fatalf("ParseGitKustomizeURL(%q) ok = %v, want %v", tt.chart, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if ref.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", ref.Host, tt.wantHost)
			}
			if ref.Org != tt.wantOrg {
				t.Errorf("Org = %q, want %q", ref.Org, tt.wantOrg)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", ref.Repo, tt.wantRepo)
			}
			if ref.Subpath != tt.wantSub {
				t.Errorf("Subpath = %q, want %q", ref.Subpath, tt.wantSub)
			}
			if ref.Ref != tt.wantRef {
				t.Errorf("Ref = %q, want %q", ref.Ref, tt.wantRef)
			}
		})
	}
}

func TestGitKustomizeRef_CloneURL(t *testing.T) {
	ref := &GitKustomizeRef{
		Host: "github.com",
		Org:  "kubeflow",
		Repo: "manifests",
		Ref:  "v1.9.1",
	}
	want := "https://github.com/kubeflow/manifests.git"
	if got := ref.CloneURL(); got != want {
		t.Errorf("CloneURL() = %q, want %q", got, want)
	}
}

func TestIsKustomizeDir(t *testing.T) {
	t.Run("directory with kustomization.yaml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte("resources:\n- deploy.yaml\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if !isKustomizeDir(dir) {
			t.Error("expected isKustomizeDir to return true")
		}
	})

	t.Run("directory with kustomization.yml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "kustomization.yml"), []byte("resources:\n- deploy.yaml\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if !isKustomizeDir(dir) {
			t.Error("expected isKustomizeDir to return true")
		}
	})

	t.Run("directory with Kustomization", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "Kustomization"), []byte("resources:\n- deploy.yaml\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if !isKustomizeDir(dir) {
			t.Error("expected isKustomizeDir to return true")
		}
	})

	t.Run("directory without kustomization", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: test\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if isKustomizeDir(dir) {
			t.Error("expected isKustomizeDir to return false")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		if isKustomizeDir("/nonexistent/path") {
			t.Error("expected isKustomizeDir to return false for nonexistent dir")
		}
	})
}

func TestRewriteHelmfileContent(t *testing.T) {
	t.Run("no git URLs leaves content unchanged", func(t *testing.T) {
		content := `repositories:
- name: stable
  url: https://charts.helm.sh/stable

releases:
- name: myapp
  chart: stable/nginx
  version: 1.0.0
`
		dir := t.TempDir()
		result, dirs, err := RewriteHelmfileContent(content, dir)
		if err != nil {
			t.Fatal(err)
		}
		if result != content {
			t.Errorf("expected content to be unchanged, got:\n%s", result)
		}
		if len(dirs) != 0 {
			t.Errorf("expected no cleanup dirs, got %d", len(dirs))
		}
	})

	t.Run("rewrites git URL when kustomization.yaml exists", func(t *testing.T) {
		// Set up a fake "cloned" repo
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, ".kustomize-cache", "kustomize-testorg-testrepo-deploy-base-v1.0.0")
		targetDir := filepath.Join(cacheDir, "deploy", "base")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "kustomization.yaml"), []byte("resources:\n- deploy.yaml\n"), 0644); err != nil {
			t.Fatal(err)
		}

		content := `releases:
- name: myapp
  chart: github.com/testorg/testrepo/deploy/base?ref=v1.0.0
  namespace: default
`
		result, dirs, err := RewriteHelmfileContent(content, dir)
		if err != nil {
			t.Fatal(err)
		}

		// The chart reference should be rewritten to the local path
		if result == content {
			t.Error("expected content to be rewritten, but it was unchanged")
		}
		if len(dirs) != 1 {
			t.Errorf("expected 1 cleanup dir, got %d", len(dirs))
		}
		// Verify the local path is in the result
		if !filepath.IsAbs(targetDir) {
			t.Error("expected absolute path in rewritten content")
		}
	})

	t.Run("leaves non-kustomize git URL unchanged", func(t *testing.T) {
		// Set up a fake "cloned" repo WITHOUT kustomization.yaml
		dir := t.TempDir()
		cacheDir := filepath.Join(dir, ".kustomize-cache", "kustomize-testorg-helmchart-charts-myapp-v2.0.0")
		targetDir := filepath.Join(cacheDir, "charts", "myapp")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Write a Chart.yaml instead (helm chart, not kustomize)
		if err := os.WriteFile(filepath.Join(targetDir, "Chart.yaml"), []byte("name: myapp\n"), 0644); err != nil {
			t.Fatal(err)
		}

		content := `releases:
- name: myapp
  chart: github.com/testorg/helmchart/charts/myapp?ref=v2.0.0
  namespace: default
`
		result, _, err := RewriteHelmfileContent(content, dir)
		if err != nil {
			t.Fatal(err)
		}

		// Content should be unchanged since the directory isn't a kustomize dir
		if result != content {
			t.Errorf("expected content to be unchanged for non-kustomize dir, got:\n%s", result)
		}
	})

	t.Run("handles multiple chart references", func(t *testing.T) {
		dir := t.TempDir()

		// Set up two fake repos, one kustomize and one not
		kustomizeDir := filepath.Join(dir, ".kustomize-cache", "kustomize-org-kustomize-repo-base-v1.0.0", "base")
		if err := os.MkdirAll(kustomizeDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(kustomizeDir, "kustomization.yaml"), []byte("resources:\n"), 0644); err != nil {
			t.Fatal(err)
		}

		content := `releases:
- name: app1
  chart: github.com/org/kustomize-repo/base?ref=v1.0.0
- name: app2
  chart: stable/nginx
`
		result, _, err := RewriteHelmfileContent(content, dir)
		if err != nil {
			t.Fatal(err)
		}

		// First chart should be rewritten, second should remain
		if result == content {
			t.Error("expected content to be modified")
		}
		if !containsString(result, "stable/nginx") {
			t.Error("expected stable/nginx to remain unchanged")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestResolveRemoteKustomize_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test clones a real public repository to verify the full flow.
	// Using a well-known, stable public repo.
	ref := &GitKustomizeRef{
		Host:    "github.com",
		Org:     "kubernetes-sigs",
		Repo:    "kustomize",
		Subpath: "examples/helloWorld",
		Ref:     "kustomize/v5.6.0",
	}

	dir := t.TempDir()
	localPath, err := ResolveRemoteKustomize(ref, dir)
	if err != nil {
		t.Fatalf("ResolveRemoteKustomize failed: %v", err)
	}

	if localPath == "" {
		t.Fatal("expected non-empty local path for kustomize directory")
	}

	// Verify kustomization.yaml exists
	if _, err := os.Stat(filepath.Join(localPath, "kustomization.yaml")); err != nil {
		t.Errorf("expected kustomization.yaml to exist in %s: %v", localPath, err)
	}

	// Verify idempotency — calling again should reuse the clone
	localPath2, err := ResolveRemoteKustomize(ref, dir)
	if err != nil {
		t.Fatalf("second ResolveRemoteKustomize failed: %v", err)
	}
	if localPath != localPath2 {
		t.Errorf("expected same path on second call, got %s and %s", localPath, localPath2)
	}
}
