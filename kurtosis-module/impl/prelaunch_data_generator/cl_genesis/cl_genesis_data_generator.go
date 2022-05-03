package cl_genesis
import (
	"context"
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/el_genesis"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/prelaunch_data_generator_launcher"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
)

const (
	// Needed to copy the JWT secret
	elGenesisDirpathOnGenerator = "/el-genesis"

	configDirpathOnGenerator              = "/config"
	genesisConfigYmlFilename              = "config.yaml" // WARNING: Do not change this! It will get copied to the CL genesis data, and the CL clients are hardcoded to look for this filename
	mnemonicsYmlFilename = "mnemonics.yaml"

	outputDirpathOnGenerator = "/output"
	tranchesDiranme          = "tranches"
	genesisStateFilename     = "genesis.ssz"
	deployBlockFilename      = "deploy_block.txt"
	depositContractFilename = "deposit_contract.txt"

	// Generation constants
	clGenesisGenerationBinaryFilepathOnContainer = "/usr/local/bin/eth2-testnet-genesis"
	deployBlock = "0"
	eth1Block              = "0x0000000000000000000000000000000000000000000000000000000000000000"

	successfulExecCmdExitCode = 0
)

type clGenesisConfigTemplateData struct {
	NetworkId                          string
	SecondsPerSlot                     uint32
	UnixTimestamp                      uint64
	TotalTerminalDifficulty            uint64
	AltairForkEpoch                    uint64
	MergeForkEpoch                     uint64
	NumValidatorKeysToPreregister uint32
	PreregisteredValidatorKeysMnemonic string
	DepositContractAddress string
}

func GenerateCLGenesisData(
	ctx context.Context,
	enclaveCtx *enclaves.EnclaveContext,
	genesisGenerationConfigYmlTemplate *template.Template,
	genesisGenerationMnemonicsYmlTemplate *template.Template,
	elGenesisData *el_genesis.ELGenesisData, // Needed to get JWT secret
	genesisUnixTimestamp uint64,
	networkId string,
	depositContractAddress string,
	totalTerminalDifficulty uint64,
	secondsPerSlot uint32,
	altairForkEpoch uint64,
	mergeForkEpoch uint64,
	preregisteredValidatorKeysMnemonic string,
	numValidatorKeysToPreregister uint32,
) (
	*CLGenesisData,
	error,
) {
	tempDirpath, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred creating a temporary directory to store CL genesis config data in")
	}
	templateData := &clGenesisConfigTemplateData{
		NetworkId:                          networkId,
		SecondsPerSlot:                     secondsPerSlot,
		UnixTimestamp:                      genesisUnixTimestamp,
		TotalTerminalDifficulty:            totalTerminalDifficulty,
		AltairForkEpoch:                    altairForkEpoch,
		MergeForkEpoch:                     mergeForkEpoch,
		NumValidatorKeysToPreregister:      numValidatorKeysToPreregister,
		PreregisteredValidatorKeysMnemonic: preregisteredValidatorKeysMnemonic,
		DepositContractAddress:             depositContractAddress,
	}

	if err := service_launch_utils.FillTemplateToPath(
		genesisGenerationConfigYmlTemplate,
		templateData,
		path.Join(tempDirpath, genesisConfigYmlFilename),
	); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred filling the CL genesis generation config YML template")
	}

	if err := service_launch_utils.FillTemplateToPath(
		genesisGenerationMnemonicsYmlTemplate,
		templateData,
		path.Join(tempDirpath, mnemonicsYmlFilename),
	); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred filling the CL genesis generation mnemonics YML template")
	}

	genesisGenerationConfigArtifactId, err := enclaveCtx.UploadFiles(tempDirpath)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred storing the CL genesis generation config files at '%v'", tempDirpath)
	}

	// TODO Make this the actual data generator
	serviceCtx, err := prelaunch_data_generator_launcher.LaunchPrelaunchDataGenerator(
		enclaveCtx,
		map[services.FilesArtifactID]string{
			genesisGenerationConfigArtifactId: configDirpathOnGenerator,
			elGenesisData.GetFilesArtifactID(): elGenesisDirpathOnGenerator,
		},
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the generator container")
	}
	defer func() {
		serviceId := serviceCtx.GetServiceID()
		if err := enclaveCtx.RemoveService(serviceId, 0); err != nil {
			logrus.Warnf("Tried to remove prelaunch data generator service '%v', but doing so threw an error:\n%v", serviceId, err)
		}
	}()

	allDirpathsToCreateOnGenerator := []string{
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
	if err := execCommand(serviceCtx, dirCreationCmd); err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred executing dir creation command '%+v' on the generator container",
			dirCreationCmd,
		)
	}

	// Copy files to output
	allFilepathsToCopyToOuptutDirectory := []string{
		// The path.Base is necessary due to Kurtosis not yet flattening directories when uploaded
		path.Join(configDirpathOnGenerator, path.Base(tempDirpath), genesisConfigYmlFilename),
		path.Join(configDirpathOnGenerator, path.Base(tempDirpath), mnemonicsYmlFilename),
		path.Join(elGenesisDirpathOnGenerator, elGenesisData.GetJWTSecretRelativeFilepath()),
	}
	for _, filepathOnGenerator := range allFilepathsToCopyToOuptutDirectory {
		cmd := []string{
			"cp",
			filepathOnGenerator,
			outputDirpathOnGenerator,
		}
		if err := execCommand(serviceCtx, cmd); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred executing command '%+v' to copy a file to CL genesis output directory '%v'",
				cmd,
				outputDirpathOnGenerator,
			)
		}
	}

	// Generate files that need dynamic content
	contentToWriteToOutputFilename := map[string]string{
		deployBlock: deployBlockFilename,
		depositContractAddress: depositContractFilename,
	}
	for content, destFilename := range contentToWriteToOutputFilename {
		destFilepath := path.Join(outputDirpathOnGenerator, destFilename)
		cmd := []string{
			"sh",
			"-c",
			fmt.Sprintf(
				 "echo %v > %v",
				 content,
				 destFilepath,
			),
		}
		if err := execCommand(serviceCtx, cmd); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred executing command '%+v' to write content '%v' to file '%v'",
				cmd,
				content,
				destFilepath,
			)
		}
	}

	clGenesisGenerationCmdArgs := []string{
		clGenesisGenerationBinaryFilepathOnContainer,
		"phase0",
		"--config", path.Join(outputDirpathOnGenerator, genesisConfigYmlFilename),
		"--eth1-block", eth1Block,
		"--mnemonics", path.Join(outputDirpathOnGenerator, mnemonicsYmlFilename),
		"--timestamp", fmt.Sprintf("%v", genesisUnixTimestamp),
		"--tranches-dir", path.Join(outputDirpathOnGenerator, tranchesDiranme),
		"--state-output", path.Join(outputDirpathOnGenerator, genesisStateFilename),
	}
	if err := execCommand(serviceCtx, clGenesisGenerationCmdArgs); err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred executing command '%+v' to generate CL genesis data in directory '%v'",
			clGenesisGenerationCmdArgs,
			outputDirpathOnGenerator,
		)
	}

	clGenesisDataArtifactId, err := enclaveCtx.StoreFilesFromService(ctx, serviceCtx.GetServiceID(), outputDirpathOnGenerator)
	if err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred storing the CL genesis files at '%v' in service '%v'",
			outputDirpathOnGenerator,
			serviceCtx.GetServiceID(),
		)
	}

	jwtSecretRelFilepath := path.Join(
		path.Base(outputDirpathOnGenerator),
		path.Base(elGenesisData.GetJWTSecretRelativeFilepath()),
	)
	genesisConfigRelFilepath := path.Join(
		path.Base(outputDirpathOnGenerator),
		genesisConfigYmlFilename,
	)
	genesisSszRelFilepath := path.Join(
		path.Base(outputDirpathOnGenerator),
		genesisStateFilename,
	)
	result := newCLGenesisData(
		clGenesisDataArtifactId,
		jwtSecretRelFilepath,
		genesisConfigRelFilepath,
		genesisSszRelFilepath,
	)

	return result, nil
}

/*
func createGenesisGenerationConfig(
	genesisGenerationConfigYmlTemplate *template.Template,
	genesisGenerationMnemonicsYmlTemplate *template.Template,
	templateData *clGenesisConfigTemplateData,
	configSharedDir *services.SharedPath,
) (
	resultConfigYmlSharedFile *services.SharedPath,
	resultMnemonicsYmlSharedFile *services.SharedPath,
	resultErr error,
){

	return genesisGenerationConfigSharedFile, genesisGenerationMnemonicsSharedFile, nil
}

 */

/*
func runClGenesisGeneration(
	genesisGenerationConfigSharedFile *services.SharedPath,
	genesisGenerationMnemonicsSharedFile *services.SharedPath,
	jwtSecretFilepathOnModuleContainer string,
	genesisTimestamp uint64,
	depositContractAddress string,
	serviceCtx *services.ServiceContext,
	outputSharedDir *services.SharedPath,
) (
	*CLGenesisData,
	error,
){
	// Copy the genesis config file to output directory
	genesisGenerationConfigFilepathOnThisContainer := genesisGenerationConfigSharedFile.GetAbsPathOnThisContainer()
	genesisGenerationConfigSrcFp, err := os.Open(genesisGenerationConfigFilepathOnThisContainer)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred opening genesis generation config file '%v' for reading", genesisGenerationConfigFilepathOnThisContainer)
	}
	genesisConfigSharedFile := outputSharedDir.GetChildPath(genesisConfigYmlFilename)
	genesisConfigFilepathOnThisContainer := genesisConfigSharedFile.GetAbsPathOnThisContainer()
	genesisConfigFp, err := os.Create(genesisConfigFilepathOnThisContainer)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred opening genesis config file '%v' for writing", genesisConfigFilepathOnThisContainer)
	}
	if _, err := io.Copy(genesisConfigFp, genesisGenerationConfigSrcFp); err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred copying the genesis generation config file '%v' to '%v'",
			genesisGenerationConfigFilepathOnThisContainer,
			genesisConfigFilepathOnThisContainer,
		 )
	}

	// Create deploy block file
	deployBlockSharedFile := outputSharedDir.GetChildPath(deployBlockFilename)
	deployBlockFilepathOnThisContainer := deployBlockSharedFile.GetAbsPathOnThisContainer()
	if err := ioutil.WriteFile(deployBlockFilepathOnThisContainer, []byte(deployBlock), os.ModePerm); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred writing the deploy block file at '%v'", deployBlockFilepathOnThisContainer)
	}

	// Create deposit contract file
	depositContractSharedFile := outputSharedDir.GetChildPath(depositContractFilename)
	depositContractFilepathOnThisContainer := depositContractSharedFile.GetAbsPathOnThisContainer()
	if err := ioutil.WriteFile(depositContractFilepathOnThisContainer, []byte(depositContractAddress), os.ModePerm); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred writing the deposit contract file at '%v'", depositContractFilepathOnThisContainer)
	}

	genesisStateSharedFile := outputSharedDir.GetChildPath(genesisStateFilename)
	tranchesSharedDir := outputSharedDir.GetChildPath(tranchesDiranme)

	clGenesisGenerationCmdArgs := []string{
		clGenesisGenerationBinaryFilepathOnContainer,
		"phase0",
		"--config", genesisGenerationConfigSharedFile.GetAbsPathOnServiceContainer(),
		"--eth1-block", eth1Block,
		"--mnemonics", genesisGenerationMnemonicsSharedFile.GetAbsPathOnServiceContainer(),
		"--timestamp", fmt.Sprintf("%v", genesisTimestamp),
		"--tranches-dir", tranchesSharedDir.GetAbsPathOnServiceContainer(),
		"--state-output", genesisStateSharedFile.GetAbsPathOnServiceContainer(),
	}

	genesisGenerationExitCode, genesisGenerationOutput, err := serviceCtx.ExecCommand(clGenesisGenerationCmdArgs)
	if err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred executing command '%v' to generate the CL genesis data",
			strings.Join(clGenesisGenerationCmdArgs, " "),
		 )
	}
	if genesisGenerationExitCode != successCommandExitCode {
		return nil, stacktrace.NewError(
			"Expected CL genesis data generation command '%v' to return exit code '%v' but returned '%v' with the following logs:\n%v",
			strings.Join(clGenesisGenerationCmdArgs, " "),
			successCommandExitCode,
			genesisGenerationExitCode,
			genesisGenerationOutput,
		 )
	}

	jwtSecretSharedFile := outputSharedDir.GetChildPath(jwtSecretFilename)
	if err := service_launch_utils.CopyFileToSharedPath(jwtSecretFilepathOnModuleContainer, jwtSecretSharedFile); err != nil {
		return nil, stacktrace.Propagate(
			err,
			"An error occurred copying JWT secret file from path '%v' to shared filepath '%v'",
			jwtSecretFilepathOnModuleContainer,
			jwtSecretSharedFile.GetAbsPathOnThisContainer(),
		)
	}

	result := newCLGenesisData(
		outputSharedDir.GetAbsPathOnThisContainer(),
		jwtSecretSharedFile.GetAbsPathOnThisContainer(),
		genesisConfigSharedFile.GetAbsPathOnThisContainer(),
		genesisStateSharedFile.GetAbsPathOnThisContainer(),
	)
	return result, nil
}

 */


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
