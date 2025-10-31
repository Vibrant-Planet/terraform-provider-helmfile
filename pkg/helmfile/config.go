package helmfile

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"go.uber.org/zap"
)

type ProviderInstance struct {
	MaxDiffOutputLen int
	Executor         HelmfileExecutor
}

func New(d *schema.ResourceData) *ProviderInstance {
	// Always use library executor
	logger, err := zap.NewDevelopment()
	if err != nil {
		// This should rarely fail, but log it if it does
		logf("Failed to create logger for library executor: %v", err)
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}

	return &ProviderInstance{
		MaxDiffOutputLen: d.Get(KeyMaxDiffOutputLen).(int),
		Executor:         NewLibraryExecutor(logger.Sugar()),
	}
}
