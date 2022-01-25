package el_genesis

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	// The prefix that the directory for containing information about this EL genesis generation run will have
	//  inside the shared directory
	elGenesisGenerationInstanceSharedDirpathPrefix = "el-genesis-"

	configDirname                      = "config"
	genesisConfigFilename  = "genesis-config.yaml"

	outputDirname = "output"

	gethGenesisFilename = "geth.json"
	nethermindGenesisFilename = "nethermind.json"
	besuGenesisFilename = "besu.json"

	successfulExecCmdExitCode = 0
)

type genesisGenerationConfigTemplateData struct {
	NetworkId string
	DepositContractAddress string
	UnixTimestamp uint64
	TotalTerminalDifficulty uint64
}

type genesisGenerationCmd func(genesisConfigPath *services.SharedPath)[]string

// Mapping of output genesis filename -> generator to create the file
var allGenesisGenerationCmds = map[string]genesisGenerationCmd{
	gethGenesisFilename: func(genesisConfigPath *services.SharedPath)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_geth.py",
			genesisConfigPath.GetAbsPathOnServiceContainer(),
		}
	},
	nethermindGenesisFilename: func(genesisConfigPath *services.SharedPath)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_chainspec.py",
			genesisConfigPath.GetAbsPathOnServiceContainer(),
		}
	},
	besuGenesisFilename: func(genesisConfigPath *services.SharedPath)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_besu.py",
			genesisConfigPath.GetAbsPathOnServiceContainer(),
		}
	},
}


func GenerateELGenesisData(
	serviceCtx *services.ServiceContext,
	genesisGenerationConfigTemplate *template.Template,
	genesisUnixTimestamp uint64,
	networkId string,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
) (
	*ELGenesisData,
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

	generationConfigSharedFile := configSharedDir.GetChildPath(genesisConfigFilename)
	templateData := &genesisGenerationConfigTemplateData{
		NetworkId:               networkId,
		DepositContractAddress:  depositContractAddress,
		UnixTimestamp:           genesisUnixTimestamp,
		TotalTerminalDifficulty: totalTerminalDifficulty,
	}
	if err := service_launch_utils.FillTemplateToSharedPath(genesisGenerationConfigTemplate, templateData, generationConfigSharedFile); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred filling the genesis generation config template")
	}

	genesisFilenameToFilepathOnModuleContainer := map[string]string{}
	for outputFilename, generationCmd := range allGenesisGenerationCmds {
		cmd := generationCmd(generationConfigSharedFile)
		genesisSharedFile := outputSharedDir.GetChildPath(outputFilename)
		if err := execCmdAndWriteOutputToSharedFile(serviceCtx, cmd, genesisSharedFile); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred running command '%v' to generate file '%v'",
				strings.Join(cmd, " "),
				outputFilename,
			 )
		}
		genesisFilenameToFilepathOnModuleContainer[outputFilename] = genesisSharedFile.GetAbsPathOnThisContainer()
	}

	result := newELGenesisData(
		outputSharedDir.GetAbsPathOnThisContainer(),
		genesisFilenameToFilepathOnModuleContainer[gethGenesisFilename],
		genesisFilenameToFilepathOnModuleContainer[nethermindGenesisFilename],
		genesisFilenameToFilepathOnModuleContainer[besuGenesisFilename],
	)
	return result, nil
}

func execCmdAndWriteOutputToSharedFile(
	serviceCtx *services.ServiceContext,
	cmdArgs []string,
	outputSharedFile *services.SharedPath,
) error {
	exitCode, output, err := serviceCtx.ExecCommand(cmdArgs)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred running command '%v'",
			strings.Join(cmdArgs, " "),
		)
	}
	if exitCode != successfulExecCmdExitCode {
		return stacktrace.NewError(
			"Expected command '%v' to return exit code '%v' but returned '%v' with the following logs:\n%v",
			strings.Join(cmdArgs, " "),
			successfulExecCmdExitCode,
			exitCode,
			output,
		)
	}
	filepathOnModuleContainer := outputSharedFile.GetAbsPathOnThisContainer()
	if err := ioutil.WriteFile(filepathOnModuleContainer, []byte(output), os.ModePerm); err != nil {
		return stacktrace.Propagate(err, "An error occurred writing chainspec file '%v'", filepathOnModuleContainer)
	}
	return nil
}
