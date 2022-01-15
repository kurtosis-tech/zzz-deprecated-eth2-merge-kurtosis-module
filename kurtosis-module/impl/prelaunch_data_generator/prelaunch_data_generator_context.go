package prelaunch_data_generator

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
)

type ELPrelaunchData struct {
	GethELGenesisJsonFilepathOnModuleContainer string
	NethermindGenesisJsonFilepathOnModuleContainer string
}

type CLPrelaunchData struct {
	CLGenesisPaths *CLGenesisPaths
	KeystoresGenerationResult *cl.GenerateKeystoresResult
}

type PrelaunchDataGeneratorContext struct {
	serviceCtx *services.ServiceContext
}

func (ctx *PrelaunchDataGeneratorContext) GenerateELData() (*ELPrelaunchData, error) {

}

func (ctx *PrelaunchDataGeneratorContext) GenerateCLData() (*CLPrelaunchData, error) {

}
