package helmfile

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/mumoshu/terraform-provider-eksctl/pkg/sdk"
	"github.com/mumoshu/terraform-provider-eksctl/pkg/sdk/api"
	"gopkg.in/yaml.v2"
)

// EKSClusterConfig contains the configuration needed to generate a kubeconfig for an EKS cluster
type EKSClusterConfig struct {
	ClusterName string
	Region      string
	Endpoint    string
	CA          string
	AWSProfile  string
}

// fetchEKSClusterInfo retrieves EKS cluster details from AWS API
func fetchEKSClusterInfo(ctx *sdk.Context, clusterName, region string) (*EKSClusterConfig, error) {
	logf("Fetching EKS cluster info for cluster: %s in region: %s", clusterName, region)

	// Get AWS session from the context
	sess := ctx.Session()
	if sess == nil {
		return nil, fmt.Errorf("AWS session is nil - ensure AWS credentials are configured")
	}

	// Create EKS client
	eksClient := eks.New(sess, &aws.Config{Region: aws.String(region)})

	// Call DescribeCluster API
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	result, err := eksClient.DescribeCluster(input)
	if err != nil {
		return nil, fmt.Errorf("describing EKS cluster %s: %w", clusterName, err)
	}

	if result.Cluster == nil {
		return nil, fmt.Errorf("EKS cluster %s not found in region %s", clusterName, region)
	}

	cluster := result.Cluster

	// Validate required fields
	if cluster.Endpoint == nil || *cluster.Endpoint == "" {
		return nil, fmt.Errorf("EKS cluster %s has no endpoint", clusterName)
	}

	if cluster.CertificateAuthority == nil || cluster.CertificateAuthority.Data == nil || *cluster.CertificateAuthority.Data == "" {
		return nil, fmt.Errorf("EKS cluster %s has no certificate authority data", clusterName)
	}

	config := &EKSClusterConfig{
		ClusterName: clusterName,
		Region:      region,
		Endpoint:    *cluster.Endpoint,
		CA:          *cluster.CertificateAuthority.Data,
	}

	logf("Successfully fetched EKS cluster info: endpoint=%s", config.Endpoint)

	return config, nil
}

// getEKSRegion returns the region to use for EKS operations
// Prefers eks_cluster_region over aws_region
func getEKSRegion(d api.Getter) string {
	if region := d.Get(KeyEKSClusterRegion).(string); region != "" {
		return region
	}
	return d.Get(KeyAWSRegion).(string)
}

// KubeconfigData represents a Kubernetes kubeconfig file structure
type KubeconfigData struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	Clusters       []ClusterEntry `yaml:"clusters"`
	Contexts       []ContextEntry `yaml:"contexts"`
	CurrentContext string         `yaml:"current-context"`
	Users          []UserEntry    `yaml:"users"`
}

// ClusterEntry represents a cluster in the kubeconfig
type ClusterEntry struct {
	Name    string        `yaml:"name"`
	Cluster ClusterDetail `yaml:"cluster"`
}

// ClusterDetail contains cluster connection details
type ClusterDetail struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

// ContextEntry represents a context in the kubeconfig
type ContextEntry struct {
	Name    string        `yaml:"name"`
	Context ContextDetail `yaml:"context"`
}

// ContextDetail contains context configuration
type ContextDetail struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

// UserEntry represents a user in the kubeconfig
type UserEntry struct {
	Name string     `yaml:"name"`
	User UserDetail `yaml:"user"`
}

// UserDetail contains user authentication details
type UserDetail struct {
	Exec ExecConfig `yaml:"exec"`
}

// ExecConfig configures exec-based authentication
type ExecConfig struct {
	APIVersion string       `yaml:"apiVersion"`
	Command    string       `yaml:"command"`
	Args       []string     `yaml:"args"`
	Env        []ExecEnvVar `yaml:"env,omitempty"`
}

// ExecEnvVar represents an environment variable for exec auth
type ExecEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// generateKubeconfigYAML creates a kubeconfig YAML string with AWS exec plugin authentication
func generateKubeconfigYAML(config *EKSClusterConfig) (string, error) {
	logf("Generating kubeconfig YAML for cluster: %s", config.ClusterName)

	// Build exec args for aws eks get-token
	args := []string{
		"eks",
		"get-token",
		"--cluster-name", config.ClusterName,
	}

	if config.Region != "" {
		args = append(args, "--region", config.Region)
	}

	// Build exec env vars
	var envVars []ExecEnvVar
	if config.AWSProfile != "" {
		envVars = append(envVars, ExecEnvVar{
			Name:  "AWS_PROFILE",
			Value: config.AWSProfile,
		})
	}

	// Build kubeconfig structure
	kubeconfig := KubeconfigData{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []ClusterEntry{
			{
				Name: config.ClusterName,
				Cluster: ClusterDetail{
					Server:                   config.Endpoint,
					CertificateAuthorityData: config.CA,
				},
			},
		},
		Contexts: []ContextEntry{
			{
				Name: config.ClusterName,
				Context: ContextDetail{
					Cluster: config.ClusterName,
					User:    config.ClusterName,
				},
			},
		},
		CurrentContext: config.ClusterName,
		Users: []UserEntry{
			{
				Name: config.ClusterName,
				User: UserDetail{
					Exec: ExecConfig{
						APIVersion: "client.authentication.k8s.io/v1beta1",
						Command:    "aws",
						Args:       args,
						Env:        envVars,
					},
				},
			},
		},
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(&kubeconfig)
	if err != nil {
		return "", fmt.Errorf("marshaling kubeconfig to YAML: %w", err)
	}

	logf("Successfully generated kubeconfig YAML (%d bytes)", len(yamlBytes))
	return string(yamlBytes), nil
}

// writeTemporaryKubeconfig writes the kubeconfig YAML to a temporary file
func writeTemporaryKubeconfig(kubeconfigYAML, workingDir, clusterName string) (string, error) {
	// Generate random suffix for uniqueness
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generating random suffix: %w", err)
	}
	randomSuffix := hex.EncodeToString(randomBytes)

	// Determine directory for temp file
	dir := workingDir
	if dir == "" || dir == "." {
		dir = os.TempDir()
	}

	// Create filename
	filename := fmt.Sprintf(".terraform-helmfile-kubeconfig-%s-%s", clusterName, randomSuffix)
	filePath := filepath.Join(dir, filename)

	// Write file with restrictive permissions (owner read/write only)
	if err := ioutil.WriteFile(filePath, []byte(kubeconfigYAML), 0600); err != nil {
		return "", fmt.Errorf("writing kubeconfig to %s: %w", filePath, err)
	}

	logf("Generated temporary kubeconfig at: %s", filePath)
	return filePath, nil
}

// cleanupKubeconfig removes the temporary kubeconfig file
func cleanupKubeconfig(path string) error {
	if path == "" {
		return nil
	}

	if err := os.Remove(path); err != nil {
		// Log but don't fail - file might already be deleted
		if !os.IsNotExist(err) {
			logf("Warning: failed to cleanup kubeconfig at %s: %v", path, err)
			return err
		}
	}

	logf("Cleaned up temporary kubeconfig at: %s", path)
	return nil
}
