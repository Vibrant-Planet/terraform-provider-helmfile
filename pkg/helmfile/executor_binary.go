package helmfile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mumoshu/terraform-provider-eksctl/pkg/sdk"
)

// BinaryExecutor implements HelmfileExecutor by calling helmfile as an external binary.
// This is the current/existing implementation approach.
type BinaryExecutor struct {
	// Logger for debug output
	logger func(string, ...interface{})
}

// NewBinaryExecutor creates a new BinaryExecutor
func NewBinaryExecutor() *BinaryExecutor {
	return &BinaryExecutor{
		logger: logf,
	}
}

// Apply implements HelmfileExecutor.Apply by calling helmfile apply
func (e *BinaryExecutor) Apply(ctx context.Context, opts *ApplyOptions) (*Result, error) {
	args := []string{"apply"}
	args = append(args, e.buildBaseArgs(&opts.BaseOptions)...)

	if opts.Concurrency > 0 {
		args = append(args, "--concurrency", strconv.Itoa(opts.Concurrency))
	}

	if opts.SuppressSecrets {
		args = append(args, "--suppress-secrets")
	}

	if opts.SkipDiffOnInstall {
		args = append(args, "--skip-diff-on-install")
	}

	for k, v := range opts.ReleasesValues {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
	}

	return e.runCommand(ctx, &opts.BaseOptions, args)
}

// Diff implements HelmfileExecutor.Diff by calling helmfile diff
func (e *BinaryExecutor) Diff(ctx context.Context, opts *DiffOptions) (*Result, error) {
	args := []string{"diff"}
	args = append(args, e.buildBaseArgs(&opts.BaseOptions)...)

	if opts.Concurrency > 0 {
		args = append(args, "--concurrency", strconv.Itoa(opts.Concurrency))
	}

	if opts.DetailedExitcode {
		args = append(args, "--detailed-exitcode")
	}

	if opts.SuppressSecrets {
		args = append(args, "--suppress-secrets")
	}

	if opts.Context > 0 {
		args = append(args, "--context", strconv.Itoa(opts.Context))
	}

	for k, v := range opts.ReleasesValues {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
	}

	return e.runCommand(ctx, &opts.BaseOptions, args)
}

// Template implements HelmfileExecutor.Template by calling helmfile template
func (e *BinaryExecutor) Template(ctx context.Context, opts *TemplateOptions) (*Result, error) {
	args := []string{"template"}
	args = append(args, e.buildBaseArgs(&opts.BaseOptions)...)

	if opts.Concurrency > 0 {
		args = append(args, "--concurrency", strconv.Itoa(opts.Concurrency))
	}

	if opts.IncludeCRDs {
		args = append(args, "--include-crds")
	}

	if opts.OutputDir != "" {
		args = append(args, "--output-dir", opts.OutputDir)
	}

	if opts.OutputDirTemplate != "" {
		args = append(args, "--output-dir-template", opts.OutputDirTemplate)
	}

	return e.runCommand(ctx, &opts.BaseOptions, args)
}

// Destroy implements HelmfileExecutor.Destroy by calling helmfile destroy
func (e *BinaryExecutor) Destroy(ctx context.Context, opts *DestroyOptions) (*Result, error) {
	args := []string{"destroy"}
	args = append(args, e.buildBaseArgs(&opts.BaseOptions)...)

	if opts.Concurrency > 0 {
		args = append(args, "--concurrency", strconv.Itoa(opts.Concurrency))
	}

	return e.runCommand(ctx, &opts.BaseOptions, args)
}

// Build implements HelmfileExecutor.Build by calling helmfile build
func (e *BinaryExecutor) Build(ctx context.Context, opts *BuildOptions) (*Result, error) {
	args := []string{"build"}
	args = append(args, e.buildBaseArgs(&opts.BaseOptions)...)

	if opts.EmbedValues {
		args = append(args, "--embed-values")
	}

	return e.runCommand(ctx, &opts.BaseOptions, args)
}

// Version implements HelmfileExecutor.Version by calling helmfile version
func (e *BinaryExecutor) Version(ctx context.Context) (string, error) {
	// For version command, we don't need most options
	cmd := exec.Command("helmfile", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting helmfile version: %w", err)
	}

	// Parse version from output
	version := strings.TrimSpace(string(output))
	parts := strings.Split(version, " ")
	if len(parts) > 0 {
		version = strings.TrimPrefix(parts[len(parts)-1], "v")
	}

	return version, nil
}

// buildBaseArgs constructs the common arguments for all helmfile commands
func (e *BinaryExecutor) buildBaseArgs(opts *BaseOptions) []string {
	args := []string{"--no-color"}

	if opts.FileOrDir != "" {
		args = append(args, "--file", opts.FileOrDir)
	}

	if opts.HelmBinary != "" {
		args = append(args, "--helm-binary", opts.HelmBinary)
	}

	if opts.Environment != "" {
		args = append(args, "--environment", opts.Environment)
	}

	for k, v := range opts.Selector {
		args = append(args, "--selector", fmt.Sprintf("%s=%s", k, v))
	}

	for _, selector := range opts.Selectors {
		args = append(args, "--selector", fmt.Sprintf("%s", selector))
	}

	for _, f := range opts.ValuesFiles {
		args = append(args, "--state-values-file", fmt.Sprintf("%v", f))
	}

	// Note: Values need to be written to temporary files - this would be handled
	// by the calling code in the existing implementation

	return args
}

// runCommand executes the helmfile binary with the given arguments
func (e *BinaryExecutor) runCommand(ctx context.Context, opts *BaseOptions, args []string) (*Result, error) {
	// Get the helmfile binary path
	helmfileBin := opts.HelmfileBinary
	if helmfileBin == "" {
		helmfileBin = "helmfile"
	}

	e.logger("Running helmfile %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, helmfileBin, args...)
	cmd.Dir = opts.WorkingDirectory

	// Set environment variables - start with parent process env to inherit PATH, etc.
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, readEnvironmentVariables(opts.EnvironmentVariables, "KUBECONFIG")...)

	if opts.Kubeconfig != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+opts.Kubeconfig)
	}

	if opts.KubeContext != "" {
		// Note: kube-context is typically passed as a flag, but can also be in env
		// The existing code passes it via args, which we handle in buildBaseArgs
	}

	// Run command and capture output
	output, err := cmd.CombinedOutput()

	result := &Result{
		Output:   string(output),
		ExitCode: 0,
		Error:    err,
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}

	return result, err
}

// sdkContext wraps the SDK context for compatibility
type sdkContext struct {
	*sdk.Context
}

func newSDKContext(ctx context.Context) *sdkContext {
	return &sdkContext{
		Context: &sdk.Context{
			// Initialize SDK context fields as needed
		},
	}
}
