package ethereum_genesis_generator

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
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
	outputGethGenesisJsonRelFilepath    = "el/geth.json"
	outputClGenesisRelDirpath           = "cl"
	outputClGenesisConfigYmlRelFilepath = outputClGenesisRelDirpath + "/config.yaml"
	outputClGenesisSszRelFilepath       = outputClGenesisRelDirpath + "/genesis.ssz"

	containerStopTimeoutSeconds = 3
)
// We run the genesis generation as an exec command instead, so that we get immediate feedback if it fails
var entrypoingArgs = []string{
	"sleep",
	"99999",
}

type elGenesisConfigTemplateData struct {
	NetworkId string
	UnixTimestamp int64
}
type clGenesisConfigTemplateData struct {
	NetworkId string
	SecondsPerSlot uint32
	UnixTimestamp int64
}

func GenerateELAndCLGenesisConfig(
	enclaveCtx *enclaves.EnclaveContext,
	elGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clMnemonicsYmlFilepathOnModuleContainer string,
	unixTimestamp int64,
	networkId string,
	secondsPerSlot uint32,
) (
	resultGethELGenesisJSONFilepath string,
	resultCLGenesisPaths *CLGenesisPaths,
	resultErr error,
) {
	serviceCtx, err := enclaveCtx.AddService(serviceId, getContainerConfig)
	if err != nil {
		return "", nil, stacktrace.Propagate(err, "An error occurred launching the Ethereum genesis-generating container with service ID '%v'", serviceId)
	}

	gethGenesisJsonFilepath, clGenesisPaths, err := generateGenesisData(
		serviceCtx,
		elGenesisConfigYmlTemplate,
		clGenesisConfigYmlTemplate,
		clMnemonicsYmlFilepathOnModuleContainer,
		unixTimestamp,
		networkId,
		secondsPerSlot,
	)
	if err != nil {
		return "", nil, stacktrace.Propagate(err, "An error occurred generating genesis data")
	}

	if err := enclaveCtx.RemoveService(serviceId, containerStopTimeoutSeconds); err != nil {
		logrus.Errorf(
			"An error occurred stopping the genesis generation service with ID '%v' and timeout '%vs'; you'll need to stop it manually",
			serviceId,
			containerStopTimeoutSeconds,
		)
	}

	return gethGenesisJsonFilepath, clGenesisPaths, nil
}

func generateGenesisData(
	serviceCtx *services.ServiceContext,
	gethGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	clMnemonicsYmlFilepathOnModuleContainer string,
	unixTimestamp int64,
	networkId string,
	secondsPerSlot uint32,
) (
	resultGethGenesisJsonFilepathOnModuleContainer string,
	resultClGenesisPaths *CLGenesisPaths,
	resultErr error,
) {
	elTemplateData := elGenesisConfigTemplateData{
		NetworkId:     networkId,
		UnixTimestamp: unixTimestamp,
	}
	clTemplateData := clGenesisConfigTemplateData{
		NetworkId:      networkId,
		SecondsPerSlot: secondsPerSlot,
		UnixTimestamp:  unixTimestamp,
	}

	sharedDir := serviceCtx.GetSharedDirectory()

	// Make the Geth genesis config available to the generator
	gethGenesisConfigYmlSharedPath := sharedDir.GetChildPath(elGenesisConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(gethGenesisConfigYmlTemplate, elTemplateData, gethGenesisConfigYmlSharedPath); err != nil {
		return "", nil, stacktrace.Propagate(err, "An error occurred filling the Geth genesis config template")
	}

	// Make the CL genesis config available to the generator
	clGenesisConfigYmlSharedPath := sharedDir.GetChildPath(clGenesisConfigYmlRelFilepathInSharedDir)
	if err := service_launch_utils.FillTemplateToSharedPath(clGenesisConfigYmlTemplate, clTemplateData, clGenesisConfigYmlSharedPath); err != nil {
		return "", nil, stacktrace.Propagate(err, "An error occurred filling the CL genesis config template")
	}

	// Make the CL mnemonics file available to the generator container
	clMnemonicsYmlSharedPath := sharedDir.GetChildPath(clMnemonicsYmlRelFilepathInSharedDir)
	if err := service_launch_utils.CopyFileToSharedPath(clMnemonicsYmlFilepathOnModuleContainer, clMnemonicsYmlSharedPath); err != nil {
		return "", nil, stacktrace.Propagate(
			err,
			"An error occurred copying CL mnemonics file '%v' into shared directory relative path '%v'",
			clMnemonicsYmlFilepathOnModuleContainer,
			clMnemonicsYmlRelFilepathInSharedDir,
		)
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
		return "", nil, stacktrace.Propagate(err, "An error occurred executing command '%v' to generate the genesis data inside the generator container", cmdStr)
	}
	if exitCode != generationCommandExpectedExitCode {
		return "", nil, stacktrace.NewError(
			"Expected genesis-generating command '%v' to exit with code %v but got %v instead and the following logs:\n%v",
			cmdStr,
			generationCommandExpectedExitCode,
			exitCode,
			output,

		)
	}
	logrus.Debugf("Genesis generation output:\n%v", output)

	outputDirpathOnModuleContainer := outputSharedPath.GetAbsPathOnThisContainer()
	gethGenesisJsonFilepathOnModuleContainer := path.Join(
		outputDirpathOnModuleContainer,
		outputGethGenesisJsonRelFilepath,
	)
	clGenesisPaths := getClGenesisPathsFromOutputDirpath(outputDirpathOnModuleContainer)

	return gethGenesisJsonFilepathOnModuleContainer, clGenesisPaths, nil
}

func getContainerConfig(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfig := services.NewContainerConfigBuilder(
		imageName,
	).WithEntrypointOverride(
		entrypoingArgs,
	).Build()

	return containerConfig, nil
}

func getClGenesisPathsFromOutputDirpath(outputDirpathOnModuleContainer string) *CLGenesisPaths {
	clGenesisDirpathOnModuleContainer := path.Join(
		outputDirpathOnModuleContainer,
		outputClGenesisRelDirpath,
	)
	clGenesisConfigYmlFilepathOnModuleContainer := path.Join(
		outputDirpathOnModuleContainer,
		outputClGenesisConfigYmlRelFilepath,
	)
	clGenesisSszFilepathOnModuleContainer := path.Join(
		outputDirpathOnModuleContainer,
		outputClGenesisSszRelFilepath,
	)

	clGenesisPaths := NewCLGenesisPaths(
		clGenesisDirpathOnModuleContainer,
		clGenesisConfigYmlFilepathOnModuleContainer,
		clGenesisSszFilepathOnModuleContainer,
	)
	return clGenesisPaths
}