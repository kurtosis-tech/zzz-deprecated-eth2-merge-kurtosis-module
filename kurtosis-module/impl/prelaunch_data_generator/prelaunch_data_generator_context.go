package prelaunch_data_generator

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"text/template"
)

type PrelaunchDataGeneratorContext struct {
	serviceCtx *services.ServiceContext
	networkId string
	depositContractAddress string
	totalTerminalDifficulty uint64
	preregisteredValidatorKeysMnemonic string
}

func newPrelaunchDataGeneratorContext(serviceCtx *services.ServiceContext, networkId string, depositContractAddress string, totalTerminalDifficulty uint64, preregisteredValidatorKeysMnemonic string) *PrelaunchDataGeneratorContext {
	return &PrelaunchDataGeneratorContext{serviceCtx: serviceCtx, networkId: networkId, depositContractAddress: depositContractAddress, totalTerminalDifficulty: totalTerminalDifficulty, preregisteredValidatorKeysMnemonic: preregisteredValidatorKeysMnemonic}
}

func (ctx *PrelaunchDataGeneratorContext) GenerateELGenesisData(
	genesisGenerationConfigTemplate *template.Template,
	genesisUnixTimestamp uint64,
) (*el_genesis.ELGenesisData, error) {
	result, err := el_genesis.GenerateELGenesisData(
		ctx.serviceCtx,
		genesisGenerationConfigTemplate,
		genesisUnixTimestamp,
		ctx.networkId,
		ctx.depositContractAddress,
		ctx.totalTerminalDifficulty,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the EL genesis data")
	}
	return result, nil
}

func (ctx *PrelaunchDataGeneratorContext) GenerateCLValidatorData(
	numValidatorNodes uint32,
	numValidatorsPerNode uint32,
) (*cl_validator_keystores.GenerateKeystoresResult, error) {
	result, err := cl_validator_keystores.GenerateCLValidatorKeystores(
		ctx.serviceCtx,
		ctx.preregisteredValidatorKeysMnemonic,
		numValidatorNodes,
		numValidatorsPerNode,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the CL client validator keystores")
	}
	return result, nil
}

func (ctx *PrelaunchDataGeneratorContext) GenerateCLGenesisData(
	genesisGenerationConfigYmlTemplate *template.Template,
	genesisGenerationMnemonicsYmlTemplate *template.Template,
	jwtSecretFilepathOnModuleContainer string,
	genesisUnixTimestamp uint64,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	numValidatorNodes uint32,
	numValidatorsPerNode uint32,
) (
	*cl_genesis.CLGenesisData,
	error,
) {
	numValidatorKeysToPreregister := numValidatorNodes * numValidatorsPerNode
	result, err := cl_genesis.GenerateCLGenesisData(
		genesisGenerationConfigYmlTemplate,
		genesisGenerationMnemonicsYmlTemplate,
		jwtSecretFilepathOnModuleContainer,
		ctx.serviceCtx,
		genesisUnixTimestamp,
		ctx.networkId,
		ctx.depositContractAddress,
		ctx.totalTerminalDifficulty,
		secondsPerSlot,
		altairForkEpoch,
		mergeForkEpoch,
		ctx.preregisteredValidatorKeysMnemonic,
		numValidatorKeysToPreregister,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred generating the CL prelaunch data")
	}
	return result, nil
}
