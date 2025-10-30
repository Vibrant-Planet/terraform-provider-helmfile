package helmfile

import (
	"context"
	"fmt"

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
	// Create output capture
	capture := NewOutputCapture()
	captureLogger := CreateCaptureLogger(capture)

	// Create config provider with capture logger
	config := &applyConfigProvider{
		baseConfigProvider: newBaseConfigProvider(opts.BaseOptions, captureLogger),
		concurrency:        opts.Concurrency,
		suppressSecrets:    opts.SuppressSecrets,
	}

	// Initialize helmfile app
	helmfileApp := app.New(config)

	// Run apply operation
	err := helmfileApp.Apply(config)

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

// Diff implements HelmfileExecutor.Diff using helmfile library
func (e *LibraryExecutor) Diff(ctx context.Context, opts *DiffOptions) (*Result, error) {
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
