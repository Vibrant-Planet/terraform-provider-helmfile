package helmfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

// TestGenerateKubeconfigYAML tests the kubeconfig YAML generation
func TestGenerateKubeconfigYAML(t *testing.T) {
	tests := []struct {
		name           string
		config         *EKSClusterConfig
		expectedCluster string
		expectedServer  string
		expectedCA      string
		expectProfile   bool
	}{
		{
			name: "Basic kubeconfig with profile",
			config: &EKSClusterConfig{
				ClusterName: "test-cluster",
				Region:      "us-west-2",
				Endpoint:    "https://ABC123.gr7.us-west-2.eks.amazonaws.com",
				CA:          "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
				AWSProfile:  "my-profile",
			},
			expectedCluster: "test-cluster",
			expectedServer:  "https://ABC123.gr7.us-west-2.eks.amazonaws.com",
			expectedCA:      "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
			expectProfile:   true,
		},
		{
			name: "Kubeconfig without profile",
			config: &EKSClusterConfig{
				ClusterName: "prod-cluster",
				Region:      "us-east-1",
				Endpoint:    "https://XYZ789.gr7.us-east-1.eks.amazonaws.com",
				CA:          "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
				AWSProfile:  "",
			},
			expectedCluster: "prod-cluster",
			expectedServer:  "https://XYZ789.gr7.us-east-1.eks.amazonaws.com",
			expectedCA:      "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
			expectProfile:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate kubeconfig YAML
			yamlStr, err := generateKubeconfigYAML(tt.config)
			if err != nil {
				t.Fatalf("generateKubeconfigYAML() error = %v", err)
			}

			// Parse the YAML
			var kubeconfig KubeconfigData
			if err := yaml.Unmarshal([]byte(yamlStr), &kubeconfig); err != nil {
				t.Fatalf("Failed to parse generated YAML: %v", err)
			}

			// Verify basic structure
			if kubeconfig.APIVersion != "v1" {
				t.Errorf("Expected APIVersion v1, got %s", kubeconfig.APIVersion)
			}

			if kubeconfig.Kind != "Config" {
				t.Errorf("Expected Kind Config, got %s", kubeconfig.Kind)
			}

			if kubeconfig.CurrentContext != tt.expectedCluster {
				t.Errorf("Expected CurrentContext %s, got %s", tt.expectedCluster, kubeconfig.CurrentContext)
			}

			// Verify cluster configuration
			if len(kubeconfig.Clusters) != 1 {
				t.Fatalf("Expected 1 cluster, got %d", len(kubeconfig.Clusters))
			}

			cluster := kubeconfig.Clusters[0]
			if cluster.Name != tt.expectedCluster {
				t.Errorf("Expected cluster name %s, got %s", tt.expectedCluster, cluster.Name)
			}

			if cluster.Cluster.Server != tt.expectedServer {
				t.Errorf("Expected server %s, got %s", tt.expectedServer, cluster.Cluster.Server)
			}

			if cluster.Cluster.CertificateAuthorityData != tt.expectedCA {
				t.Errorf("Expected CA %s, got %s", tt.expectedCA, cluster.Cluster.CertificateAuthorityData)
			}

			// Verify context configuration
			if len(kubeconfig.Contexts) != 1 {
				t.Fatalf("Expected 1 context, got %d", len(kubeconfig.Contexts))
			}

			context := kubeconfig.Contexts[0]
			if context.Name != tt.expectedCluster {
				t.Errorf("Expected context name %s, got %s", tt.expectedCluster, context.Name)
			}

			if context.Context.Cluster != tt.expectedCluster {
				t.Errorf("Expected context cluster %s, got %s", tt.expectedCluster, context.Context.Cluster)
			}

			if context.Context.User != tt.expectedCluster {
				t.Errorf("Expected context user %s, got %s", tt.expectedCluster, context.Context.User)
			}

			// Verify user configuration
			if len(kubeconfig.Users) != 1 {
				t.Fatalf("Expected 1 user, got %d", len(kubeconfig.Users))
			}

			user := kubeconfig.Users[0]
			if user.Name != tt.expectedCluster {
				t.Errorf("Expected user name %s, got %s", tt.expectedCluster, user.Name)
			}

			// Verify exec config
			exec := user.User.Exec
			if exec.APIVersion != "client.authentication.k8s.io/v1beta1" {
				t.Errorf("Expected exec APIVersion client.authentication.k8s.io/v1beta1, got %s", exec.APIVersion)
			}

			if exec.Command != "aws" {
				t.Errorf("Expected exec command aws, got %s", exec.Command)
			}

			// Verify exec args
			expectedArgs := []string{"eks", "get-token", "--cluster-name", tt.config.ClusterName}
			if tt.config.Region != "" {
				expectedArgs = append(expectedArgs, "--region", tt.config.Region)
			}

			if len(exec.Args) != len(expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(expectedArgs), len(exec.Args))
			}

			for i, arg := range expectedArgs {
				if i >= len(exec.Args) || exec.Args[i] != arg {
					t.Errorf("Expected arg[%d] = %s, got %s", i, arg, exec.Args[i])
				}
			}

			// Verify AWS_PROFILE environment variable
			if tt.expectProfile {
				foundProfile := false
				for _, env := range exec.Env {
					if env.Name == "AWS_PROFILE" {
						foundProfile = true
						if env.Value != tt.config.AWSProfile {
							t.Errorf("Expected AWS_PROFILE=%s, got %s", tt.config.AWSProfile, env.Value)
						}
					}
				}
				if !foundProfile {
					t.Error("Expected AWS_PROFILE in exec env, but not found")
				}
			} else {
				for _, env := range exec.Env {
					if env.Name == "AWS_PROFILE" {
						t.Error("Did not expect AWS_PROFILE in exec env when profile is empty")
					}
				}
			}
		})
	}
}

// TestWriteTemporaryKubeconfig tests the temporary kubeconfig file creation
func TestWriteTemporaryKubeconfig(t *testing.T) {
	tests := []struct {
		name          string
		kubeconfigYAML string
		workingDir    string
		clusterName   string
		expectError   bool
	}{
		{
			name:          "Valid kubeconfig in temp dir",
			kubeconfigYAML: "apiVersion: v1\nkind: Config",
			workingDir:    "",
			clusterName:   "test-cluster",
			expectError:   false,
		},
		{
			name:          "Valid kubeconfig in custom dir",
			kubeconfigYAML: "apiVersion: v1\nkind: Config",
			workingDir:    t.TempDir(),
			clusterName:   "my-cluster",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write temporary kubeconfig
			filePath, err := writeTemporaryKubeconfig(tt.kubeconfigYAML, tt.workingDir, tt.clusterName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("writeTemporaryKubeconfig() error = %v", err)
			}

			// Verify file was created
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file %s to exist, but it doesn't", filePath)
			}

			// Verify filename contains cluster name
			filename := filepath.Base(filePath)
			if !strings.Contains(filename, tt.clusterName) {
				t.Errorf("Expected filename to contain %s, got %s", tt.clusterName, filename)
			}

			// Verify filename starts with expected prefix
			if !strings.HasPrefix(filename, ".terraform-helmfile-kubeconfig-") {
				t.Errorf("Expected filename to start with .terraform-helmfile-kubeconfig-, got %s", filename)
			}

			// Verify file content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if string(content) != tt.kubeconfigYAML {
				t.Errorf("Expected content %s, got %s", tt.kubeconfigYAML, string(content))
			}

			// Verify file permissions (should be 0600)
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			perm := info.Mode().Perm()
			if perm != 0600 {
				t.Errorf("Expected file permissions 0600, got %o", perm)
			}

			// Clean up
			os.Remove(filePath)
		})
	}
}

// TestCleanupKubeconfig tests the kubeconfig cleanup function
func TestCleanupKubeconfig(t *testing.T) {
	tests := []struct {
		name        string
		createFile  bool
		path        string
		expectError bool
	}{
		{
			name:        "Cleanup existing file",
			createFile:  true,
			path:        "",
			expectError: false,
		},
		{
			name:        "Cleanup non-existent file",
			createFile:  false,
			path:        "/tmp/non-existent-file-12345",
			expectError: false,
		},
		{
			name:        "Cleanup empty path",
			createFile:  false,
			path:        "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string

			if tt.createFile {
				// Create a temporary file
				tmpFile, err := os.CreateTemp("", "kubeconfig-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				filePath = tmpFile.Name()
				tmpFile.Close()

				// Verify file exists
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Fatalf("Temp file should exist but doesn't")
				}
			} else {
				filePath = tt.path
			}

			// Cleanup kubeconfig
			err := cleanupKubeconfig(filePath)

			if tt.expectError && err == nil {
				t.Error("Expected error, but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("cleanupKubeconfig() error = %v", err)
			}

			// If file was created, verify it was deleted
			if tt.createFile {
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("Expected file to be deleted, but it still exists")
				}
			}
		})
	}
}

// TestGetEKSRegion tests the region selection logic
func TestGetEKSRegion(t *testing.T) {
	tests := []struct {
		name              string
		eksClusterRegion  string
		awsRegion         string
		expectedRegion    string
	}{
		{
			name:             "EKS cluster region takes precedence",
			eksClusterRegion: "us-west-2",
			awsRegion:        "us-east-1",
			expectedRegion:   "us-west-2",
		},
		{
			name:             "Falls back to AWS region",
			eksClusterRegion: "",
			awsRegion:        "us-east-1",
			expectedRegion:   "us-east-1",
		},
		{
			name:             "EKS region when both set",
			eksClusterRegion: "eu-west-1",
			awsRegion:        "ap-southeast-1",
			expectedRegion:   "eu-west-1",
		},
		{
			name:             "Empty when both empty",
			eksClusterRegion: "",
			awsRegion:        "",
			expectedRegion:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock ResourceRead
			mockData := &mockResourceRead{
				data: map[string]interface{}{
					KeyEKSClusterRegion: tt.eksClusterRegion,
					KeyAWSRegion:        tt.awsRegion,
				},
			}

			// Get the region
			region := getEKSRegion(mockData)

			if region != tt.expectedRegion {
				t.Errorf("Expected region %s, got %s", tt.expectedRegion, region)
			}
		})
	}
}

// mockResourceRead is a simple mock for ResourceRead interface
type mockResourceRead struct {
	data map[string]interface{}
}

func (m *mockResourceRead) Get(key string) interface{} {
	if val, ok := m.data[key]; ok {
		return val
	}
	return ""
}

func (m *mockResourceRead) GetOk(key string) (interface{}, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *mockResourceRead) Id() string {
	return "mock-id"
}

// TestValidateEKSConfiguration tests the EKS configuration validation
func TestValidateEKSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid - kubeconfig provided",
			data: map[string]interface{}{
				KeyKubeconfig: "/path/to/kubeconfig",
			},
			expectError: false,
		},
		{
			name: "Valid - EKS cluster with aws_region",
			data: map[string]interface{}{
				KeyEKSClusterName: "my-cluster",
				KeyAWSRegion:      "us-west-2",
			},
			expectError: false,
		},
		{
			name: "Valid - EKS cluster with eks_cluster_region",
			data: map[string]interface{}{
				KeyEKSClusterName:   "my-cluster",
				KeyEKSClusterRegion: "us-west-2",
			},
			expectError: false,
		},
		{
			name: "Valid - EKS cluster with both endpoint and CA",
			data: map[string]interface{}{
				KeyEKSClusterName:     "my-cluster",
				KeyAWSRegion:          "us-west-2",
				KeyEKSClusterEndpoint: "https://example.eks.amazonaws.com",
				KeyEKSClusterCA:       "LS0tLS1CRUdJTi0tLS0t",
			},
			expectError: false,
		},
		{
			name: "Valid - Both kubeconfig and EKS cluster (kubeconfig takes precedence)",
			data: map[string]interface{}{
				KeyKubeconfig:     "/path/to/kubeconfig",
				KeyEKSClusterName: "my-cluster",
			},
			expectError: false,
		},
		{
			name:        "Invalid - Neither kubeconfig nor EKS cluster",
			data:        map[string]interface{}{},
			expectError: true,
			errorMsg:    "either 'kubeconfig' or 'eks_cluster_name' must be provided",
		},
		{
			name: "Invalid - EKS cluster without region",
			data: map[string]interface{}{
				KeyEKSClusterName: "my-cluster",
			},
			expectError: true,
			errorMsg:    "either eks_cluster_region or aws_region must be provided",
		},
		{
			name: "Invalid - EKS cluster with endpoint but no CA",
			data: map[string]interface{}{
				KeyEKSClusterName:     "my-cluster",
				KeyAWSRegion:          "us-west-2",
				KeyEKSClusterEndpoint: "https://example.eks.amazonaws.com",
			},
			expectError: true,
			errorMsg:    "eks_cluster_endpoint and eks_cluster_ca must be provided together",
		},
		{
			name: "Invalid - EKS cluster with CA but no endpoint",
			data: map[string]interface{}{
				KeyEKSClusterName: "my-cluster",
				KeyAWSRegion:      "us-west-2",
				KeyEKSClusterCA:   "LS0tLS1CRUdJTi0tLS0t",
			},
			expectError: true,
			errorMsg:    "eks_cluster_endpoint and eks_cluster_ca must be provided together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock data with defaults
			mockData := &mockResourceRead{
				data: make(map[string]interface{}),
			}

			// Set defaults for all keys
			mockData.data[KeyKubeconfig] = ""
			mockData.data[KeyEKSClusterName] = ""
			mockData.data[KeyEKSClusterRegion] = ""
			mockData.data[KeyAWSRegion] = ""
			mockData.data[KeyEKSClusterEndpoint] = ""
			mockData.data[KeyEKSClusterCA] = ""

			// Override with test data
			for k, v := range tt.data {
				mockData.data[k] = v
			}

			// Validate
			err := validateEKSConfiguration(mockData)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, but got none", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}
