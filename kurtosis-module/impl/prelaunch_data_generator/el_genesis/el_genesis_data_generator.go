package el_genesis

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io/ioutil"
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
	genesisConfigArtifactId, err := enclaveCtx.UploadFiles(genesisConfigFilepathOnModule)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred uploading the genesis config filepath from '%v'", genesisConfigFilepathOnModule)
	}

	configDirpathOnGenerator := path.Join(genesisDirpathOnGenerator, configDirname)
	outputDirpathOnGenerator := path.Join(genesisDirpathOnGenerator, outputDirname)

	serviceCtx, err := prelaunch_data_generator.LaunchPrelaunchDataGenerator(
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
		"sh",
		"-c",
		strings.Join(allDirpathCreationCommands, " && "),
	}
	exitCode, output, err := serviceCtx.ExecCommand(dirCreationCmd)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred executing dir creation command '%+v' on the generator container")
	}
	if exitCode != successfulExecCmdExitCode {
		return nil, stacktrace.NewError(
			"Dir creation command '%+v' should have returned %v but returned %v with the following output:\n%v",
			successfulExecCmdExitCode,
			exitCode,
			output,
		)
	}

	genesisConfigFilepathOnGenerator := path.Join(configDirpathOnGenerator, genesisConfigFilename)
	genesisFilenameToFilepathOnModuleContainer := map[string]string{}
	for outputFilename, generationCmd := range allGenesisGenerationCmds {
		cmd := generationCmd(genesisConfigFilepathOnGenerator)
		outputFilepathOnGenerator := path.Join(outputDirpathOnGenerator, outputFilename)
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

	/*
	sharedDir := serviceCtx.GetSharedDirectory()
	generationInstanceSharedDir := sharedDir.GetChildPath(fmt.Sprintf(
		"%v%v",
		genesisDirpathOnGenerator,
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
	*/

	jwtSecretSharedFile := outputSharedDir.GetChildPath(jwtSecretFilename)
	jwtSecretGenerationCmdArgs := []string{
		"bash",
		"-c",
		fmt.Sprintf(
			"openssl rand -hex 32 | tr -d \"\\n\" | sed 's/^/0x/' > %v",
			jwtSecretSharedFile.GetAbsPathOnServiceContainer(),
		),
	}

	jwtSecretGenerationExitCode, jwtSecretGenerationOutput, err := serviceCtx.ExecCommand(jwtSecretGenerationCmdArgs)
	if err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred executing command '%v' to generate the JWT secret",
			strings.Join(jwtSecretGenerationCmdArgs, " "),
		)
	}
	if jwtSecretGenerationExitCode != successfulExecCmdExitCode {
		return nil, stacktrace.NewError(
			"Expected JWT secret generation command '%v' to return exit code '%v' but returned '%v' with the following logs:\n%v",
			strings.Join(jwtSecretGenerationCmdArgs, " "),
			successfulExecCmdExitCode,
			jwtSecretGenerationExitCode,
			jwtSecretGenerationOutput,
		)
	}

	result := newELGenesisData(
		outputSharedDir.GetAbsPathOnThisContainer(),
		jwtSecretSharedFile.GetAbsPathOnThisContainer(),
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
