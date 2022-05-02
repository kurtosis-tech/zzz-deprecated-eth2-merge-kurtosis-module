package el_genesis

import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/new_launcher"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
	"text/template"
)

const (
	// The prefix dirpath on the generation container where generation output will be placed
	genesisDirpathOnGenerator = "/el-genesis"

	configDirname                      = "config"
	genesisConfigFilename  = "genesis-config.yaml"

	outputDirname = "output"

	gethGenesisFilename = "geth.json"
	nethermindGenesisFilename = "nethermind.json"
	besuGenesisFilename = "besu.json"

	jwtSecretFilename = "jwtsecret"

	successfulExecCmdExitCode = 0
)

type genesisGenerationConfigTemplateData struct {
	NetworkId string
	DepositContractAddress string
	UnixTimestamp uint64
	TotalTerminalDifficulty uint64
}

type genesisGenerationCmd func(genesisConfigFilepathOnGenerator string)[]string

// Mapping of output genesis filename -> generator to create the file
var allGenesisGenerationCmds = map[string]genesisGenerationCmd{
	gethGenesisFilename: func(genesisConfigFilepathOnGenerator string)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_geth.py",
			genesisConfigFilepathOnGenerator,
		}
	},
	nethermindGenesisFilename: func(genesisConfigFilepathOnGenerator string)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_chainspec.py",
			genesisConfigFilepathOnGenerator,
		}
	},
	besuGenesisFilename: func(genesisConfigFilepathOnGenerator string)[]string{
		return []string{
			"python3",
			"/apps/el-gen/genesis_besu.py",
			genesisConfigFilepathOnGenerator,
		}
	},
}


func GenerateELGenesisData(
	ctx context.Context,
	enclaveCtx *enclaves.EnclaveContext,
	// serviceCtx *services.ServiceContext,
	genesisGenerationConfigTemplate *template.Template,
	genesisUnixTimestamp uint64,
	networkId string,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
) (
	*ELGenesisData,
	error,
) {
	templateData := &genesisGenerationConfigTemplateData{
		NetworkId:               networkId,
		DepositContractAddress:  depositContractAddress,
		UnixTimestamp:           genesisUnixTimestamp,
		TotalTerminalDifficulty: totalTerminalDifficulty,
	}
	genesisConfigFilepathOnModule := path.Join(os.TempDir(), genesisConfigFilename)
	if err := service_launch_utils.FillTemplateToPath(
		genesisGenerationConfigTemplate,
		templateData,
		genesisConfigFilepathOnModule,
	); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred creating the genesis config file at '%v'", genesisConfigFilepathOnModule)
	}
	logrus.Info(genesisConfigFilepathOnModule)
	genesisConfigArtifactId, err := enclaveCtx.UploadFiles(genesisConfigFilepathOnModule)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred uploading the genesis config filepath from '%v'", genesisConfigFilepathOnModule)
	}

	configDirpathOnGenerator := path.Join(genesisDirpathOnGenerator, configDirname)
	outputDirpathOnGenerator := path.Join(genesisDirpathOnGenerator, outputDirname)

	// TODO Make this the actual data generator
	serviceCtx, err := new_launcher.LaunchPrelaunchDataGenerator(
		enclaveCtx,
		map[services.FilesArtifactID]string{
			genesisConfigArtifactId: configDirpathOnGenerator,
		},
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the generator container")
	}

	allDirpathsToCreateOnGenerator := []string{
		genesisDirpathOnGenerator,
		configDirpathOnGenerator,
		outputDirpathOnGenerator,
	}
	allDirpathCreationCommands := []string{}
	for _, dirpathToCreateOnGenerator := range allDirpathsToCreateOnGenerator {
		allDirpathCreationCommands = append(
			allDirpathCreationCommands,
			fmt.Sprintf("mkdir -p %v", dirpathToCreateOnGenerator),
		)
	}
	dirCreationCmd := []string{
		"bash",
		"-c",
		strings.Join(allDirpathCreationCommands, " && "),
	}
	exitCode, output, err := serviceCtx.ExecCommand(dirCreationCmd)
	if err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred executing dir creation command '%+v' on the generator container",
			dirCreationCmd,
		)
	}
	if exitCode != successfulExecCmdExitCode {
		return nil, stacktrace.NewError(
			"Dir creation command '%+v' should have returned %v but returned %v with the following output:\n%v",
			dirCreationCmd,
			successfulExecCmdExitCode,
			exitCode,
			output,
		)
	}

	genesisConfigFilepathOnGenerator := path.Join(configDirpathOnGenerator, genesisConfigFilename)
	genesisFilenameToRelativeFilepathInArtifact := map[string]string{}
	for outputFilename, generationCmd := range allGenesisGenerationCmds {
		cmd := generationCmd(genesisConfigFilepathOnGenerator)
		outputFilepathOnGenerator := path.Join(outputDirpathOnGenerator, outputFilename)
		outputRedirectingCommand := append(cmd, ">", outputFilepathOnGenerator)
		cmdToExecute := []string{
			"bash",
			"-c",
			strings.Join(outputRedirectingCommand, " "),
		}
		if err := execCommand(serviceCtx, cmdToExecute); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred executing command '%+v' to create genesis config file '%v'",
				cmdToExecute,
				outputFilepathOnGenerator,
			)
		}
		genesisFilenameToRelativeFilepathInArtifact[outputFilename] = path.Join(
			path.Base(outputDirpathOnGenerator),
			outputFilename,
		)
	}

	jwtSecretFilepathOnGenerator := path.Join(outputDirpathOnGenerator, jwtSecretFilename)
	jwtSecretGenerationCmdArgs := []string{
		"bash",
		"-c",
		fmt.Sprintf(
			"openssl rand -hex 32 | tr -d \"\\n\" | sed 's/^/0x/' > %v",
			jwtSecretFilepathOnGenerator,
		),
	}
	if err := execCommand(serviceCtx, jwtSecretGenerationCmdArgs); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred executing the JWT secret generation command")
	}

	elGenesisDataArtifactId, err := enclaveCtx.StoreFilesFromService(ctx, serviceCtx.GetServiceID(), outputDirpathOnGenerator)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred storing the generated EL genesis data in the enclave")
	}

	result := newELGenesisData(
		elGenesisDataArtifactId,
		path.Join(path.Base(outputDirpathOnGenerator), jwtSecretFilename),
		genesisFilenameToRelativeFilepathInArtifact[gethGenesisFilename],
		genesisFilenameToRelativeFilepathInArtifact[nethermindGenesisFilename],
		genesisFilenameToRelativeFilepathInArtifact[besuGenesisFilename],
	)
	return result, nil
}

func execCommand(serviceCtx *services.ServiceContext, cmd []string) error {
	exitCode, output, err := serviceCtx.ExecCommand(cmd)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred executing command '%+v' on the generator container",
			cmd,
		)
	}
	if exitCode != successfulExecCmdExitCode {
		return stacktrace.NewError(
			"Command '%+v' should have returned %v but returned %v with the following output:\n%v",
			cmd,
			successfulExecCmdExitCode,
			exitCode,
			output,
		)
	}
	return nil
}