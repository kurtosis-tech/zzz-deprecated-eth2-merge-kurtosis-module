package ethereum_genesis_generator

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"strings"
	"text/template"
)

const (
	imageName                    = "kurtosistech/ethereum-genesis-generator"
	serviceId services.ServiceID = "eth-genesis-generator"

	// The filepaths, relative to shared dir root, where we're going to put EL & CL config
	// (and then copy them into the expected locations on image start)
	elGenesisConfigYmlRelFilepathInSharedDir = "el-genesis-config.yml"
	clGenesisConfigYmlRelFilepathInSharedDir = "cl-genesis-config.yml"
	clMnemonicsYmlRelFilepathInSharedDir = "cl-mnemonics.yml"

	// The genesis generation image is configured by dropping files into a specific hardcoded location
	// These are those locations
	// See https://github.com/skylenet/ethereum-genesis-generator
	expectedConfigDirpathOnService     = "/config"
	expectedELConfigDirpathOnService   = expectedConfigDirpathOnService + "/el"
	expectedELGenesisConfigYmlFilepath = expectedELConfigDirpathOnService + "/genesis-config.yaml"
	expectedCLConfigDirpathOnService   = expectedConfigDirpathOnService + "/cl"
	expectedCLGenesisConfigYmlFilepath = expectedCLConfigDirpathOnService + "/config.yaml"
	expectedCLMnemonicsYmlFilepath = expectedCLConfigDirpathOnService + "/mnemonics.yaml"

	// The generator container hardcodes the location where it'll write the output to; this is the location
	outputGenesisDataDirpathOnGeneratorContainer = "/data"

	// Location, relative to the root of shared dir, where the genesis data will be copied to from the generator container
	//  so that the module container can store and use it
	genesisDataRelDirpathInSharedDir = "data"

	// This is the entrypoint that the Dockerfile uses (though we override it so that we can do some extra work
	//  before it runs)
	entrypointFromDockerfile = "/work/entrypoint.sh"

	generationCommandExpectedExitCode = 0

	// Paths, *relative to the root of the output genesis data directory, where the generator writes data
	outputGethGenesisConfigRelFilepath = "el/geth.json"
	outputClGenesisConfigRelDirpath = "cl"

	containerStopTimeoutSeconds = 3
)
// We run the genesis generation as an exec command instead, so that we get immediate feedback if it fails
var entrypoingArgs = []string{
	"sleep",
	"99999",
}

type elGenesisConfigTemplateData struct {
	NetworkId string
}
type clGenesisConfigTemplateData struct {
	NetworkId string
	SecondsPerSlot uint32
}

func GenerateELAndCLGenesisConfig(
	enclaveCtx *enclaves.EnclaveContext,
	elGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clMnemonicsYmlFilepathOnModuleContainer string,
	networkId string,
	secondsPerSlot uint32,
) (
	resultGethELGenesisJSONFilepath string,
	resultCLGenesisDataDirpath string,
	resultErr error,
) {
	serviceCtx, err := enclaveCtx.AddService(serviceId, getContainerConfig)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis-generating container with service ID '%v'", serviceId)
	}

	gethGenesisJsonFilepath, clGenesisDataDirpath, err := generateGenesisData(
		serviceCtx,
		elGenesisConfigYmlTemplate,
		clGenesisConfigYmlTemplate,
		clMnemonicsYmlFilepathOnModuleContainer,
		networkId,
		secondsPerSlot,
	)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred generating genesis data")
	}

	if err := enclaveCtx.RemoveService(serviceId, containerStopTimeoutSeconds); err != nil {
		logrus.Errorf(
			"An error occurred stopping the genesis generation service with ID '%v' and timeout '%vs'; you'll need to stop it manually",
			serviceId,
			containerStopTimeoutSeconds,
		)
	}

	return gethGenesisJsonFilepath, clGenesisDataDirpath, nil
}

func generateGenesisData(
	serviceCtx *services.ServiceContext,
	gethGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clMnemonicsYmlFilepathOnModuleContainer string,
	networkId string,
	secondsPerSlot uint32,
) (
	resultGethGenesisJsonFilepathOnModuleContainer string,
	resultClConfigDataDirpathOnModuleContainer string,
	resultErr error,
) {
	elTemplateData := elGenesisConfigTemplateData{
		NetworkId: networkId,
	}
	clTemplateData := clGenesisConfigTemplateData{
		NetworkId:      networkId,
		SecondsPerSlot: secondsPerSlot,
	}

	sharedDir := serviceCtx.GetSharedDirectory()

	// Make the Geth genesis config available to the generator
	gethGenesisConfigYmlSharedPath := sharedDir.GetChildPath(elGenesisConfigYmlRelFilepathInSharedDir)
	gethGenesisConfigYmlFilepathOnModuleContainer := gethGenesisConfigYmlSharedPath.GetAbsPathOnThisContainer()
	gethGenesisConfigYmlFp, err := os.Create(gethGenesisConfigYmlFilepathOnModuleContainer)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred opening filepath '%v' on the module container for writing the Geth genesis config YAML", gethGenesisConfigYmlFilepathOnModuleContainer)
	}
	if err := gethGenesisConfigYmlTemplate.Execute(gethGenesisConfigYmlFp, elTemplateData); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred filling the Geth genesis config template")
	}

	// Make the CL genesis config available to the generator
	clGenesisConfigYmlSharedPath := sharedDir.GetChildPath(clGenesisConfigYmlRelFilepathInSharedDir)
	clGenesisConfigYmlFilepathOnModuleContainer := clGenesisConfigYmlSharedPath.GetAbsPathOnThisContainer()
	clGenesisConfigYmlFp, err := os.Create(clGenesisConfigYmlFilepathOnModuleContainer)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred opening filepath '%v' on the module container for writing the CL genesis config YAML", clGenesisConfigYmlFilepathOnModuleContainer)
	}
	if err := clGenesisConfigYmlTemplate.Execute(clGenesisConfigYmlFp, clTemplateData); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred filling the CL genesis config template")
	}

	// Make the CL mnemonics file available to the generator container
	clMnemonicsYmlSharedPath := sharedDir.GetChildPath(clMnemonicsYmlRelFilepathInSharedDir)
	srcMnemonicsFp, err := os.Open(clMnemonicsYmlFilepathOnModuleContainer)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred opening the CL mnemonics file on the module container at '%v'", clMnemonicsYmlFilepathOnModuleContainer)
	}
	destMnemonicsFilepath := clMnemonicsYmlSharedPath.GetAbsPathOnThisContainer()
	destMnemonicsFp, err := os.Create(destMnemonicsFilepath)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred creating the CL mnemonics file inside the shared directory at '%v'", destMnemonicsFilepath)
	}
	if _, err := io.Copy(destMnemonicsFp, srcMnemonicsFp); err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred copying the contents of the CL mnemonics file from '%v' to '%v'", clMnemonicsYmlFilepathOnModuleContainer, destMnemonicsFilepath)
	}

	outputSharedPath := sharedDir.GetChildPath(genesisDataRelDirpathInSharedDir)

	cmdArgs := []string{
		// We first symlink the EL & CL config dirpaths in the shared directory (which we've populated) to the locations
		//  where the container expects, so that it picks up the files we're dropping into place
		"cp",
		gethGenesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
		expectedELGenesisConfigYmlFilepath,
		"&&",
		"cp",
		clGenesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
		expectedCLGenesisConfigYmlFilepath,
		"&&",
		"cp",
		clMnemonicsYmlSharedPath.GetAbsPathOnServiceContainer(),
		expectedCLMnemonicsYmlFilepath,
		"&&",
		entrypointFromDockerfile,
		"all",
		"&&",
		"cp",
		"-R",
		outputGenesisDataDirpathOnGeneratorContainer,
		outputSharedPath.GetAbsPathOnServiceContainer(),
	}
	cmdStr := strings.Join(cmdArgs, " ")

	exitCode, output, err := serviceCtx.ExecCommand([]string{"sh", "-c", cmdStr})
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred executing command '%v' to generate the genesis data inside the generator container", cmdStr)
	}
	if exitCode != generationCommandExpectedExitCode {
		return "", "", stacktrace.NewError(
			"Expected genesis-generating command '%v' to exit with code %v but got %v instead and the following logs:\n%v",
			cmdStr,
			generationCommandExpectedExitCode,
			exitCode,
			output,

		)
	}
	logrus.Debugf("Genesis generation output:\n%v", output)

	gethGenesisJsonFilepathOnModuleContainer := path.Join(
		outputSharedPath.GetAbsPathOnThisContainer(),
		outputGethGenesisConfigRelFilepath,
	)
	clGenesisDirpathOnModuleContainer := path.Join(
		outputSharedPath.GetAbsPathOnThisContainer(),
		outputClGenesisConfigRelDirpath,
	)
	return gethGenesisJsonFilepathOnModuleContainer, clGenesisDirpathOnModuleContainer, nil
}

func getContainerConfig(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfig := services.NewContainerConfigBuilder(
		imageName,
	).WithEntrypointOverride(
		entrypoingArgs,
	).Build()

	return containerConfig, nil
}
