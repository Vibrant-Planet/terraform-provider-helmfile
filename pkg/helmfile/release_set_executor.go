package helmfile

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

// prepareHelmfileFile writes the helmfile content to a temporary file and returns the path
func prepareHelmfileFile(fs *ReleaseSet) (string, error) {
	if fs.WorkingDirectory != "" {
		if err := os.MkdirAll(fs.WorkingDirectory, 0755); err != nil {
			return "", fmt.Errorf("creating working directory %q: %w", fs.WorkingDirectory, err)
		}
	}

	bs := []byte(fs.Content)
	first := sha256.New()
	first.Write(bs)

	// Use .yaml.gotmpl extension when go template rendering is enabled
	extension := ".yaml"
	if fs.EnableGoTemplate {
		extension = ".yaml.gotmpl"
	}
	tmpFile := fmt.Sprintf("helmfile-%x%s", first.Sum(nil), extension)
	tmpFilePath := filepath.Join(fs.WorkingDirectory, tmpFile)

	if err := ioutil.WriteFile(tmpFilePath, bs, 0700); err != nil {
		return "", err
	}

	// Also write values files
	for _, vs := range fs.Values {
		js := []byte(fmt.Sprintf("%s", vs))

		valuesHash := sha256.New()
		valuesHash.Write(js)

		relpath := filepath.Join(
			fs.WorkingDirectory,
			fmt.Sprintf("temp.values-%x.yaml", valuesHash.Sum(nil)),
		)

		abspath, err := filepath.Abs(relpath)
		if err != nil {
			return "", xerrors.Errorf("getting absolute path to %s: %w", abspath, err)
		}

		if err := ioutil.WriteFile(abspath, js, 0700); err != nil {
			return "", err
		}
	}

	return tmpFilePath, nil
}

// buildBaseOptions creates BaseOptions from ReleaseSet
func buildBaseOptions(fs *ReleaseSet, tmpFile string) *BaseOptions {
	kubeconfig, _ := getKubeconfig(fs)
	kubeconfigPath := ""
	if kubeconfig != nil {
		kubeconfigPath = *kubeconfig
	}

	return &BaseOptions{
		FileOrDir:            tmpFile,
		WorkingDirectory:     fs.WorkingDirectory,
		Kubeconfig:           kubeconfigPath,
		Environment:          fs.Environment,
		Selector:             fs.Selector,
		Selectors:            fs.Selectors,
		ValuesFiles:          fs.ValuesFiles,
		Values:               fs.Values,
		EnvironmentVariables: fs.EnvironmentVariables,
		HelmBinary:           fs.HelmBin,
		HelmfileBinary:       fs.Bin,
		EnableGoTemplate:     fs.EnableGoTemplate,
	}
}

// buildApplyOptions creates ApplyOptions from ReleaseSet
func buildApplyOptions(fs *ReleaseSet, tmpFile string) *ApplyOptions {
	return &ApplyOptions{
		BaseOptions:     *buildBaseOptions(fs, tmpFile),
		Concurrency:     fs.Concurrency,
		ReleasesValues:  fs.ReleasesValues,
		SuppressSecrets: true,
	}
}

// buildDiffOptions creates DiffOptions from ReleaseSet
func buildDiffOptions(fs *ReleaseSet, tmpFile string, maxLen int) *DiffOptions {
	return &DiffOptions{
		BaseOptions:      *buildBaseOptions(fs, tmpFile),
		Concurrency:      fs.Concurrency,
		ReleasesValues:   fs.ReleasesValues,
		DetailedExitcode: true,
		SuppressSecrets:  true,
		Context:          3,
		MaxDiffOutputLen: maxLen,
	}
}

// buildTemplateOptions creates TemplateOptions from ReleaseSet
func buildTemplateOptions(fs *ReleaseSet, tmpFile string) *TemplateOptions {
	return &TemplateOptions{
		BaseOptions: *buildBaseOptions(fs, tmpFile),
		Concurrency: fs.Concurrency,
		IncludeCRDs: true,
	}
}

// buildDestroyOptions creates DestroyOptions from ReleaseSet
func buildDestroyOptions(fs *ReleaseSet, tmpFile string) *DestroyOptions {
	return &DestroyOptions{
		BaseOptions: *buildBaseOptions(fs, tmpFile),
		Concurrency: fs.Concurrency,
	}
}
