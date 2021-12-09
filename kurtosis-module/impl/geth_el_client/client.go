package geth_el_client

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"os"
)

const (
	serviceId services.ServiceID = "geth-el-client"
	imageName = "parithoshj/geth:merge-893c372"

	rpcPortNum       uint16 = 8545
	wsPortNum        uint16 = 8546
	discoveryPortNum uint16 = 30303

	// Port IDs
	rpcPortId = "rpc"
	wsPortId = "ws"
	tcpDiscoveryPortId = "tcp-discovery"
	udpDiscoveryPortId = "udp-discovery"

	// The name of the execution data directory inside the client service's shared directory
	executionDataDirname = "execution-data"

	// Because the execution data dir lives in the enclave data volume which is bind-mounted to the
	//  host machine, we need to make sure that the copy we make on the Geth client has full permissions
	//  so that the Geth client can create its socket inside the directory
	executionDataDirPerms = os.ModePerm

	// Passing in too long of a path to the execution directory will cause an error, so we have to symlink the execution
	//  data directory that we prepared for the client to a short path
	// See https://github.com/ethereum/go-ethereum/issues/16342
	executionDataSymlinkFilepath = "~/execution-data"
)
var usedPorts = map[string]*services.PortSpec{
	rpcPortId: services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	wsPortId: services.NewPortSpec(wsPortNum, services.PortProtocol_TCP),
	tcpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortId: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
}


func LaunchGethELClient(
	enclaveCtx *enclaves.EnclaveContext,
	executionDataDirpath string,
	networkId string,
	externalIpAddress string,
	bootnodeEnode string,
) (*services.ServiceContext, error) {
	containerConfigSupplier := getGethELContainerConfigSupplier(executionDataDirpath, networkId, externalIpAddress, bootnodeEnode)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}
	return serviceCtx, nil
}

func getGethELContainerConfigSupplier(srcExecutionDataDirpath string, networkId string, externalIpAddress string, bootnodeEnode string) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	result := func(privateIpAddr string, sharedPath *services.SharedPath) (*services.ContainerConfig, error) {
		executionDataDirpathOnServiceContainerSharedPath := sharedPath.GetChildPath(executionDataDirname)
		destExecutionDataDirpath := executionDataDirpathOnServiceContainerSharedPath.GetAbsPathOnThisContainer()
		if err := recursive_copy.Copy(srcExecutionDataDirpath, destExecutionDataDirpath, recursive_copy.Options{AddPermission: executionDataDirPerms}); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the initialized execution data directory from '%v' to '%v'",
				srcExecutionDataDirpath,
				destExecutionDataDirpath,
			)
		}

		entrypointArgs := []string{"sh", "-c"}
		commandStr := fmt.Sprintf(
			"ln -s '%v' '%v' && " +

		)

		gethCmd := []string{
			"geth",
			"--datadir=" + executionDataDirpathOnServiceContainerSharedPath.GetAbsPathOnServiceContainer(),
			"--networkid=" + networkId,
			"--catalyst",
			"--http",
			"--http.api",
			"engine,net,eth",
			"--ws",
			"--ws.api",
			"engine,net,eth",
			"--allow-insecure-unlock",
			"--nat",
			"extip:" + externalIpAddress,
			"--bootnodes=" + bootnodeEnode,
		}

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return result
}
