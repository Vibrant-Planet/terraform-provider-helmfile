package helmfile

import (
	"testing"

	"github.com/helmfile/helmfile/pkg/app"
	"go.uber.org/zap"
)

// Compile-time interface assertions to ensure our config providers
// satisfy the helmfile library's interfaces.
var (
	_ app.ConfigProvider          = (*baseConfigProvider)(nil)
	_ app.ApplyConfigProvider     = (*applyConfigProvider)(nil)
	_ app.DiffConfigProvider      = (*diffConfigProvider)(nil)
	_ app.DestroyConfigProvider   = (*destroyConfigProvider)(nil)
	_ app.TemplateConfigProvider  = (*templateConfigProvider)(nil)
)

func TestConfigProviderInterfaces(t *testing.T) {
	logger := zap.NewNop().Sugar()

	base := newBaseConfigProvider(BaseOptions{
		FileOrDir:   "/tmp/helmfile.yaml",
		HelmBinary:  "helm",
		Environment: "default",
	}, logger)

	t.Run("baseConfigProvider satisfies ConfigProvider", func(t *testing.T) {
		if base.HelmBinary() != "helm" {
			t.Errorf("expected helm, got %s", base.HelmBinary())
		}
		if base.FileOrDir() != "/tmp/helmfile.yaml" {
			t.Errorf("expected /tmp/helmfile.yaml, got %s", base.FileOrDir())
		}
		if base.Env() != "default" {
			t.Errorf("expected default, got %s", base.Env())
		}
		// New v1.x methods should return safe defaults
		if base.EnforcePluginVerification() {
			t.Error("expected EnforcePluginVerification to be false")
		}
		if base.HelmOCIPlainHTTP() {
			t.Error("expected HelmOCIPlainHTTP to be false")
		}
		if base.SkipRefresh() {
			t.Error("expected SkipRefresh to be false")
		}
		if base.SequentialHelmfiles() {
			t.Error("expected SequentialHelmfiles to be false")
		}
	})

	t.Run("applyConfigProvider satisfies ApplyConfigProvider", func(t *testing.T) {
		cfg := &applyConfigProvider{
			baseConfigProvider: base,
			concurrency:        2,
			suppressSecrets:    true,
			skipDiffOnInstall:  true,
		}
		if cfg.Concurrency() != 2 {
			t.Errorf("expected 2, got %d", cfg.Concurrency())
		}
		if !cfg.SuppressSecrets() {
			t.Error("expected SuppressSecrets to be true")
		}
		if !cfg.SkipDiffOnInstall() {
			t.Error("expected SkipDiffOnInstall to be true")
		}
		// New v1.x methods
		if cfg.SkipSchemaValidation() {
			t.Error("expected SkipSchemaValidation to be false")
		}
		if cfg.TakeOwnership() {
			t.Error("expected TakeOwnership to be false")
		}
		if cfg.TrackMode() != "" {
			t.Errorf("expected empty TrackMode, got %s", cfg.TrackMode())
		}
	})

	t.Run("diffConfigProvider satisfies DiffConfigProvider", func(t *testing.T) {
		cfg := &diffConfigProvider{
			baseConfigProvider: base,
			detailedExitcode:   true,
			context:            5,
		}
		if !cfg.DetailedExitcode() {
			t.Error("expected DetailedExitcode to be true")
		}
		if cfg.Context() != 5 {
			t.Errorf("expected 5, got %d", cfg.Context())
		}
		if cfg.SkipSchemaValidation() {
			t.Error("expected SkipSchemaValidation to be false")
		}
		if cfg.TakeOwnership() {
			t.Error("expected TakeOwnership to be false")
		}
	})

	t.Run("destroyConfigProvider satisfies DestroyConfigProvider", func(t *testing.T) {
		cfg := &destroyConfigProvider{
			baseConfigProvider: base,
			concurrency:        3,
		}
		if cfg.Concurrency() != 3 {
			t.Errorf("expected 3, got %d", cfg.Concurrency())
		}
		if cfg.Args() != "" {
			t.Errorf("expected empty Args, got %s", cfg.Args())
		}
	})

	t.Run("templateConfigProvider satisfies TemplateConfigProvider", func(t *testing.T) {
		cfg := &templateConfigProvider{
			baseConfigProvider: base,
			includeCRDs:        true,
			outputDir:          "/tmp/out",
		}
		if !cfg.IncludeCRDs() {
			t.Error("expected IncludeCRDs to be true")
		}
		if cfg.OutputDir() != "/tmp/out" {
			t.Errorf("expected /tmp/out, got %s", cfg.OutputDir())
		}
		if cfg.SkipSchemaValidation() {
			t.Error("expected SkipSchemaValidation to be false")
		}
	})
}
