package helmfile

import (
	"os"
	"strings"
	"testing"
)

// TestEnvironmentVariablesPassedToCommand verifies that environment variables
// are correctly passed to the helmfile command
func TestEnvironmentVariablesPassedToCommand(t *testing.T) {
	// Set a test environment variable in the parent process
	testEnvKey := "TEST_PARENT_ENV_VAR"
	testEnvValue := "parent_value"
	os.Setenv(testEnvKey, testEnvValue)
	defer os.Unsetenv(testEnvKey)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a ReleaseSet with custom environment variables
	fs := &ReleaseSet{
		Content:          "test: content",
		WorkingDirectory: tempDir,
		Kubeconfig:       "/tmp/kubeconfig",
		Bin:              "helmfile",
		HelmBin:          "helm",
		EnvironmentVariables: map[string]interface{}{
			"CUSTOM_VAR_1": "value1",
			"CUSTOM_VAR_2": "value2",
			"AWS_PROFILE":  "test-profile",
		},
	}

	// Create the command
	cmd, err := NewCommandWithKubeconfig(fs, "version")
	if err != nil {
		t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
	}

	// Convert environment to a map for easier testing
	envMap := make(map[string]string)
	for _, env := range cmd.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Test 1: Verify parent environment variables are inherited
	if val, ok := envMap[testEnvKey]; !ok {
		t.Errorf("Parent environment variable %s not found in command environment", testEnvKey)
	} else if val != testEnvValue {
		t.Errorf("Parent environment variable %s has wrong value: got %q, want %q", testEnvKey, val, testEnvValue)
	}

	// Test 2: Verify PATH is inherited from parent
	if _, ok := envMap["PATH"]; !ok {
		t.Error("PATH environment variable not found in command environment")
	}

	// Test 3: Verify custom environment variables are present
	customVars := map[string]string{
		"CUSTOM_VAR_1": "value1",
		"CUSTOM_VAR_2": "value2",
		"AWS_PROFILE":  "test-profile",
	}

	for key, expectedValue := range customVars {
		if val, ok := envMap[key]; !ok {
			t.Errorf("Custom environment variable %s not found in command environment", key)
		} else if val != expectedValue {
			t.Errorf("Custom environment variable %s has wrong value: got %q, want %q", key, val, expectedValue)
		}
	}

	// Test 4: Verify KUBECONFIG is set
	if val, ok := envMap["KUBECONFIG"]; !ok {
		t.Error("KUBECONFIG environment variable not found in command environment")
	} else if !strings.Contains(val, "kubeconfig") {
		t.Errorf("KUBECONFIG has unexpected value: %s", val)
	}

	// Test 5: Verify that custom vars can override parent vars
	overrideKey := "OVERRIDE_TEST"
	overrideParentValue := "parent"
	overrideCustomValue := "custom"

	os.Setenv(overrideKey, overrideParentValue)
	defer os.Unsetenv(overrideKey)

	fs2 := &ReleaseSet{
		Content:          "test: content",
		WorkingDirectory: tempDir,
		Kubeconfig:       "/tmp/kubeconfig",
		Bin:              "helmfile",
		HelmBin:          "helm",
		EnvironmentVariables: map[string]interface{}{
			overrideKey: overrideCustomValue,
		},
	}

	cmd2, err := NewCommandWithKubeconfig(fs2, "version")
	if err != nil {
		t.Fatalf("NewCommandWithKubeconfig failed for override test: %v", err)
	}

	envMap2 := make(map[string]string)
	for _, env := range cmd2.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap2[parts[0]] = parts[1]
		}
	}

	if val, ok := envMap2[overrideKey]; !ok {
		t.Errorf("Override environment variable %s not found", overrideKey)
	} else if val != overrideCustomValue {
		t.Errorf("Custom environment variable should override parent: got %q, want %q", val, overrideCustomValue)
	}

	// Clean up created file
	os.Remove(tempDir + "/" + fs.TmpHelmFilePath)
	os.Remove(tempDir + "/" + fs2.TmpHelmFilePath)
}

// TestEnvironmentVariablesKubeconfigValidation verifies that KUBECONFIG
// cannot be set in both kubeconfig attribute and environment_variables
func TestEnvironmentVariablesKubeconfigValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Test that setting KUBECONFIG in both places is rejected
	fs := &ReleaseSet{
		Content:          "test: content",
		WorkingDirectory: tempDir,
		Kubeconfig:       "/path/to/kubeconfig",
		Bin:              "helmfile",
		HelmBin:          "helm",
		EnvironmentVariables: map[string]interface{}{
			"KUBECONFIG":  "should-cause-error",
			"AWS_PROFILE": "test-profile",
		},
	}

	_, err := NewCommandWithKubeconfig(fs, "version")
	if err == nil {
		t.Fatal("Expected error when KUBECONFIG is set in both places, but got none")
	}

	expectedError := "helmfile_release_set.environment_variables.KUBECONFIG cannot be set with helmfile_release_set.kubeconfig"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got: %v", expectedError, err)
	}
}

// TestKubeconfigInEnvironmentOnly verifies KUBECONFIG can be set via environment_variables alone
func TestKubeconfigInEnvironmentOnly(t *testing.T) {
	tempDir := t.TempDir()

	fs := &ReleaseSet{
		Content:          "test: content",
		WorkingDirectory: tempDir,
		Kubeconfig:       "",  // Empty kubeconfig attribute
		Bin:              "helmfile",
		HelmBin:          "helm",
		EnvironmentVariables: map[string]interface{}{
			"KUBECONFIG":  "/path/from/env/kubeconfig",
			"AWS_PROFILE": "test-profile",
		},
	}

	cmd, err := NewCommandWithKubeconfig(fs, "version")
	if err != nil {
		t.Fatalf("NewCommandWithKubeconfig failed: %v", err)
	}

	// Find KUBECONFIG in environment
	var kubeconfigValue string
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "KUBECONFIG=") {
			kubeconfigValue = strings.TrimPrefix(env, "KUBECONFIG=")
			break
		}
	}

	if kubeconfigValue == "" {
		t.Error("KUBECONFIG not found in environment")
	}

	if !strings.Contains(kubeconfigValue, "kubeconfig") {
		t.Errorf("KUBECONFIG should contain path from environment_variables: got %s", kubeconfigValue)
	}

	// Clean up
	os.Remove(tempDir + "/" + fs.TmpHelmFilePath)
}

// TestReadEnvironmentVariables tests the readEnvironmentVariables utility function
func TestReadEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]interface{}
		exclude  string
		expected map[string]string
	}{
		{
			name: "basic environment variables",
			envVars: map[string]interface{}{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			exclude: "",
			expected: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name: "exclude specific variable",
			envVars: map[string]interface{}{
				"VAR1":       "value1",
				"VAR2":       "value2",
				"KUBECONFIG": "should-be-excluded",
			},
			exclude: "KUBECONFIG",
			expected: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name:     "nil environment variables",
			envVars:  nil,
			exclude:  "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readEnvironmentVariables(tt.envVars, tt.exclude)

			// Convert result to map for easier comparison
			resultMap := make(map[string]string)
			for _, env := range result {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					resultMap[parts[0]] = parts[1]
				}
			}

			// Check all expected variables are present
			for key, expectedValue := range tt.expected {
				if val, ok := resultMap[key]; !ok {
					t.Errorf("Expected variable %s not found in result", key)
				} else if val != expectedValue {
					t.Errorf("Variable %s has wrong value: got %q, want %q", key, val, expectedValue)
				}
			}

			// Check that excluded variable is not present
			if tt.exclude != "" {
				if _, ok := resultMap[tt.exclude]; ok {
					t.Errorf("Excluded variable %s should not be in result", tt.exclude)
				}
			}

			// Check no unexpected variables
			if len(resultMap) != len(tt.expected) {
				t.Errorf("Result has wrong number of variables: got %d, want %d", len(resultMap), len(tt.expected))
			}
		})
	}
}
