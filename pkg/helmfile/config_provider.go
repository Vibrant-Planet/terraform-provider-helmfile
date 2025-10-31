package helmfile

import (
	"go.uber.org/zap"
)

// baseConfigProvider implements the base app.ConfigProvider interface
// This is the foundation for all operation-specific config providers
type baseConfigProvider struct {
	fileOrDir            string
	kubeContext          string
	namespace            string
	helmBinary           string
	environment          string
	selector             map[string]interface{}
	selectors            []interface{}
	valuesFiles          []interface{}
	values               []interface{}
	environmentVariables map[string]interface{}
	kubeconfig           string
	logger               *zap.SugaredLogger
}

func newBaseConfigProvider(opts BaseOptions, logger *zap.SugaredLogger) *baseConfigProvider {
	return &baseConfigProvider{
		fileOrDir:            opts.FileOrDir,
		kubeContext:          opts.KubeContext,
		namespace:            opts.Namespace,
		helmBinary:           opts.HelmBinary,
		environment:          opts.Environment,
		selector:             opts.Selector,
		selectors:            opts.Selectors,
		valuesFiles:          opts.ValuesFiles,
		values:               opts.Values,
		environmentVariables: opts.EnvironmentVariables,
		kubeconfig:           opts.Kubeconfig,
		logger:               logger,
	}
}

// Implement app.ConfigProvider interface
func (c *baseConfigProvider) Args() string                       { return "" }
func (c *baseConfigProvider) ConfigFile() string                 { return "" }
func (c *baseConfigProvider) HelmBinary() string                 { return c.helmBinary }
func (c *baseConfigProvider) KustomizeBinary() string            { return "" }
func (c *baseConfigProvider) EnableLiveOutput() bool             { return true }
func (c *baseConfigProvider) FileOrDir() string                  { return c.fileOrDir }
func (c *baseConfigProvider) KubeContext() string                { return c.kubeContext }
func (c *baseConfigProvider) Namespace() string                  { return c.namespace }
func (c *baseConfigProvider) Chart() string                      { return "" }
func (c *baseConfigProvider) Selectors() []string                { return convertSelectorsToStrings(c.selectors) }
func (c *baseConfigProvider) StateValuesSet() map[string]any     { return nil }
func (c *baseConfigProvider) StateValuesFiles() []string         { return convertToStringSlice(c.valuesFiles) }
func (c *baseConfigProvider) Environment() string                { return c.environment }
func (c *baseConfigProvider) Logger() *zap.SugaredLogger         { return c.logger }
func (c *baseConfigProvider) Validate() bool                     { return false }
func (c *baseConfigProvider) EmbedValues() bool                  { return false }
func (c *baseConfigProvider) IncludeTransitiveNeeds() bool       { return false }
func (c *baseConfigProvider) IncludeNeeds() bool                 { return false }
func (c *baseConfigProvider) Interactive() bool                  { return false }
func (c *baseConfigProvider) SkipDeps() bool                     { return false }
func (c *baseConfigProvider) IncludeCRDs() bool                  { return true }
func (c *baseConfigProvider) DisableForceUpdate() bool           { return false }
func (c *baseConfigProvider) Env() string                        { return c.environment }
func (c *baseConfigProvider) Kubeconfig() string                 { return c.kubeconfig }
func (c *baseConfigProvider) StripArgsValuesOnExitError() bool   { return false }

// applyConfigProvider implements app.ApplyConfigProvider
type applyConfigProvider struct {
	*baseConfigProvider
	concurrency       int
	suppressSecrets   bool
	skipDiffOnInstall bool
}

// Implement additional methods for ApplyConfigProvider
func (c *applyConfigProvider) Concurrency() int          { return c.concurrency }
func (c *applyConfigProvider) Values() []string          { return convertToStringSlice(c.values) }
func (c *applyConfigProvider) Set() []string             { return nil }
func (c *applyConfigProvider) OutputDir() string         { return "" }
func (c *applyConfigProvider) OutputDirTemplate() string { return "" }
func (c *applyConfigProvider) OutputFileTemplate() string{ return "" }
func (c *applyConfigProvider) ShowOnly() []string        { return nil }
func (c *applyConfigProvider) KubeVersion() string       { return "" }
func (c *applyConfigProvider) NoHooks() bool             { return false }
func (c *applyConfigProvider) SkipTests() bool           { return false }
func (c *applyConfigProvider) SkipCleanup() bool         { return false }
func (c *applyConfigProvider) SkipNeeds() bool           { return false }
func (c *applyConfigProvider) PostRenderer() string      { return "" }
func (c *applyConfigProvider) PostRendererArgs() []string{ return nil }
func (c *applyConfigProvider) Wait() bool                { return false }
func (c *applyConfigProvider) WaitForJobs() bool         { return false }
func (c *applyConfigProvider) SuppressSecrets() bool     { return c.suppressSecrets }
func (c *applyConfigProvider) SuppressDiff() bool        { return false }
func (c *applyConfigProvider) Suppress() []string        { return nil }
func (c *applyConfigProvider) ShowSecrets() bool         { return !c.suppressSecrets }
func (c *applyConfigProvider) Context() int              { return 3 }
func (c *applyConfigProvider) DiffOutput() string        { return "" }
func (c *applyConfigProvider) DetailedExitcode() bool    { return false }
func (c *applyConfigProvider) Color() bool               { return false }
func (c *applyConfigProvider) NoColor() bool             { return true }
func (c *applyConfigProvider) Cascade() string           { return "" }
func (c *applyConfigProvider) DiffArgs() string          { return "" }
func (c *applyConfigProvider) IncludeTests() bool        { return false }
func (c *applyConfigProvider) ResetValues() bool         { return false }
func (c *applyConfigProvider) RetainValuesFiles() bool   { return false }
func (c *applyConfigProvider) ReuseValues() bool         { return false }
func (c *applyConfigProvider) SkipCRDs() bool            { return false }
func (c *applyConfigProvider) SkipDiffOnInstall() bool   { return c.skipDiffOnInstall }
func (c *applyConfigProvider) StripTrailingCR() bool     { return false }
func (c *applyConfigProvider) SuppressOutputLineRegex() []string { return nil }
func (c *applyConfigProvider) SyncArgs() string          { return "" }

// diffConfigProvider implements app.DiffConfigProvider
type diffConfigProvider struct {
	*baseConfigProvider
	concurrency      int
	detailedExitcode bool
	suppressSecrets  bool
	context          int
}

func (c *diffConfigProvider) Concurrency() int           { return c.concurrency }
func (c *diffConfigProvider) Values() []string           { return convertToStringSlice(c.values) }
func (c *diffConfigProvider) Set() []string              { return nil }
func (c *diffConfigProvider) DetailedExitcode() bool     { return c.detailedExitcode }
func (c *diffConfigProvider) SuppressSecrets() bool      { return c.suppressSecrets }
func (c *diffConfigProvider) Context() int               { return c.context }
func (c *diffConfigProvider) Suppress() []string         { return nil }
func (c *diffConfigProvider) ShowSecrets() bool          { return !c.suppressSecrets }
func (c *diffConfigProvider) Color() bool                { return false }
func (c *diffConfigProvider) NoColor() bool              { return true }
func (c *diffConfigProvider) OutputDir() string          { return "" }
func (c *diffConfigProvider) OutputDirTemplate() string  { return "" }
func (c *diffConfigProvider) OutputFileTemplate() string { return "" }
func (c *diffConfigProvider) ShowOnly() []string         { return nil }
func (c *diffConfigProvider) KubeVersion() string        { return "" }
func (c *diffConfigProvider) NoHooks() bool              { return false }
func (c *diffConfigProvider) SkipTests() bool            { return false }
func (c *diffConfigProvider) SkipCleanup() bool          { return false }
func (c *diffConfigProvider) SkipNeeds() bool            { return false }
func (c *diffConfigProvider) PostRenderer() string       { return "" }
func (c *diffConfigProvider) PostRendererArgs() []string { return nil }
func (c *diffConfigProvider) DiffArgs() string           { return "" }
func (c *diffConfigProvider) DiffOutput() string         { return "" }
func (c *diffConfigProvider) IncludeTests() bool         { return false }
func (c *diffConfigProvider) ResetValues() bool          { return false }
func (c *diffConfigProvider) ReuseValues() bool          { return false }
func (c *diffConfigProvider) SkipCRDs() bool             { return false }
func (c *diffConfigProvider) SkipDiffOnInstall() bool    { return false }
func (c *diffConfigProvider) StripTrailingCR() bool      { return false }
func (c *diffConfigProvider) SuppressDiff() bool         { return false }
func (c *diffConfigProvider) SuppressOutputLineRegex() []string { return nil }

// templateConfigProvider implements app.TemplateConfigProvider
type templateConfigProvider struct {
	*baseConfigProvider
	concurrency       int
	includeCRDs       bool
	outputDir         string
	outputDirTemplate string
}

func (c *templateConfigProvider) Concurrency() int            { return c.concurrency }
func (c *templateConfigProvider) Values() []string            { return convertToStringSlice(c.values) }
func (c *templateConfigProvider) Set() []string               { return nil }
func (c *templateConfigProvider) OutputDir() string           { return c.outputDir }
func (c *templateConfigProvider) OutputDirTemplate() string   { return c.outputDirTemplate }
func (c *templateConfigProvider) OutputFileTemplate() string  { return "" }
func (c *templateConfigProvider) ShowOnly() []string          { return nil }
func (c *templateConfigProvider) KubeVersion() string         { return "" }
func (c *templateConfigProvider) NoHooks() bool               { return false }
func (c *templateConfigProvider) SkipTests() bool             { return false }
func (c *templateConfigProvider) SkipCleanup() bool           { return false }
func (c *templateConfigProvider) SkipNeeds() bool             { return false }
func (c *templateConfigProvider) PostRenderer() string        { return "" }
func (c *templateConfigProvider) PostRendererArgs() []string  { return nil }

// Override IncludeCRDs for template
func (c *templateConfigProvider) IncludeCRDs() bool { return c.includeCRDs }

// destroyConfigProvider implements app.DestroyConfigProvider
type destroyConfigProvider struct {
	*baseConfigProvider
	concurrency int
}

func (c *destroyConfigProvider) Concurrency() int  { return c.concurrency }
func (c *destroyConfigProvider) Cascade() string    { return "" }
func (c *destroyConfigProvider) DeleteTimeout() int { return 0 }
func (c *destroyConfigProvider) DeleteWait() bool   { return false }
func (c *destroyConfigProvider) SkipCharts() bool   { return false }

// Helper functions
func convertToStringSlice(items []interface{}) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func convertSelectorsToStrings(selectors []interface{}) []string {
	result := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		if str, ok := selector.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
