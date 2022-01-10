package prelaunch_data_generator

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"text/template"
)

const (
	imageName                    = "kurtosistech/ethereum-genesis-generator:0.1.4"

	// Path on the Docker image of the script that should be run
	generateGenesisScriptFilepath = "/work/generate-genesis.sh"

	generationCommandExpectedExitCode = 0

	elDataDirname = "el"
	clDataDirname = "cl"

	// ------------------------------ Config directory paths -------------------------------
	configDataRelDirpathInSharedDir = "config"

	elConfigDataRelDirpathInSharedDir = configDataRelDirpathInSharedDir + "/" + elDataDirname
	elGenesisConfigYmlRelFilepathInSharedDir = elConfigDataRelDirpathInSharedDir + "/genesis-config.yaml"

	clConfigDataRelDirpathInSharedDir = configDataRelDirpathInSharedDir + "/" + clDataDirname
	clGenesisConfigYmlRelFilepathInSharedDir = clConfigDataRelDirpathInSharedDir + "/config.yaml"
	clMnemonicsConfigYmlRelFilepathInSharedDir = clConfigDataRelDirpathInSharedDir + "/mnemonics.yaml"

	// ------------------------------ Output directory paths -------------------------------
	outputDataRelDirpathInSharedDir = "output"

	elOutputDataRelDirpathInSharedDir = outputDataRelDirpathInSharedDir + "/" + elDataDirname
	outputGethGenesisJsonRelFilepath    = elOutputDataRelDirpathInSharedDir + "/geth.json"
	outputNethermindGenesisJsonRelFilepath    = elOutputDataRelDirpathInSharedDir + "/nethermind.json"

	clOutputDataRelDirpathInSharedDir   = outputDataRelDirpathInSharedDir + "/" + clDataDirname
	outputClGenesisConfigYmlRelFilepath = clOutputDataRelDirpathInSharedDir + "/config.yaml"
	outputClGenesisSszRelFilepath       = clOutputDataRelDirpathInSharedDir + "/genesis.ssz"
)
// We run the genesis generation as an exec command instead, so that we get immediate feedback if it fails
var entrypoingArgs = []string{
	"sleep",
	"99999",
}

type gethGenesisConfigYamlTemplateData struct {
	NetworkId string
	UnixTimestamp int64
	TotalTerminalDifficulty uint64
}
type nethermindGenesisJsonTemplateData struct {
	NetworkIDAsHex string
	// TODO add genesis timestamp here???
}
type clGenesisConfigTemplateData struct {
	NetworkId                          string
	SecondsPerSlot                     uint32
	UnixTimestamp                      int64
	Delay 							   uint64
	TotalTerminalDifficulty            uint64
	AltairForkEpoch                    uint64
	MergeForkEpoch                     uint64
	NumValidatorKeysToPreregister uint32
	PreregisteredValidatorKeysMnemonic string
}

func generateGenesisData(
	serviceCtx *services.ServiceContext,
	gethGenesisConfigYmlTemplate *template.Template,
	nethermindGenesisConfigJsonTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clMnemonicsYmlTemplate *template.Template,
	unixTimestamp int64,
	delay uint64,
	networkId string,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	totalTerminalDifficulty uint64,
	preregisteredValidatorKeysMnemonic string,
	numValidatorKeysToPreregister uint32,
) (
	resultGethGenesisJsonFilepathOnModuleContainer string,
	resultNethermindGenesisJsonFilepathOnModuleContainer string,
	resultClGenesisPaths *CLGenesisPaths,
	resultErr error,
) {
	sharedDir := serviceCtx.GetSharedDirectory()

	relDirpathsToCreate := []string{
		configDataRelDirpathInSharedDir,
		elConfigDataRelDirpathInSharedDir,
		clConfigDataRelDirpathInSharedDir,
	}
	for _, relDirpathToCreate := range relDirpathsToCreate {
		sharedPathToCreate := sharedDir.GetChildPath(relDirpathToCreate)
		absDirpathToCreateOnModuleContainer := sharedPathToCreate.GetAbsPathOnThisContainer()
		if err := os.Mkdir(absDirpathToCreateOnModuleContainer, os.ModePerm); err != nil {
			return "", "", nil, stacktrace.Propagate(err, "An error occurred creating directory at relative path '%v' inside the shared directory", absDirpathToCreateOnModuleContainer)
		}
	}


	elTemplateData := gethGenesisConfigYamlTemplateData{
		NetworkId:                   networkId,
		UnixTimestamp:               unixTimestamp,
		TotalTerminalDifficulty:     totalTerminalDifficulty,
	}
	clTemplateData := clGenesisConfigTemplateData{
		NetworkId:                          networkId,
		SecondsPerSlot:                     secondsPerSlot,
		UnixTimestamp:                      unixTimestamp,
		Delay:                              delay,
		TotalTerminalDifficulty:            totalTerminalDifficulty,
		AltairForkEpoch:                    altairForkEpoch,
		MergeForkEpoch:                     mergeForkEpoch,
		NumValidatorKeysToPreregister:      numValidatorKeysToPreregister,
		PreregisteredValidatorKeysMnemonic: preregisteredValidatorKeysMnemonic,
	}

	// Make the Geth genesis config available to the generator
	gethGenesisConfigYmlSharedPath := sharedDir.GetChildPath(elGenesisConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(gethGenesisConfigYmlTemplate, elTemplateData, gethGenesisConfigYmlSharedPath); err != nil {
		return "", "", nil, stacktrace.Propagate(err, "An error occurred filling the Geth genesis config template")
	}

	// Make the CL genesis config available to the generator
	clGenesisConfigYmlSharedPath := sharedDir.GetChildPath(clGenesisConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(clGenesisConfigYmlTemplate, clTemplateData, clGenesisConfigYmlSharedPath); err != nil {
		return "", "", nil, stacktrace.Propagate(err, "An error occurred filling the CL genesis config template")
	}

	// Make the CL mnemonics file available to the generator container
	clMnemonicsYmlSharedPath := sharedDir.GetChildPath(clMnemonicsConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(clMnemonicsYmlTemplate, clTemplateData, clMnemonicsYmlSharedPath); err != nil {
		return "", "", nil, stacktrace.Propagate(err, "An error occurred filling the CL mnemonics config YML template")
	}

	configSharedPath := sharedDir.GetChildPath(configDataRelDirpathInSharedDir)
	outputSharedPath := sharedDir.GetChildPath(outputDataRelDirpathInSharedDir)
	cmdArgs := []string{
		generateGenesisScriptFilepath,
		configSharedPath.GetAbsPathOnServiceContainer(),
		outputSharedPath.GetAbsPathOnServiceContainer(),
	}
	cmdStr := strings.Join(cmdArgs, " ")

	exitCode, output, err := serviceCtx.ExecCommand([]string{"sh", "-c", cmdStr})
	if err != nil {
		return "", "", nil, stacktrace.Propagate(err, "An error occurred executing command '%v' to generate the genesis data inside the generator container", cmdStr)
	}
	if exitCode != generationCommandExpectedExitCode {
		return "", "", nil, stacktrace.NewError(
			"Expected genesis-generating command '%v' to exit with code %v but got %v instead and the following logs:\n%v",
			cmdStr,
			generationCommandExpectedExitCode,
			exitCode,
			output,

		)
	}
	logrus.Debugf("Genesis generation output:\n%v", output)


	gethGenesisJsonFilepathOnModuleContainer := sharedDir.GetChildPath(
		outputGethGenesisJsonRelFilepath,
	).GetAbsPathOnThisContainer()
	clGenesisPaths := NewCLGenesisPaths(
		sharedDir.GetChildPath(clOutputDataRelDirpathInSharedDir).GetAbsPathOnThisContainer(),
		sharedDir.GetChildPath(outputClGenesisConfigYmlRelFilepath).GetAbsPathOnThisContainer(),
		sharedDir.GetChildPath(outputClGenesisSszRelFilepath).GetAbsPathOnThisContainer(),
	)

	networkIdAsHex, err := getNetworkIdHexSting(networkId)
	if err != nil {
		return "", "", nil, stacktrace.Propagate(
			err,
			"An error occurred rendering network ID '%v' as a hex string",
			networkId,
		)
	}
	nethermindGenesisJsonTemplateDataToUse := nethermindGenesisJsonTemplateData{
		NetworkIDAsHex: networkIdAsHex,
	}
	nethermindGenesisJsonSharedPath := sharedDir.GetChildPath(outputNethermindGenesisJsonRelFilepath)
	nethermindGenesisJsonFilepathOnModuleContainer := nethermindGenesisJsonSharedPath.GetAbsPathOnThisContainer()
	nethermindGenesisJsonFp, err := os.Create(nethermindGenesisJsonFilepathOnModuleContainer)
	if err != nil {
		return "", "", nil, stacktrace.Propagate(
			err,
			"An error occurred opening Nethermind genesis JSON file '%v' for writing",
			nethermindGenesisJsonFilepathOnModuleContainer,
		)
	}
	if err := nethermindGenesisConfigJsonTemplate.Execute(nethermindGenesisJsonFp, nethermindGenesisJsonTemplateDataToUse); err != nil {
		return "", "", nil, stacktrace.Propagate(
			err,
			"An error occurred filling the Nethermind genesis JSON template to file '%v'",
			nethermindGenesisJsonFilepathOnModuleContainer,
		)
	}

	return gethGenesisJsonFilepathOnModuleContainer, nethermindGenesisJsonFilepathOnModuleContainer, clGenesisPaths, nil
}

func getContainerConfig(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfig := services.NewContainerConfigBuilder(
		imageName,
	).WithEntrypointOverride(
		entrypoingArgs,
	).Build()

	return containerConfig, nil
}

func getNetworkIdHexSting(networkId string) (string, error) {
	uintBase := 10
	uintBits := 64
	networkIdUint64, err := strconv.ParseUint(networkId, uintBase, uintBits)
	if err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred parsing network ID string '%v' to uint with base %v and %v bits",
			networkId,
			uintBase,
			uintBits,
		)
	}
	return fmt.Sprintf("0x%x", networkIdUint64), nil
}
