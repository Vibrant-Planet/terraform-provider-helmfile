package helmfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

// TestStateValuesFileCreation tests that state values are correctly written
// to temporary files that will be passed to helmfile via --state-values-file
func TestStateValuesFileCreation(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		values         []interface{}
		expectedFiles  int
		validateContent func(t *testing.T, content string)
	}{
		{
			name: "Single value with namespace",
			values: []interface{}{
				`{"namespace": "test-namespace"}`,
			},
			expectedFiles: 1,
			validateContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "namespace") {
					t.Errorf("Expected content to contain 'namespace', got: %s", content)
				}
				if !strings.Contains(content, "test-namespace") {
					t.Errorf("Expected content to contain 'test-namespace', got: %s", content)
				}
			},
		},
		{
			name: "YAML format with namespace",
			values: []interface{}{
				"namespace: my-namespace\nregion: us-west-2\n",
			},
			expectedFiles: 1,
			validateContent: func(t *testing.T, content string) {
				if !strings.Contains(content, "namespace: my-namespace") {
					t.Errorf("Expected content to contain 'namespace: my-namespace', got: %s", content)
				}
			},
		},
		{
			name: "Multiple values files",
			values: []interface{}{
				`{"namespace": "test-ns"}`,
				`{"region": "us-east-1"}`,
			},
			expectedFiles: 2,
			validateContent: func(t *testing.T, content string) {
				// Just verify first file has namespace
				if !strings.Contains(content, "namespace") {
					t.Errorf("Expected content to contain 'namespace', got: %s", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &ReleaseSet{
				Content:          "test: content",
				WorkingDirectory: tempDir,
				Kubeconfig:       "/tmp/kubeconfig",
				Values:           tt.values,
				Bin:              "helmfile",
				HelmBin:          "helm",
			}

			_, err := NewCommandWithKubeconfig(fs, "version")
			if err != nil {
				t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
			}

			// Check that temp values files were created
			files, err := filepath.Glob(filepath.Join(tempDir, "temp.values-*.yaml"))
			if err != nil {
				t.Fatalf("Failed to glob temp files: %v", err)
			}

			if len(files) != tt.expectedFiles {
				t.Errorf("Expected %d temp values files, got %d", tt.expectedFiles, len(files))
			}

			// Validate content of first file
			if len(files) > 0 {
				content, err := os.ReadFile(files[0])
				if err != nil {
					t.Fatalf("Failed to read temp values file: %v", err)
				}
				tt.validateContent(t, string(content))
			}

			// Clean up temp files
			for _, file := range files {
				os.Remove(file)
			}
		})
	}
}

// TestStateValuesWithGoTemplate tests that Go template rendering works
// with StateValues.namespace
func TestStateValuesWithGoTemplate(t *testing.T) {
	tempDir := t.TempDir()

	helmfileContent := `repositories:
- name: stable
  url: https://charts.helm.sh/stable

releases:
- name: myapp
  namespace: {{ .StateValues.namespace }}
  chart: stable/nginx
  version: 1.0.0
`

	fs := &ReleaseSet{
		Content:          helmfileContent,
		WorkingDirectory: tempDir,
		Kubeconfig:       "/tmp/kubeconfig",
		EnableGoTemplate: true,
		Values: []interface{}{
			`{"namespace": "production"}`,
		},
		Bin:     "helmfile",
		HelmBin: "helm",
	}

	_, err := NewCommandWithKubeconfig(fs, "version")
	if err != nil {
		t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
	}

	// Verify the helmfile template was created with .yaml.gotmpl extension
	if !strings.HasSuffix(fs.TmpHelmFilePath, ".yaml.gotmpl") {
		t.Errorf("Expected file path to end with .yaml.gotmpl when enable_go_template is true, got: %s", fs.TmpHelmFilePath)
	}

	// Verify the helmfile content contains the template variable
	fullPath := filepath.Join(tempDir, fs.TmpHelmFilePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read helmfile template: %v", err)
	}

	if !strings.Contains(string(content), "{{ .StateValues.namespace }}") {
		t.Errorf("Expected helmfile content to contain '{{ .StateValues.namespace }}', got: %s", string(content))
	}

	// Verify state values file was created
	valueFiles, err := filepath.Glob(filepath.Join(tempDir, "temp.values-*.yaml"))
	if err != nil {
		t.Fatalf("Failed to glob temp values files: %v", err)
	}

	if len(valueFiles) != 1 {
		t.Errorf("Expected 1 temp values file, got %d", len(valueFiles))
	}

	if len(valueFiles) > 0 {
		valuesContent, err := os.ReadFile(valueFiles[0])
		if err != nil {
			t.Fatalf("Failed to read values file: %v", err)
		}

		if !strings.Contains(string(valuesContent), "production") {
			t.Errorf("Expected values file to contain 'production', got: %s", string(valuesContent))
		}
	}

	// Clean up
	os.Remove(fullPath)
	for _, file := range valueFiles {
		os.Remove(file)
	}
}

// TestAccHelmfileReleaseSet_stateValuesNamespace is an acceptance test that
// verifies StateValues.namespace is correctly rendered in helmfile templates
func TestAccHelmfileReleaseSet_stateValuesNamespace(t *testing.T) {
	resourceName := "helmfile_release_set.test_namespace"
	namespace := "test-ns-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckShellScriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHelmfileReleaseSetConfig_stateValuesNamespace(namespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "enable_go_template", "true"),
					resource.TestCheckResourceAttr(resourceName, "dry_run", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "template_output"),
					testCheckOutputContainsNamespace(resourceName, namespace),
				),
			},
		},
	})
}

// testAccHelmfileReleaseSetConfig_stateValuesNamespace creates a test configuration
// that uses StateValues.namespace in the helmfile template
func testAccHelmfileReleaseSetConfig_stateValuesNamespace(namespace string) string {
	return fmt.Sprintf(`
resource "helmfile_release_set" "test_namespace" {
  content = <<EOF
repositories:
- name: sp
  url: https://stefanprodan.github.io/podinfo

releases:
- name: test-release
  namespace: {{ .StateValues.namespace }}
  chart: sp/podinfo
  version: 4.0.6
  values:
  - replicaCount: 1
EOF

  enable_go_template = true
  dry_run = true

  helm_binary = "helm"
  kubeconfig = pathexpand("~/.kube/config")
  working_directory = path.module
  environment = "default"

  # Pass namespace via state values
  values = [
    yamlencode({
      namespace = "%s"
    })
  ]
}
`, namespace)
}

// testCheckOutputContainsNamespace is a custom check function that verifies
// the template_output contains the expected namespace
func testCheckOutputContainsNamespace(resourceName, expectedNamespace string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		templateOutput := rs.Primary.Attributes["template_output"]
		if templateOutput == "" {
			return fmt.Errorf("template_output is empty")
		}

		// Check if the rendered output contains the namespace
		// The output should show the namespace in the manifest
		if !strings.Contains(templateOutput, fmt.Sprintf("namespace: %s", expectedNamespace)) {
			return fmt.Errorf("template_output does not contain expected namespace '%s'. Output:\n%s",
				expectedNamespace, templateOutput)
		}

		return nil
	}
}

// TestAccHelmfileReleaseSet_multipleStateValues tests that multiple state values
// can be provided and all are accessible in the template
func TestAccHelmfileReleaseSet_multipleStateValues(t *testing.T) {
	resourceName := "helmfile_release_set.test_multi"
	namespace := "multi-" + acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckShellScriptDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHelmfileReleaseSetConfig_multipleStateValues(namespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "dry_run", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "template_output"),
					testCheckOutputContainsNamespace(resourceName, namespace),
					testCheckOutputContains(resourceName, "replicaCount: 3"),
				),
			},
		},
	})
}

func testAccHelmfileReleaseSetConfig_multipleStateValues(namespace string) string {
	return fmt.Sprintf(`
resource "helmfile_release_set" "test_multi" {
  content = <<EOF
repositories:
- name: sp
  url: https://stefanprodan.github.io/podinfo

releases:
- name: multi-test
  namespace: {{ .StateValues.namespace }}
  chart: sp/podinfo
  version: 4.0.6
  values:
  - replicaCount: {{ .StateValues.replicas }}
EOF

  enable_go_template = true
  dry_run = true

  helm_binary = "helm"
  kubeconfig = pathexpand("~/.kube/config")
  working_directory = path.module
  environment = "default"

  # Pass multiple state values
  values = [
    yamlencode({
      namespace = "%s"
    }),
    yamlencode({
      replicas = 3
    })
  ]
}
`, namespace)
}

func testCheckOutputContains(resourceName, expectedString string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		templateOutput := rs.Primary.Attributes["template_output"]
		if templateOutput == "" {
			return fmt.Errorf("template_output is empty")
		}

		if !strings.Contains(templateOutput, expectedString) {
			return fmt.Errorf("template_output does not contain expected string '%s'. Output:\n%s",
				expectedString, templateOutput)
		}

		return nil
	}
}
