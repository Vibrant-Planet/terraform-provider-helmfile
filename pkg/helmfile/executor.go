package helmfile

import (
	"context"
)

// HelmfileExecutor defines the interface for executing helmfile operations.
// This abstraction allows for multiple implementations (binary vs library).
type HelmfileExecutor interface {
	// Apply runs helmfile apply/sync to deploy releases
	Apply(ctx context.Context, opts *ApplyOptions) (*Result, error)

	// Diff runs helmfile diff to show changes
	Diff(ctx context.Context, opts *DiffOptions) (*Result, error)

	// Template runs helmfile template to render manifests
	Template(ctx context.Context, opts *TemplateOptions) (*Result, error)

	// Destroy runs helmfile destroy to delete releases
	Destroy(ctx context.Context, opts *DestroyOptions) (*Result, error)

	// Build runs helmfile build to validate configuration
	Build(ctx context.Context, opts *BuildOptions) (*Result, error)

	// Version returns the helmfile version
	Version(ctx context.Context) (string, error)
}

// Result contains the output from a helmfile operation
type Result struct {
	// Output is the stdout/stderr from the operation
	Output string

	// ExitCode is the exit code (0 for success)
	ExitCode int

	// Error is any error that occurred (may be nil even if ExitCode != 0)
	Error error
}

// BaseOptions contains common options for all helmfile operations
type BaseOptions struct {
	// FileOrDir is the path to helmfile.yaml or directory containing it
	FileOrDir string

	// WorkingDirectory is the directory to run helmfile in
	WorkingDirectory string

	// Kubeconfig is the path to kubeconfig file
	Kubeconfig string

	// KubeContext is the kubernetes context to use
	KubeContext string

	// Namespace is the kubernetes namespace to use
	Namespace string

	// Environment is the helmfile environment to use
	Environment string

	// Selector is the label selector (map of key=value)
	Selector map[string]interface{}

	// Selectors is a list of label selectors (OR logic)
	Selectors []interface{}

	// ValuesFiles is a list of values files to pass
	ValuesFiles []interface{}

	// Values is a list of inline values (YAML strings)
	Values []interface{}

	// EnvironmentVariables are environment variables to set
	EnvironmentVariables map[string]interface{}

	// HelmBinary is the path to helm binary
	HelmBinary string

	// HelmfileBinary is the path to helmfile binary (for binary executor)
	HelmfileBinary string

	// EnableGoTemplate enables Go template rendering (.gotmpl extension)
	EnableGoTemplate bool
}

// ApplyOptions contains options for helmfile apply/sync
type ApplyOptions struct {
	BaseOptions

	// Concurrency is the number of concurrent operations
	Concurrency int

	// ReleasesValues is a map of release-specific values
	ReleasesValues map[string]interface{}

	// SkipDiffOnInstall skips diff when installing (helmfile >= 0.136.0)
	SkipDiffOnInstall bool

	// SuppressSecrets suppresses secret values in output
	SuppressSecrets bool
}

// DiffOptions contains options for helmfile diff
type DiffOptions struct {
	BaseOptions

	// Concurrency is the number of concurrent operations
	Concurrency int

	// ReleasesValues is a map of release-specific values
	ReleasesValues map[string]interface{}

	// DetailedExitcode enables detailed exit codes
	DetailedExitcode bool

	// SuppressSecrets suppresses secret values in output
	SuppressSecrets bool

	// Context is the number of lines of context (default: 3)
	Context int

	// MaxDiffOutputLen is the maximum length of diff output
	MaxDiffOutputLen int
}

// TemplateOptions contains options for helmfile template
type TemplateOptions struct {
	BaseOptions

	// Concurrency is the number of concurrent operations
	Concurrency int

	// IncludeCRDs includes CRDs in output
	IncludeCRDs bool

	// OutputDir is the directory to write templates to
	OutputDir string

	// OutputDirTemplate is the template for output directory structure
	OutputDirTemplate string
}

// DestroyOptions contains options for helmfile destroy
type DestroyOptions struct {
	BaseOptions

	// Concurrency is the number of concurrent operations
	Concurrency int
}

// BuildOptions contains options for helmfile build
type BuildOptions struct {
	BaseOptions

	// EmbedValues embeds values inline (helmfile >= 0.126.0)
	EmbedValues bool
}
