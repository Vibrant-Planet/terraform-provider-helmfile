package helmfile

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/helmfile/helmfile/pkg/app"
	"go.uber.org/zap"
)

// LibraryExecutor implements HelmfileExecutor by calling helmfile as a Go library.
// This is the new implementation approach that embeds helmfile.
type LibraryExecutor struct {
	logger *zap.SugaredLogger
}

// NewLibraryExecutor creates a new LibraryExecutor
func NewLibraryExecutor(logger *zap.SugaredLogger) *LibraryExecutor {
	return &LibraryExecutor{
		logger: logger,
	}
}

// Apply implements HelmfileExecutor.Apply using helmfile library
func (e *LibraryExecutor) Apply(ctx context.Context, opts *ApplyOptions) (*Result, error) {
	// Build debug info about AWS environment
	var debugOutput strings.Builder
	debugOutput.WriteString("=== PROVIDER DEBUG INFO ===\n")
	debugOutput.WriteString("AWS Environment BEFORE setting:\n")
	awsVars := []string{"AWS_PROFILE", "AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "HOME", "AWS_CONFIG_FILE", "AWS_SHARED_CREDENTIALS_FILE"}
	for _, key := range awsVars {
		if val, exists := os.LookupEnv(key); exists {
			// Mask sensitive values
			if key == "AWS_SECRET_ACCESS_KEY" || key == "AWS_SESSION_TOKEN" {
				if len(val) > 4 {
					val = val[:4] + "***" + val[len(val)-4:]
				} else {
					val = "***"
				}
			}
			debugOutput.WriteString(fmt.Sprintf("  %s=%s\n", key, val))
		} else {
			debugOutput.WriteString(fmt.Sprintf("  %s=(not set)\n", key))
		}
	}
	debugOutput.WriteString(fmt.Sprintf("Environment variables from config: %v\n", opts.EnvironmentVariables))

	// Set environment variables before running helmfile
	// This ensures helm/kubectl can access AWS credentials
	restoreEnv := setEnvironmentVariables(opts.EnvironmentVariables)
	defer restoreEnv()

	// Log AWS environment AFTER setting
	debugOutput.WriteString("\nAWS Environment AFTER setting:\n")
	for _, key := range awsVars {
		if val, exists := os.LookupEnv(key); exists {
			// Mask sensitive values
			if key == "AWS_SECRET_ACCESS_KEY" || key == "AWS_SESSION_TOKEN" {
				if len(val) > 4 {
					val = val[:4] + "***" + val[len(val)-4:]
				} else {
					val = "***"
				}
			}
			debugOutput.WriteString(fmt.Sprintf("  %s=%s\n", key, val))
		} else {
			debugOutput.WriteString(fmt.Sprintf("  %s=(not set)\n", key))
		}
	}
	debugOutput.WriteString("=== END PROVIDER DEBUG INFO ===\n\n")

	// Create output capture
	capture := NewOutputCapture()
	captureLogger := CreateCaptureLogger(capture)

	// Create config provider with capture logger
	config := &applyConfigProvider{
		baseConfigProvider: newBaseConfigProvider(opts.BaseOptions, captureLogger),
		concurrency:        opts.Concurrency,
		suppressSecrets:    opts.SuppressSecrets,
		skipDiffOnInstall:  opts.SkipDiffOnInstall,
	}

	// Initialize helmfile app
	helmfileApp := app.New(config)

	// Run apply operation
	err := helmfileApp.Apply(config)

	// Get captured output and prepend debug info
	output := debugOutput.String() + capture.String()

	if err != nil {
		return &Result{
			Output:   output,
			ExitCode: 1,
			Error:    err,
		}, err
	}

	return &Result{
		Output:   output,
		ExitCode: 0,
		Error:    nil,
	}, nil
}

// Diff implements HelmfileExecutor.Diff using helmfile library
func (e *LibraryExecutor) Diff(ctx context.Context, opts *DiffOptions) (*Result, error) {
	// Set environment variables before running helmfile
	// This ensures helm/kubectl can access AWS credentials
	restoreEnv := setEnvironmentVariables(opts.EnvironmentVariables)
	defer restoreEnv()

	// Create output capture
	capture := NewOutputCapture()
	captureLogger := CreateCaptureLogger(capture)

	// Create config provider with capture logger
	config := &diffConfigProvider{
		baseConfigProvider: newBaseConfigProvider(opts.BaseOptions, captureLogger),
		concurrency:        opts.Concurrency,
		detailedExitcode:   opts.DetailedExitcode,
		suppressSecrets:    opts.SuppressSecrets,
		context:            opts.Context,
	}

	helmfileApp := app.New(config)

	err := helmfileApp.Diff(config)

	// Get captured output
	output := capture.String()

	if err != nil {
		return &Result{
			Output:   output,
			ExitCode: 1,
			Error:    err,
		}, err
	}

	return &Result{
		Output:   output,
		ExitCode: 0,
		Error:    nil,
	}, nil
}

// Template implements HelmfileExecutor.Template using helmfile library
func (e *LibraryExecutor) Template(ctx context.Context, opts *TemplateOptions) (*Result, error) {
	// Set environment variables before running helmfile
	// This ensures helm/kubectl can access AWS credentials
	restoreEnv := setEnvironmentVariables(opts.EnvironmentVariables)
	defer restoreEnv()

	// Create output capture
	capture := NewOutputCapture()
	captureLogger := CreateCaptureLogger(capture)

	// Create config provider with capture logger
	config := &templateConfigProvider{
		baseConfigProvider: newBaseConfigProvider(opts.BaseOptions, captureLogger),
		concurrency:        opts.Concurrency,
		includeCRDs:        opts.IncludeCRDs,
		outputDir:          opts.OutputDir,
		outputDirTemplate:  opts.OutputDirTemplate,
	}

	helmfileApp := app.New(config)

	err := helmfileApp.Template(config)

	// Get captured output
	output := capture.String()

	if err != nil {
		return &Result{
			Output:   output,
			ExitCode: 1,
			Error:    err,
		}, err
	}

	return &Result{
		Output:   output,
		ExitCode: 0,
		Error:    nil,
	}, nil
}

// Destroy implements HelmfileExecutor.Destroy using helmfile library
func (e *LibraryExecutor) Destroy(ctx context.Context, opts *DestroyOptions) (*Result, error) {
	// Set environment variables before running helmfile
	// This ensures helm/kubectl can access AWS credentials
	restoreEnv := setEnvironmentVariables(opts.EnvironmentVariables)
	defer restoreEnv()

	// Create output capture
	capture := NewOutputCapture()
	captureLogger := CreateCaptureLogger(capture)

	// Create config provider with capture logger
	config := &destroyConfigProvider{
		baseConfigProvider: newBaseConfigProvider(opts.BaseOptions, captureLogger),
		concurrency:        opts.Concurrency,
	}

	helmfileApp := app.New(config)

	err := helmfileApp.Destroy(config)

	// Get captured output
	output := capture.String()

	if err != nil {
		return &Result{
			Output:   output,
			ExitCode: 1,
			Error:    err,
		}, err
	}

	return &Result{
		Output:   output,
		ExitCode: 0,
		Error:    nil,
	}, nil
}

// Build implements HelmfileExecutor.Build using helmfile library
func (e *LibraryExecutor) Build(ctx context.Context, opts *BuildOptions) (*Result, error) {
	// Build doesn't have a direct method in app, but we can use template for validation
	// For now, return not implemented
	return nil, fmt.Errorf("Build operation not yet implemented for library executor")
}

// Version implements HelmfileExecutor.Version using helmfile library
func (e *LibraryExecutor) Version(ctx context.Context) (string, error) {
	// The library doesn't expose a version function easily
	// We can either:
	// 1. Return a hardcoded version based on the imported library version
	// 2. Call the binary version command
	// For now, return a placeholder
	return "library-mode", nil
}

// setEnvironmentVariables sets environment variables and returns a function to restore them
// This is critical for library mode because helmfile shells out to helm, which shells out to kubectl,
// which needs AWS credentials to authenticate to EKS clusters.
func setEnvironmentVariables(envVars map[string]interface{}) func() {
	// Store original values for restoration
	originalValues := make(map[string]string)
	keysToUnset := make([]string, 0)

	// CRITICAL: Ensure AWS environment variables from parent process are preserved
	// These are needed for kubectl exec authentication to EKS clusters
	// HOME is required for AWS CLI to resolve ~/.aws/config and ~/.aws/credentials
	awsEnvVars := []string{"AWS_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_CONFIG_FILE", "AWS_SHARED_CREDENTIALS_FILE", "HOME"}

	// Build a complete environment variable map that includes AWS vars from parent
	completeEnvVars := make(map[string]interface{})

	// First, copy AWS environment variables from parent process if they exist
	for _, key := range awsEnvVars {
		if val, exists := os.LookupEnv(key); exists {
			completeEnvVars[key] = val
		}
	}

	// Then, overlay with explicitly configured environment variables (these take precedence)
	for key, value := range envVars {
		completeEnvVars[key] = value
	}

	// Set each environment variable
	for key, value := range completeEnvVars {
		// Store original value if it exists
		if originalValue, exists := os.LookupEnv(key); exists {
			originalValues[key] = originalValue
		} else {
			// Mark for unsetting on cleanup
			keysToUnset = append(keysToUnset, key)
		}

		// Set the new value
		if strValue, ok := value.(string); ok {
			os.Setenv(key, strValue)
		}
	}

	// Return cleanup function
	return func() {
		// Restore original values
		for key, value := range originalValues {
			os.Setenv(key, value)
		}

		// Unset keys that didn't exist before
		for _, key := range keysToUnset {
			os.Unsetenv(key)
		}
	}
}
