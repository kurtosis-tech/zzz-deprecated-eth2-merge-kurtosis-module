package ethereum_genesis_generator

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"path"
	"strings"
	"text/template"
)

const (
	imageName                    = "kurtosistech/ethereum-genesis-generator"
	serviceId services.ServiceID = "eth-genesis-generator"

	webserverPortId            = "webserver"
	webserverPortNumber uint16 = 8000

	waitForStartupMillisBetweenPolls = 1000
	waitForStartupMaxPolls           = 10
	waitInitialDelayMilliseconds     = 1500

	healthCheckUrlSlug = ""
	healthyValue       = ""

	successExitCode int32 = 0

	consensusConfigDataDirname = "data"

	executionLayerDirname = "el"
	consensusLayerDirname = "cl"
	gethGenesisJsonFilename = "geth.json"

	// The filepaths, relative to shared dir root, where we're going to put EL & CL config
	// (and then copy them into the expected locations on image start)
	elGenesisConfigYmlRelFilepathInSharedDir = "el-genesis-config.yml"
	clGenesisConfigYmlRelFilepathInSharedDir = "cl-genesis-config.yml"

	// The genesis generation image is configured by dropping files into a specific hardcoded location
	// These are those locations
	// See https://github.com/skylenet/ethereum-genesis-generator
	expectedConfigDirpathOnService     = "/config"
	expectedELConfigDirpathOnService   = expectedConfigDirpathOnService + "/el"
	expectedELGenesisConfigYmlFilepath = expectedELConfigDirpathOnService + "/genesis-config.yaml"
	expectedCLConfigDirpathOnService   = expectedConfigDirpathOnService + "/cl"
	expectedCLGenesisConfigYmlFilepath = expectedCLConfigDirpathOnService + "/config.yaml"

	// This is the entrypoint that the Dockerfile uses (though we override it so that we can do some extra work
	//  before it runs)
	entrypointFromDockerfile = "/work/entrypoint.sh"
)
var entrypoingArgs = []string{
	"sh",
	"-c",
}
var usedPorts = map[string]*services.PortSpec{
	webserverPortId: services.NewPortSpec(webserverPortNumber, services.PortProtocol_TCP),
}

type elGenesisConfigTemplateData struct {
	NetworkId string
}

type clGenesisConfigTemplateData struct {
	NetworkId string
}

func GenerateELAndCLGenesisConfig(
	enclaveCtx *enclaves.EnclaveContext,
	elGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	networkId string,
) (
	resultGethELGenesisJSONFilepath string,
	resultCLConfigDataDirpath string,
	resultErr error,
) {
	containerConfigSupplier := getContainerConfigSupplier(
		elGenesisConfigYmlTemplate,
		clGenesisConfigYmlTemplate,
		networkId,
	)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred launching the Ethereum Genesis Generator with service ID '%v'", serviceId)
	}

	//We wait for the web-service to be available as an indicator that the files were generated
	err = enclaveCtx.WaitForHttpGetEndpointAvailability(serviceId, uint32(webserverPortNumber), healthCheckUrlSlug, waitInitialDelayMilliseconds, waitForStartupMaxPolls, waitForStartupMillisBetweenPolls, healthyValue)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred checking service availability")
	}

	consensusConfigDataInGenesisGeneratorContainerDirpath := fmt.Sprintf("/%v",consensusConfigDataDirname)

	copyConfigFolderInSharedDirectoryCmd := []string{
		"cp",
		"-R",
		consensusConfigDataInGenesisGeneratorContainerDirpath,
		serviceCtx.GetSharedDirectory().GetAbsPathOnServiceContainer(),
	}

	exitCode, logOutput, err := serviceCtx.ExecCommand(copyConfigFolderInSharedDirectoryCmd)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred executing command '%v'", copyConfigFolderInSharedDirectoryCmd)
	}
	if exitCode != successExitCode {
		return "", "", stacktrace.NewError("Command '%v' execution fail with exit code '%v' and logs: \n'%v'", copyConfigFolderInSharedDirectoryCmd, exitCode, logOutput)
	}

	gethGenesisJsonFilepath := path.Join(
		serviceCtx.GetSharedDirectory().GetAbsPathOnThisContainer(),
		consensusConfigDataDirname,
		executionLayerDirname,
		gethGenesisJsonFilename,
	)
	consensusConfigDataDirpath := path.Join(
		serviceCtx.GetSharedDirectory().GetAbsPathOnThisContainer(),
		consensusConfigDataDirname,
		consensusLayerDirname,
	)

	return gethGenesisJsonFilepath, consensusConfigDataDirpath, nil
}

func getContainerConfigSupplier(
	gethGenesisConfigYmlTemplate *template.Template,
	clGenesisConfigYmlTemplate *template.Template,
	networkId string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	elTemplateData := elGenesisConfigTemplateData{
		NetworkId: networkId,
	}
	clTemplateData := clGenesisConfigTemplateData{
		NetworkId: networkId,
	}

	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		gethGenesisConfigYmlSharedPath := sharedDir.GetChildPath(elGenesisConfigYmlRelFilepathInSharedDir)
		gethGenesisConfigYmlFilepathOnModuleContainer := gethGenesisConfigYmlSharedPath.GetAbsPathOnThisContainer()
		gethGenesisConfigYmlFp, err := os.Create(gethGenesisConfigYmlFilepathOnModuleContainer)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening filepath '%v' on the module container for writing the Geth genesis config YAML", gethGenesisConfigYmlFilepathOnModuleContainer)
		}
		if err := gethGenesisConfigYmlTemplate.Execute(gethGenesisConfigYmlFp, elTemplateData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the Geth genesis config template")
		}

		clGenesisConfigYmlSharedPath := sharedDir.GetChildPath(clGenesisConfigYmlRelFilepathInSharedDir)
		clGenesisConfigYmlFilepathOnModuleContainer := clGenesisConfigYmlSharedPath.GetAbsPathOnThisContainer()
		clGenesisConfigYmlFp, err := os.Create(clGenesisConfigYmlFilepathOnModuleContainer)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening filepath '%v' on the module container for writing the CL genesis config YAML", clGenesisConfigYmlFilepathOnModuleContainer)
		}
		if err := clGenesisConfigYmlTemplate.Execute(clGenesisConfigYmlFp, clTemplateData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the CL genesis config template")
		}

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
			entrypointFromDockerfile,
			"all",
		}
		cmdStr := strings.Join(cmdArgs, " ")

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithEntrypointOverride(
			entrypoingArgs,
		).WithCmdOverride([]string{
			cmdStr,
		}).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}
	return result
}
