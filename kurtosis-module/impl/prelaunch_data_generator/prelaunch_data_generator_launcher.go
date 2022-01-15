package prelaunch_data_generator

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)


const (
	image = "skylenet/ethereum-genesis-generator:latest"

	serviceId services.ServiceID = "prelaunch-data-generator"
)
// We use Docker exec commands to run the commands we need, so we override the default
var entrypointArgs = []string{
	"sleep",
	"999999",
}

type PrelaunchData struct {
	GethELGenesisJsonFilepathOnModuleContainer string
	NethermindGenesisJsonFilepathOnModuleContainer string
	CLGenesisPaths *cl.CLGenesisPaths
	KeystoresGenerationResult *cl.GenerateKeystoresResult
}

func LaunchPrelaunchDataGenerator(
	enclaveCtx *enclaves.EnclaveContext,
	networkId string,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
) (
	*PrelaunchDataGeneratorContext,
	error,
) {
	serviceCtx, err := enclaveCtx.AddService(serviceId, getContainerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the prelaunch data generator container with service ID '%v'", serviceId)
	}

	result := newPrelaunchDataGeneratorContext(
		serviceCtx,
		networkId,
		depositContractAddress,
		totalTerminalDifficulty,
	)
	return result, nil
}

func getContainerConfig(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfig := services.NewContainerConfigBuilder(
		image,
	).WithEntrypointOverride(
		entrypointArgs,
	).Build()

	return containerConfig, nil
}
