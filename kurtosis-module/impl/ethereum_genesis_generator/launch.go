package ethereum_genesis_generator

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strings"
)

const (
	imageName                    = "kurtosistech/ethereum-genesis-generator"
	serviceId services.ServiceID = "eth-genesis-generator"

	webserverPortId            = "webserver"
	webserverPortNumber uint16 = 8000

	waitForStartupTimeBetweenPolls = 100
	waitForStartupMaxPolls         = 1000
	waitInitialDelayMilliseconds   = 1500

	healthCheckUrlSlug = ""
	healthyValue       = ""

	successExitCode int32 = 0

	consensusConfigDataDirname = "data"

	executionLayerDirname = "el"
	consensusLayerDirname = "cl"
	gethGenesisJsonFilename = "geth.json"
)

var usedPorts = map[string]*services.PortSpec{
	webserverPortId: services.NewPortSpec(webserverPortNumber, services.PortProtocol_TCP),
}

func LaunchEthereumGenesisGenerator(enclaveCtx *enclaves.EnclaveContext) (*services.ServiceContext, string, string, error) {
	containerConfigSupplier := getEthereumGenesisGeneratorConfigSupplier()
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, "", "", stacktrace.Propagate(err, "An error occurred launching the Ethereum Genesis Generator with service ID '%v'", serviceId)
	}

	//We wait for the web-service to be available as an indicator that the files were generated
	err = enclaveCtx.WaitForHttpGetEndpointAvailability(serviceId, uint32(webserverPortNumber), healthCheckUrlSlug, waitInitialDelayMilliseconds, waitForStartupMaxPolls, waitForStartupTimeBetweenPolls, healthyValue)
	if err != nil {
		return nil, "", "", stacktrace.Propagate(err, "An error occurred checking service availability")
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
		return nil, "", "", stacktrace.Propagate(err, "An error occurred executing command '%v'", copyConfigFolderInSharedDirectoryCmd)
	}
	if exitCode != successExitCode {
		return nil, "", "", stacktrace.NewError("Command '%v' execution fail with exit code '%v' and logs: \n'%v'", copyConfigFolderInSharedDirectoryCmd, exitCode, logOutput)
	}

	gethGenesisJsonFilepath := strings.Join([]string{
		serviceCtx.GetSharedDirectory().GetAbsPathOnThisContainer(),
		consensusConfigDataDirname,
		executionLayerDirname,
		gethGenesisJsonFilename,
	}, "/")

	consensusConfigDataDirpath := strings.Join([]string{
		serviceCtx.GetSharedDirectory().GetAbsPathOnThisContainer(),
		consensusConfigDataDirname,
		consensusLayerDirname,
	}, "/")

	return serviceCtx, gethGenesisJsonFilepath, consensusConfigDataDirpath, nil
}

func getEthereumGenesisGeneratorConfigSupplier() func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		cmdOverride := []string{"all"} //Generate genesis files for Execution and Consensus Layer. See more in https://github.com/skylenet/ethereum-genesis-generator

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride(
			cmdOverride,
		).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}
	return result
}
