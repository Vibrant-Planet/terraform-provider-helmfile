package helmfile

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"go.uber.org/zap"
)

type ProviderInstance struct {
	MaxDiffOutputLen int
	UseLibrary       bool
	Executor         HelmfileExecutor
}

func New(d *schema.ResourceData) *ProviderInstance {
	useLibrary := d.Get(KeyUseLibrary).(bool)

	var executor HelmfileExecutor
	if useLibrary {
		// Create zap logger for library executor
		logger, err := zap.NewDevelopment()
		if err != nil {
			// Fall back to binary executor if logger creation fails
			logf("Failed to create logger for library executor, falling back to binary: %v", err)
			executor = NewBinaryExecutor()
		} else {
			executor = NewLibraryExecutor(logger.Sugar())
		}
	} else {
		executor = NewBinaryExecutor()
	}

	return &ProviderInstance{
		MaxDiffOutputLen: d.Get(KeyMaxDiffOutputLen).(int),
		UseLibrary:       useLibrary,
		Executor:         executor,
	}
}
