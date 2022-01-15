package el

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	// The prefix that the directory for containing information about this EL genesis generation run will have
	//  inside the shared directory
	elGenesisGenerationInstanceSharedDirpathPrefix = "el-genesis-"

	configDirname                      = "config"
	gethGenesisGenerationConfigFilename = "geth-genesis-generation-config.yml"

	outputDirname = "output"
	chainspecJsonFilename = "chainspec.json"
	gethGenesisJsonFilename = "geth.json"
	nethermindGenesisJsonFilename = "nethermind.json"

	// Generation constants
	generateChainspecScriptFilepath = "/apps/el-gen/genesis_chainspec.py"
	generateGethGenesisScriptFilepath = "/apps/el-gen/genesis_geth.py"
)

type nethermindGenesisJsonTemplateData struct {
	NetworkIDAsHex string
	// TODO add genesis timestamp here???
}

func generateElGenesisData(
	serviceCtx *services.ServiceContext,
	gethGenesisConfigYmlTemplate *template.Template,
	nethermindGenesisConfigJsonTemplate *template.Template,
	genesisUnixTimestamp int64,
	networkId string,
	totalTerminalDifficulty uint64,
) (
	*ELGenesisPaths,
	error,
) {
	sharedDir := serviceCtx.GetSharedDirectory()
	generationInstanceSharedDir := sharedDir.GetChildPath(fmt.Sprintf(
		"%v%v",
		elGenesisGenerationInstanceSharedDirpathPrefix,
		time.Now().Unix(),
	))
	configSharedDir := generationInstanceSharedDir.GetChildPath(configDirname)
	outputSharedDir := generationInstanceSharedDir.GetChildPath(outputDirname)

	allSharedDirsToCreate := []*services.SharedPath{
		generationInstanceSharedDir,
		configSharedDir,
		outputSharedDir,
	}
	for _, sharedDirToCreate := range allSharedDirsToCreate {
		toCreateDirpathOnModuleContainer := sharedDirToCreate.GetAbsPathOnThisContainer()
		if err := os.Mkdir(toCreateDirpathOnModuleContainer, os.ModePerm); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred creating directory '%v'", toCreateDirpathOnModuleContainer)
		}
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
	clGenesisPaths := cl.NewCLGenesisPaths(
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

func generateGethGenesis() {

	elTemplateData := chainspecAndGethGenesisGenerationConfigTemplateData{
		NetworkId:                   networkId,
		UnixTimestamp:               unixTimestamp,
		TotalTerminalDifficulty:     totalTerminalDifficulty,
	}

	// Make the Geth genesis config available to the generator
	gethGenesisConfigYmlSharedPath := sharedDir.GetChildPath(elGenesisConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(gethGenesisConfigYmlTemplate, elTemplateData, gethGenesisConfigYmlSharedPath); err != nil {
		return "", "", nil, stacktrace.Propagate(err, "An error occurred filling the Geth genesis config template")
	}
}

func generateNethermindGenesis() {

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
