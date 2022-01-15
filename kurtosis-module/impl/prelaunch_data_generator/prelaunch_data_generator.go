package prelaunch_data_generator

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"text/template"
)

const (
	serviceId services.ServiceID = "eth-genesis-generator"

	containerStopTimeoutSeconds = 3
)

type PrelaunchData struct {
	GethELGenesisJsonFilepathOnModuleContainer string
	NethermindGenesisJsonFilepathOnModuleContainer string
	CLGenesisPaths *cl.CLGenesisPaths
	KeystoresGenerationResult *cl.GenerateKeystoresResult
}

func GeneratePrelaunchData(
	enclaveCtx *enclaves.EnclaveContext,
	gethGenesisConfigYmlTemplate *template.Template,
	nethermindGenesisConfigJsonTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clGenesisMnemonicsYmlTemplate *template.Template,
	validatorsMnemonic string,
	numValidatorKeysToPreregister uint32,
	numClNodesToStart uint32,
	genesisUnixTimestamp int64,
	genesisDelay uint64,
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

	logrus.Info("Generating genesis data...")
	gethGenesisJsonFilepath, nethermindGenesisJsonFilepath, clGenesisPaths, err := generateGenesisData(
		serviceCtx,
		gethGenesisConfigYmlTemplate,
		nethermindGenesisConfigJsonTemplate,
		clGenesisConfigYmlTemplate,
		clGenesisMnemonicsYmlTemplate,
		genesisUnixTimestamp,
		genesisDelay,
		networkId,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		totalTerminalDifficulty,
		stakingContractSeedMnemonic,
		numValidatorKeysToPreregister,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating genesis data")
	}
	logrus.Info("Successfully generated genesis data")

	logrus.Info("Generating validator keystores for nodes...")
	generateKeystoresResult, err := cl.generateClValidatorKeystores(
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
	logrus.Info("Successfully generated validator keystores for nodes")

	if err := enclaveCtx.RemoveService(serviceId, containerStopTimeoutSeconds); err != nil {
		logrus.Errorf(
			"An error occurred stopping the genesis generation service with ID '%v' and timeout '%vs'; you'll need to stop it manually",
			serviceId,
			containerStopTimeoutSeconds,
		)
	}
	
	result := &PrelaunchData{
		GethELGenesisJsonFilepathOnModuleContainer:     gethGenesisJsonFilepath,
		NethermindGenesisJsonFilepathOnModuleContainer: nethermindGenesisJsonFilepath,
		CLGenesisPaths:            clGenesisPaths,
		KeystoresGenerationResult: generateKeystoresResult,
	}

	return result, nil
}
