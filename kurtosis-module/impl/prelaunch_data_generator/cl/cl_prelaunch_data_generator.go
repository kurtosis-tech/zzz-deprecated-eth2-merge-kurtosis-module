package cl

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"text/template"
)

func GenerateCLPrelaunchData(
	genesisGenerationConfigYmlTemplate *template.Template,
	genesisGenerationMnemonicsYmlTemplate *template.Template,
	serviceCtx *services.ServiceContext,
	genesisUnixTimestamp uint64,
	networkId string,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	preregisteredValidatorKeysMnemonic string,
	numValidatorKeysToPreregister uint32,
	stakingContractSeedMnemonic string,
	numValidatorNodes uint32,
) (
	*CLPrelaunchData,
	error,
) {
	genesisPaths, err := generateClGenesisData(
		genesisGenerationConfigYmlTemplate,
		genesisGenerationMnemonicsYmlTemplate,
		serviceCtx,
		genesisUnixTimestamp,
		networkId,
		depositContractAddress,
		totalTerminalDifficulty,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		preregisteredValidatorKeysMnemonic,
		numValidatorKeysToPreregister,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the CL genesis data")
	}

	keystoreGenerationResults, err := generateClValidatorKeystores(
		serviceCtx,
		stakingContractSeedMnemonic,
		numValidatorKeysToPreregister,
		numValidatorNodes,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the CL validator keystores")
	}

	result := newCLPrelaunchData(genesisPaths, keystoreGenerationResults)
	return result, nil
}