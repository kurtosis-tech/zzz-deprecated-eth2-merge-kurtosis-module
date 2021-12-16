package prelaunch_data_generator

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"text/template"
)

const (
	imageName                    = "kurtosistech/ethereum-genesis-generator"
	serviceId services.ServiceID = "eth-genesis-generator"

	containerStopTimeoutSeconds = 3
)

type PrelaunchData struct {
	GethELGenesisJsonFilepathOnModuleContainer string
	CLGenesisPaths *CLGenesisPaths
	KeystoresGenerationResult *GenerateKeystoresResult
}

func GeneratePrelaunchData(
	enclaveCtx *enclaves.EnclaveContext,
	elGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clGenesisMnemonicsYmlTemplate *template.Template,
	validatorsMnemonic string,
	numValidatorKeysToPreregister uint32,
	numClNodesToStart uint32,
	genesisUnixTimestamp int64,
	networkId string,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	totalTerminalDifficulty uint64,
	stakingContractSeedMnemonic string,
) (
	*PrelaunchData,
	error,
) {
	serviceCtx, err := enclaveCtx.AddService(serviceId, getContainerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis-generating container with service ID '%v'", serviceId)
	}

	gethGenesisJsonFilepath, clGenesisPaths, err := generateGenesisData(
		serviceCtx,
		elGenesisConfigYmlTemplate,
		clGenesisConfigYmlTemplate,
		clGenesisMnemonicsYmlTemplate,
		genesisUnixTimestamp,
		networkId,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		totalTerminalDifficulty,
		stakingContractSeedMnemonic,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating genesis data")
	}

	generateKeystoresResult, err := generateKeystores(
		serviceCtx,
		validatorsMnemonic,
		numValidatorKeysToPreregister,
		numClNodesToStart,
	)
	if err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred allocating %v validator keys to keystores in %v CL client nodes",
			numValidatorKeysToPreregister,
			numClNodesToStart,
		)
	}

	if err := enclaveCtx.RemoveService(serviceId, containerStopTimeoutSeconds); err != nil {
		logrus.Errorf(
			"An error occurred stopping the genesis generation service with ID '%v' and timeout '%vs'; you'll need to stop it manually",
			serviceId,
			containerStopTimeoutSeconds,
		)
	}
	
	result := &PrelaunchData{
		GethELGenesisJsonFilepathOnModuleContainer: gethGenesisJsonFilepath,
		CLGenesisPaths:            clGenesisPaths,
		KeystoresGenerationResult: generateKeystoresResult,
	}

	return result, nil
}
