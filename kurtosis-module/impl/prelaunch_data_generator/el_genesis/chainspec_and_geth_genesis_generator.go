package el_genesis

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

const (
	chainspecAndGethGenesisGenerationConfigFilename = "config.yml"

	chainspecJsonFilename = "chainspec.json"
	gethGenesisJsonFilename = "geth.json"

	successfulExecCmdExitCode = 0
)

type chainspecAndGethGenesisGenerationConfigTemplateData struct {
	NetworkId string
	DepositContractAddress string
	UnixTimestamp uint64
	TotalTerminalDifficulty uint64
}

// The commands used in this function come from:
//  https://github.com/skylenet/ethereum-genesis-generator/blob/master/entrypoint.sh
func generateChainspecAndGethGenesis(
	generationConfigTemplate *template.Template,
	configSharedDir *services.SharedPath,
	networkId string,
	genesisUnixTimestamp uint64,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
	serviceCtx *services.ServiceContext,
	outputSharedDir *services.SharedPath,
) (
	resultChainspecFilepathOnModuleContainer string,
	resultGethGenesisFilepathOnModuleContainer string,
	resultErr error,
){
	generationConfigSharedFile := configSharedDir.GetChildPath(chainspecAndGethGenesisGenerationConfigFilename)
	templateData := &chainspecAndGethGenesisGenerationConfigTemplateData{
		NetworkId:               networkId,
		DepositContractAddress:  depositContractAddress,
		UnixTimestamp:           genesisUnixTimestamp,
		TotalTerminalDifficulty: totalTerminalDifficulty,
	}
	if err := service_launch_utils.FillTemplateToSharedPath(generationConfigTemplate, templateData, generationConfigSharedFile); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred filling the template for the config file for EL chainspec & Geth genesis generation")
	}

	chainspecGenerationCmdArgs := []string{
		 "python3",
		 "/apps/el-gen/genesis_chainspec.py",
		 generationConfigSharedFile.GetAbsPathOnServiceContainer(),
	}
	chainspecSharedFile := outputSharedDir.GetChildPath(chainspecJsonFilename)
	if err := execCmdAndWriteOutputToSharedFile(serviceCtx, chainspecGenerationCmdArgs, chainspecSharedFile); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred running the chainspec generation command and storing its output to file")
	}

	gethGenesisGenerationCmdArgs := []string{
		"python3",
		"/apps/el-gen/genesis_geth.py",
		generationConfigSharedFile.GetAbsPathOnServiceContainer(),
	}
	gethGenesisSharedFile := outputSharedDir.GetChildPath(gethGenesisJsonFilename)
	if err := execCmdAndWriteOutputToSharedFile(serviceCtx, gethGenesisGenerationCmdArgs, gethGenesisSharedFile); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred running the Geth genesis generation command and storing its output to file")
	}

	return chainspecSharedFile.GetAbsPathOnThisContainer(), gethGenesisSharedFile.GetAbsPathOnThisContainer(), nil
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
