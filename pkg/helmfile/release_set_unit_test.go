package helmfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoTemplateFileExtension tests that the correct file extension is used
// when enable_go_template is set to true or false
func TestGoTemplateFileExtension(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name               string
		enableGoTemplate   bool
		expectedExtension  string
	}{
		{
			name:              "Go template disabled - should use .yaml",
			enableGoTemplate:  false,
			expectedExtension: ".yaml",
		},
		{
			name:              "Go template enabled - should use .yaml.gotmpl",
			enableGoTemplate:  true,
			expectedExtension: ".yaml.gotmpl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ReleaseSet with test configuration
			fs := &ReleaseSet{
				Content:           "test: content",
				WorkingDirectory:  tempDir,
				Kubeconfig:        "/tmp/kubeconfig",
				EnableGoTemplate:  tt.enableGoTemplate,
				Bin:               "helmfile",  // Set binary name
				HelmBin:           "helm",      // Set helm binary name
			}

			// Call NewCommandWithKubeconfig which sets the TmpHelmFilePath
			_, err := NewCommandWithKubeconfig(fs, "version")
			if err != nil {
				t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
			}

			// Check that the file extension is correct
			if !strings.HasSuffix(fs.TmpHelmFilePath, tt.expectedExtension) {
				t.Errorf("Expected file path to end with %q, but got %q",
					tt.expectedExtension, fs.TmpHelmFilePath)
			}

			// Verify the file was actually created
			fullPath := filepath.Join(tempDir, fs.TmpHelmFilePath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				t.Errorf("Expected file %q to be created, but it doesn't exist", fullPath)
			}

			// Clean up the created file
			os.Remove(fullPath)
		})
	}
}

// TestGoTemplateFileContent tests that the file content is written correctly
// regardless of the template setting
func TestGoTemplateFileContent(t *testing.T) {
	tempDir := t.TempDir()

	testContent := `repositories:
- name: stable
  url: https://charts.helm.sh/stable

releases:
- name: myapp
  chart: stable/nginx
`

	fs := &ReleaseSet{
		Content:          testContent,
		WorkingDirectory: tempDir,
		Kubeconfig:       "/tmp/kubeconfig",
		EnableGoTemplate: true,
		Bin:              "helmfile",
		HelmBin:          "helm",
	}

	_, err := NewCommandWithKubeconfig(fs, "version")
	if err != nil {
		t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
	}

	// Read the created file and verify content
	fullPath := filepath.Join(tempDir, fs.TmpHelmFilePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch.\nExpected:\n%s\n\nGot:\n%s", testContent, string(content))
	}

	// Clean up
	os.Remove(fullPath)
}
