package geth_el_client

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io"
	"os"
	"strings"
)

const (
	serviceId services.ServiceID = "geth-el-client"
	// imageName = "parithoshj/geth:merge-dd90624"
	imageName = "eth2-geth"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	RpcPortId          = "rpc"
	wsPortId           = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"

	// The filepath of the genesis JSON file in the shared directory, relative to the shared directory root
	sharedGenesisJsonRelFilepath = "genesis.json"

	// The dirpath of the execution data directory on the client container
	executionDataDirpathOnClientContainer = "/execution-data"
)
var usedPorts = map[string]*services.PortSpec{
	RpcPortId:          services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId:           services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}
var entrypointArgs = []string{"sh", "-c"}

func LaunchGethELClient(
	enclaveCtx *enclaves.EnclaveContext,
	genesisJsonFilepath string,
	networkId string,
	externalIpAddress string,
	bootnodeEnodes []string,
) (*services.ServiceContext, error) {
	containerConfigSupplier := getGethELContainerConfigSupplier(genesisJsonFilepath, networkId, externalIpAddress, bootnodeEnodes)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}
	return serviceCtx, nil
}

func getGethELContainerConfigSupplier(genesisJsonOnModuleContainerFilepath string, networkId string, externalIpAddress string, bootnodeEnodes []string) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		genesisJsonOnModuleContainerSharedPath := sharedDir.GetChildPath(sharedGenesisJsonRelFilepath)

		srcFp, err := os.Open(genesisJsonOnModuleContainerFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the genesis JSON file '%v' on the module container", genesisJsonOnModuleContainerFilepath)
		}

		destFilepath := genesisJsonOnModuleContainerSharedPath.GetAbsPathOnThisContainer()
		destFp, err := os.Create(destFilepath)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening the genesis JSON destination filepath '%v' on the module container", destFilepath)
		}

		if _, err := io.Copy(destFp, srcFp); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis file from the module container '%v' to the shared directory of the client container '%v'",
				genesisJsonOnModuleContainerFilepath,
				destFilepath,
			)
		}

		commandArgs := []string{
			"geth",
			"init",
			"--datadir=" + executionDataDirpathOnClientContainer,
			genesisJsonOnModuleContainerSharedPath.GetAbsPathOnServiceContainer(),
			"&&",
			"geth",
			"--datadir="  + executionDataDirpathOnClientContainer,
			"--networkid=" + networkId,
			"--catalyst",
			"--http",
			"--http.addr=0.0.0.0",
			"--http.api=engine,net,eth",
			"--ws",
			"--ws.api=engine,net,eth",
			"--allow-insecure-unlock",
			"--nat=extip:" + externalIpAddress,
			"--bootnodes=" + strings.Join(bootnodeEnodes, ","),
			"--verbosity=3",
		}
		commandStr := strings.Join(commandArgs, " ")

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithEntrypointOverride(
			entrypointArgs,
		).WithCmdOverride([]string{
			commandStr,
		}).Build()

		return containerConfig, nil
	}
	return result
}
