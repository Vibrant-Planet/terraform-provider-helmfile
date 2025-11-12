# Implementation Plan: EKS Auto-Kubeconfig Generation

## Overview

This document outlines the implementation plan for adding automatic EKS kubeconfig generation to the terraform-provider-helmfile. This feature will allow users to specify an EKS cluster name and have the provider automatically generate the necessary kubeconfig file, eliminating manual kubeconfig management.

## Design Approach: Option 1 - EKS Cluster Parameters with Auto-Generated Kubeconfig

Users will provide EKS cluster parameters, and the provider will:
1. Call AWS EKS API to fetch cluster details (endpoint, CA certificate)
2. Generate a temporary kubeconfig file with AWS exec plugin authentication
3. Use this kubeconfig for all helmfile operations
4. Clean up the temporary file on resource deletion

### User-Facing API

```hcl
resource "helmfile_release_set" "mystack" {
  # New EKS parameters
  eks_cluster_name = "my-cluster"
  eks_cluster_region = "us-west-2"  # Optional, defaults to aws_region if set

  # Optional: override defaults
  eks_cluster_endpoint = "https://..."  # Auto-discovered if not set
  eks_cluster_ca = "base64-encoded-ca"   # Auto-discovered if not set

  # Existing AWS auth (reused for EKS auth)
  aws_region = "us-west-2"
  aws_profile = "my-profile"
  aws_assume_role {
    role_arn = "arn:aws:iam::123456789:role/my-role"
  }

  # kubeconfig becomes optional when eks_cluster_name is set
  # If not provided, a temporary kubeconfig is auto-generated
  # kubeconfig = "/path/to/kubeconfig"  # Can still override

  environment = "prod"
  # ... rest of config
}
```

## Implementation Tasks

### 1. Add EKS-related schema fields to ReleaseSetSchema

**File:** `pkg/helmfile/resource_release_set.go`

Add these new schema fields after the existing fields (around line 196):

```go
KeyEKSClusterName: {
    Type:        schema.TypeString,
    Optional:    true,
    ForceNew:    false,
    Description: "EKS cluster name for automatic kubeconfig generation",
},
KeyEKSClusterRegion: {
    Type:        schema.TypeString,
    Optional:    true,
    ForceNew:    false,
    Description: "AWS region for EKS cluster (defaults to aws_region if not set)",
},
KeyEKSClusterEndpoint: {
    Type:        schema.TypeString,
    Optional:    true,
    Computed:    true,
    Description: "EKS cluster endpoint (auto-discovered from AWS if not provided)",
},
KeyEKSClusterCA: {
    Type:        schema.TypeString,
    Optional:    true,
    Computed:    true,
    Sensitive:   true,
    Description: "EKS cluster certificate authority data (auto-discovered from AWS if not provided)",
},
```

**File:** `pkg/helmfile/schema.go`

Add constants:

```go
const (
    KeyAWSRegion          = "aws_region"
    KeyAWSProfile         = "aws_profile"
    KeyAWSAssumeRole      = "aws_assume_role"
    KeyEKSClusterName     = "eks_cluster_name"
    KeyEKSClusterRegion   = "eks_cluster_region"
    KeyEKSClusterEndpoint = "eks_cluster_endpoint"
    KeyEKSClusterCA       = "eks_cluster_ca"
)
```

### 2. Make kubeconfig field conditionally optional in schema validation

**File:** `pkg/helmfile/resource_release_set.go`

Change the kubeconfig schema (line 99-103) from:

```go
KeyKubeconfig: {
    Type:     schema.TypeString,
    Required: true,  // Change this
    ForceNew: false,
},
```

To:

```go
KeyKubeconfig: {
    Type:        schema.TypeString,
    Optional:    true,  // Now optional
    Computed:    true,  // Can be computed from EKS params
    ForceNew:    false,
    Description: "Path to kubeconfig file. Optional when eks_cluster_name is provided.",
},
```

**File:** `pkg/helmfile/release_set.go`

Add validation function (call it from `NewReleaseSet()`):

```go
func validateKubeconfigOrEKS(d api.Getter) error {
    kubeconfig := d.Get(KeyKubeconfig).(string)
    eksClusterName := d.Get(KeyEKSClusterName).(string)

    if kubeconfig == "" && eksClusterName == "" {
        return fmt.Errorf("either 'kubeconfig' or 'eks_cluster_name' must be provided")
    }
    return nil
}
```

### 3. Create eks_kubeconfig.go with cluster info fetching logic

**New File:** `pkg/helmfile/eks_kubeconfig.go`

```go
package helmfile

import (
    "context"
    "encoding/base64"
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/eks"
    "github.com/mumoshu/terraform-provider-eksctl/pkg/sdk"
)

type EKSClusterConfig struct {
    ClusterName string
    Region      string
    Endpoint    string
    CA          string
    AWSProfile  string
    AssumeRole  *AssumeRoleConfig // Reuse from existing context.go
}

// fetchEKSClusterInfo retrieves EKS cluster details from AWS
func fetchEKSClusterInfo(ctx *sdk.Context, clusterName, region string) (*EKSClusterConfig, error) {
    // Create AWS session using the context's AWS config
    // Call eks:DescribeCluster
    // Extract endpoint and CA certificate
    // Return populated EKSClusterConfig

    // Implementation:
    // 1. Create AWS session from context
    // 2. Create EKS client
    // 3. Call DescribeCluster API
    // 4. Extract cluster.Endpoint and cluster.CertificateAuthority.Data
    // 5. Return EKSClusterConfig

    return nil, fmt.Errorf("not implemented")
}

// getEKSRegion returns the region to use for EKS, preferring eks_cluster_region
// over aws_region
func getEKSRegion(d api.Getter) string {
    if region := d.Get(KeyEKSClusterRegion).(string); region != "" {
        return region
    }
    return d.Get(KeyAWSRegion).(string)
}
```

### 4. Implement kubeconfig YAML generation with exec plugin auth

**File:** `pkg/helmfile/eks_kubeconfig.go`

Add kubeconfig structure and generation:

```go
// KubeconfigData represents a Kubernetes kubeconfig file structure
type KubeconfigData struct {
    APIVersion     string          `yaml:"apiVersion"`
    Kind           string          `yaml:"kind"`
    Clusters       []ClusterEntry  `yaml:"clusters"`
    Contexts       []ContextEntry  `yaml:"contexts"`
    CurrentContext string          `yaml:"current-context"`
    Users          []UserEntry     `yaml:"users"`
}

type ClusterEntry struct {
    Name    string        `yaml:"name"`
    Cluster ClusterDetail `yaml:"cluster"`
}

type ClusterDetail struct {
    Server                   string `yaml:"server"`
    CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

type ContextEntry struct {
    Name    string        `yaml:"name"`
    Context ContextDetail `yaml:"context"`
}

type ContextDetail struct {
    Cluster string `yaml:"cluster"`
    User    string `yaml:"user"`
}

type UserEntry struct {
    Name string     `yaml:"name"`
    User UserDetail `yaml:"user"`
}

type UserDetail struct {
    Exec ExecConfig `yaml:"exec"`
}

type ExecConfig struct {
    APIVersion string         `yaml:"apiVersion"`
    Command    string         `yaml:"command"`
    Args       []string       `yaml:"args"`
    Env        []ExecEnvVar   `yaml:"env,omitempty"`
}

type ExecEnvVar struct {
    Name  string `yaml:"name"`
    Value string `yaml:"value"`
}

// generateKubeconfigYAML creates a kubeconfig YAML string with AWS exec plugin
func generateKubeconfigYAML(config *EKSClusterConfig) (string, error) {
    // Build exec args
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

    return string(yamlBytes), nil
}
```

### 5. Add temporary file management and cleanup logic

**File:** `pkg/helmfile/eks_kubeconfig.go`

Add file management functions:

```go
import (
    "crypto/rand"
    "encoding/hex"
    "io/ioutil"
    "os"
    "path/filepath"
)

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
    if dir == "" {
        dir = os.TempDir()
    }

    // Create filename
    filename := fmt.Sprintf(".terraform-helmfile-kubeconfig-%s-%s", clusterName, randomSuffix)
    filepath := filepath.Join(dir, filename)

    // Write file with restrictive permissions (owner read/write only)
    if err := ioutil.WriteFile(filepath, []byte(kubeconfigYAML), 0600); err != nil {
        return "", fmt.Errorf("writing kubeconfig to %s: %w", filepath, err)
    }

    logf("Generated temporary kubeconfig at: %s", filepath)
    return filepath, nil
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
```

### 6. Integrate EKS kubeconfig generation into NewReleaseSet()

**File:** `pkg/helmfile/release_set.go`

Modify the `NewReleaseSet()` function to integrate EKS kubeconfig generation:

```go
func NewReleaseSet(d api.Getter) (*ReleaseSet, error) {
    // Validate configuration
    if err := validateKubeconfigOrEKS(d); err != nil {
        return nil, err
    }

    if err := validateEKSConfiguration(d); err != nil {
        return nil, err
    }

    // ... existing code to read other fields ...

    kubeconfig := d.Get(KeyKubeconfig).(string)
    eksClusterName := d.Get(KeyEKSClusterName).(string)

    var generatedKubeconfig string

    // If EKS cluster name provided and no kubeconfig, generate it
    if eksClusterName != "" && kubeconfig == "" {
        ctx := newContext(d)
        region := getEKSRegion(d)

        logf("Generating kubeconfig for EKS cluster: %s in region: %s", eksClusterName, region)

        // Check if endpoint and CA are manually provided
        manualEndpoint := d.Get(KeyEKSClusterEndpoint).(string)
        manualCA := d.Get(KeyEKSClusterCA).(string)

        var clusterConfig *EKSClusterConfig

        if manualEndpoint != "" && manualCA != "" {
            // Use manually provided values
            logf("Using manually provided EKS cluster endpoint and CA")
            clusterConfig = &EKSClusterConfig{
                ClusterName: eksClusterName,
                Region:      region,
                Endpoint:    manualEndpoint,
                CA:          manualCA,
                AWSProfile:  d.Get(KeyAWSProfile).(string),
            }
        } else {
            // Fetch cluster info from AWS
            logf("Fetching EKS cluster info from AWS API")
            var err error
            clusterConfig, err = fetchEKSClusterInfo(ctx, eksClusterName, region)
            if err != nil {
                return nil, fmt.Errorf("fetching EKS cluster info: %w", err)
            }

            // Store computed values back to schema
            d.Set(KeyEKSClusterEndpoint, clusterConfig.Endpoint)
            d.Set(KeyEKSClusterCA, clusterConfig.CA)
        }

        // Generate kubeconfig YAML
        kubeconfigYAML, err := generateKubeconfigYAML(clusterConfig)
        if err != nil {
            return nil, fmt.Errorf("generating kubeconfig: %w", err)
        }

        // Write to temporary file
        workingDir := d.Get(KeyWorkingDirectory).(string)
        generatedKubeconfig, err = writeTemporaryKubeconfig(kubeconfigYAML, workingDir, eksClusterName)
        if err != nil {
            return nil, fmt.Errorf("writing kubeconfig: %w", err)
        }

        kubeconfig = generatedKubeconfig

        // Store computed kubeconfig path back to schema
        d.Set(KeyKubeconfig, kubeconfig)
    }

    // Continue with existing ReleaseSet creation
    fs := &ReleaseSet{
        // ... existing fields ...
        Kubeconfig:          kubeconfig,
        GeneratedKubeconfig: generatedKubeconfig, // Track for cleanup
    }

    return fs, nil
}
```

**File:** `pkg/helmfile/release_set.go`

Update the `ReleaseSet` struct to include the generated kubeconfig path:

```go
type ReleaseSet struct {
    // ... existing fields ...
    Kubeconfig          string
    GeneratedKubeconfig string // Path to auto-generated kubeconfig (for cleanup)
}
```

Add cleanup in `DeleteReleaseSet()`:

```go
func DeleteReleaseSet(ctx *sdk.Context, fs *ReleaseSet, d *schema.ResourceData, executor HelmfileExecutor) error {
    // ... existing delete logic ...

    // Cleanup generated kubeconfig if exists
    if fs.GeneratedKubeconfig != "" {
        if err := cleanupKubeconfig(fs.GeneratedKubeconfig); err != nil {
            logf("Warning: failed to cleanup generated kubeconfig: %v", err)
            // Don't fail the delete operation due to cleanup failure
        }
    }

    return nil
}
```

### 7. Add validation logic for EKS parameter combinations

**File:** `pkg/helmfile/release_set.go`

Add comprehensive validation function:

```go
func validateEKSConfiguration(d api.Getter) error {
    eksClusterName := d.Get(KeyEKSClusterName).(string)
    eksClusterRegion := d.Get(KeyEKSClusterRegion).(string)
    awsRegion := d.Get(KeyAWSRegion).(string)
    eksEndpoint := d.Get(KeyEKSClusterEndpoint).(string)
    eksCA := d.Get(KeyEKSClusterCA).(string)

    // If eks_cluster_name is not set, no EKS validation needed
    if eksClusterName == "" {
        return nil
    }

    // If eks_cluster_name is set, need either eks_cluster_region or aws_region
    if eksClusterRegion == "" && awsRegion == "" {
        return fmt.Errorf("when eks_cluster_name is set, either eks_cluster_region or aws_region must be provided")
    }

    // Validate endpoint/CA provided together (if manually specified)
    if (eksEndpoint != "" && eksCA == "") || (eksEndpoint == "" && eksCA != "") {
        return fmt.Errorf("eks_cluster_endpoint and eks_cluster_ca must be provided together or both omitted for auto-discovery")
    }

    return nil
}
```

### 8. Update provider documentation with EKS auto-config examples

**File:** `README.md`

Add new section after "AWS authentication and AssumeRole support" (around line 310):

```markdown
### EKS Automatic Kubeconfig Generation

The provider can automatically generate kubeconfig files for EKS clusters, eliminating the need to manage kubeconfig files manually.

When you provide `eks_cluster_name`, the provider will:
1. Call the AWS EKS API to fetch cluster endpoint and CA certificate
2. Generate a temporary kubeconfig file with AWS exec plugin authentication
3. Use this kubeconfig for all helmfile operations
4. Clean up the temporary file when the resource is destroyed

#### Basic Usage

Minimal configuration - just provide cluster name and region:

```hcl
resource "helmfile_release_set" "mystack" {
  eks_cluster_name = "my-eks-cluster"
  aws_region       = "us-west-2"

  # No kubeconfig needed! Provider auto-generates it

  environment = "prod"
  content = file("./helmfile.yaml")
}
```

#### With AWS AssumeRole

Use with cross-account or role-based access:

```hcl
resource "helmfile_release_set" "mystack" {
  eks_cluster_name   = "my-eks-cluster"
  eks_cluster_region = "us-west-2"  # Can differ from assumed role region

  aws_region = "us-east-1"
  aws_profile = "terraform"
  aws_assume_role {
    role_arn = "arn:aws:iam::123456789:role/EKSAdmin"
  }

  environment = "prod"
  content = file("./helmfile.yaml")
}
```

#### Integration with Terraform AWS Provider

Seamlessly integrate with other Terraform resources:

```hcl
data "aws_eks_cluster" "cluster" {
  name = "my-eks-cluster"
}

resource "helmfile_release_set" "mystack" {
  eks_cluster_name = data.aws_eks_cluster.cluster.name
  aws_region       = data.aws_eks_cluster.cluster.arn

  # The provider will fetch cluster details automatically

  environment = "prod"
  content = file("./helmfile.yaml")

  # Can reference cluster attributes
  environment_variables = {
    CLUSTER_ENDPOINT = data.aws_eks_cluster.cluster.endpoint
  }
}
```

#### Manual Endpoint and CA Override

Skip the AWS API call by providing cluster details manually:

```hcl
resource "helmfile_release_set" "mystack" {
  eks_cluster_name = "my-eks-cluster"

  # Manually provide cluster details (skips AWS API call)
  eks_cluster_endpoint = "https://ABC123.gr7.us-west-2.eks.amazonaws.com"
  eks_cluster_ca       = "LS0tLS1CRUdJTi..." # base64 encoded CA

  aws_region = "us-west-2"

  environment = "prod"
  content = file("./helmfile.yaml")
}
```

#### Complete Override with Custom Kubeconfig

You can still provide a custom kubeconfig to override auto-generation entirely:

```hcl
resource "helmfile_release_set" "mystack" {
  eks_cluster_name = "my-eks-cluster"  # Informational only

  # Custom kubeconfig takes precedence
  kubeconfig = "/custom/path/to/kubeconfig"

  environment = "prod"
  content = file("./helmfile.yaml")
}
```

#### How It Works

The provider generates a kubeconfig with the AWS exec plugin for authentication:

1. **Cluster Discovery**: Calls `eks:DescribeCluster` to get endpoint and CA certificate
2. **Kubeconfig Generation**: Creates a kubeconfig using the AWS exec plugin pattern
3. **Authentication**: kubectl/helm authenticate using `aws eks get-token` command
4. **Cleanup**: Temporary kubeconfig is removed on resource deletion

The generated kubeconfig uses the same AWS credentials (profile, region, assume role) that you configured for the provider, ensuring consistent authentication across all operations.

#### Requirements

- AWS credentials with `eks:DescribeCluster` permission
- `aws` CLI available in PATH (used by kubectl for authentication)
- IAM permissions to access the EKS cluster (kubernetes RBAC)
```

## Key Design Decisions

### 1. Temporary Files in Working Directory
Store generated kubeconfig files in the working directory (or system temp as fallback) with a predictable naming pattern. This makes debugging easier and allows users to inspect the generated configuration if needed.

**Filename pattern:** `.terraform-helmfile-kubeconfig-{clusterName}-{randomId}`

### 2. exec Plugin over AWS IAM Authenticator
Use the AWS CLI's `aws eks get-token` command via the exec plugin rather than requiring a separate AWS IAM authenticator binary. This is the standard approach and reduces external dependencies.

### 3. Computed Fields for Flexibility
Make `eks_cluster_endpoint` and `eks_cluster_ca` optional but computed. This allows:
- Auto-discovery from AWS (default behavior)
- Manual override to skip AWS API calls
- Visibility of discovered values in Terraform state

### 4. Reuse Existing AWS Context
Leverage the existing `newContext(d)` function from `context.go` which already handles AWS profile, region, and assume role. This ensures consistent AWS authentication across all provider operations.

### 5. Graceful Cleanup
Don't fail resource operations if temporary kubeconfig cleanup fails. Log warnings but allow the operation to succeed. The temp files are not critical and will be cleaned up by OS temp directory cleanup or by subsequent runs.

### 6. Validation at Multiple Levels
- Schema-level: Type validation, required/optional fields
- Function-level: Cross-field validation (e.g., endpoint and CA together)
- Runtime: AWS API errors, file I/O errors

## Testing Strategy

After implementation, create tests for:

1. **Basic EKS cluster name + region**
   - Verify AWS API call to DescribeCluster
   - Verify kubeconfig generation
   - Verify helmfile operations succeed

2. **With aws_assume_role**
   - Verify role assumption works
   - Verify assumed credentials used for EKS API
   - Verify assumed credentials propagated to exec plugin

3. **Manual endpoint/CA override**
   - Verify no AWS API call made
   - Verify manual values used in kubeconfig

4. **Kubeconfig parameter override**
   - Verify custom kubeconfig takes precedence
   - Verify no auto-generation occurs

5. **Error cases**
   - Invalid cluster name
   - Missing AWS permissions
   - Cluster in different account without assume role
   - Missing required fields

6. **Cleanup on destroy**
   - Verify temporary kubeconfig deleted
   - Verify cleanup doesn't fail destroy operation

7. **Integration tests**
   - Deploy actual helmfile releases to EKS
   - Verify updates work
   - Verify destroy works

## Implementation Order

Recommended order of implementation:

1. Schema changes (tasks 1-2) - Foundation
2. EKS kubeconfig core logic (tasks 3-4) - Core functionality
3. File management (task 5) - Supporting functionality
4. Integration (task 6) - Tie it all together
5. Validation (task 7) - Polish and error handling
6. Documentation (task 8) - User-facing documentation

## Dependencies

### Go Packages Required
- `github.com/aws/aws-sdk-go/aws` - Already in go.mod
- `github.com/aws/aws-sdk-go/service/eks` - Already in go.mod
- `gopkg.in/yaml.v2` or `gopkg.in/yaml.v3` - Already in go.mod
- Standard library: `os`, `io/ioutil`, `path/filepath`, `crypto/rand`, `encoding/hex`

### AWS Permissions Required
For the provider to function, the AWS credentials used must have:
- `eks:DescribeCluster` - To fetch cluster details
- Appropriate IAM/RBAC permissions to interact with the cluster

### Runtime Requirements
- `aws` CLI in PATH (required by kubectl exec plugin for authentication)
- Network access to AWS EKS API
- Network access to Kubernetes API endpoint

## Future Enhancements

Potential future improvements (out of scope for initial implementation):

1. **Support for other cloud providers**
   - GKE automatic kubeconfig generation
   - AKS automatic kubeconfig generation

2. **Kubeconfig caching**
   - Cache generated kubeconfig between runs
   - Invalidate cache when cluster details change

3. **In-memory kubeconfig**
   - Pass kubeconfig via environment variable instead of file
   - Requires investigation of helmfile library support

4. **Multiple cluster support**
   - Generate kubeconfig with multiple clusters/contexts
   - Allow helmfile to target different clusters

5. **Custom exec plugin commands**
   - Support for custom authentication methods
   - Plugin for other AWS authentication tools

## References

- [AWS EKS User Guide - Create Kubeconfig](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html)
- [Kubernetes Authentication - Exec Plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins)
- [AWS CLI eks get-token](https://awscli.amazonaws.com/v2/documentation/api/latest/reference/eks/get-token.html)
- [Terraform Provider SDK - Schema](https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors)
