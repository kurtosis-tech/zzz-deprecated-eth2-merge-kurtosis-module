package lighthouse_cl_client

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
)

const (
	serviceId services.ServiceID = "lighthouse-cl-client"
	imageName = "sigp/lighthouse:latest-unstable"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	configDataDirpathRelToSharedDirRoot = "config-data"

	// Port IDs
	enrTcpPortID = "enr-tcp"
	enrUdpPortID = "enr-udp"
	httpPortID = "http"

	// Port nums
	enrPortNum uint16 = 9000
	httpPortNum = 4000
)
var usedPorts = map[string]*services.PortSpec{
	enrTcpPortID: services.NewPortSpec(enrPortNum, services.PortProtocol_TCP),
	enrUdpPortID: services.NewPortSpec(enrPortNum, services.PortProtocol_UDP),
	httpPortID: services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

func LaunchLighthouseCLClient(
	enclaveCtx *enclaves.EnclaveContext,
	srcConfigDataDirpathOnModule string,
	externalIpAddress string,
	bootNodeEnr string,
	elClientPrivateIpAddr string,
	elClientPrivateRpcPortNum uint16,
	totalTerminalDiffulty uint,
) (
	*services.ServiceContext,
	error,
) {
	containerConfigSupplier := getContainerConfigSupplier(
		srcConfigDataDirpathOnModule,
		externalIpAddress,
		bootNodeEnr,
		elClientPrivateIpAddr,
		elClientPrivateRpcPortNum,
		totalTerminalDiffulty,
	)

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred adding the Lighthouse CL client with ID '%v' to the network", serviceId)
	}

	return serviceCtx, nil
}

func getContainerConfigSupplier(
	srcConfigDataDirpathOnModule string,
	externalIpAddress string,
	bootNodeEnr string,
	elClientPrivateIpAddr string,
	elClientPrivateRpcPortNum uint16,
	totalTerminalDiffulty uint,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		configDataDirpathOnServiceSharedPath := sharedDir.GetChildPath(configDataDirpathRelToSharedDirRoot)

		destConfigDataDirpathOnModule := configDataDirpathOnServiceSharedPath.GetAbsPathOnThisContainer()
		if err := recursive_copy.Copy(srcConfigDataDirpathOnModule, destConfigDataDirpathOnModule); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the config data directory on the module, '%v', into the service container, '%v'",
				srcConfigDataDirpathOnModule,
				destConfigDataDirpathOnModule,
			)
		}

		configDataDirpathOnService := configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer()
		// NOTE: We DON'T want the following flags; when they're not set, the node's external IP address is auto-detected
		//  from the peers it communicates with but when they're set they basically say "override the autodetection and
		//  use what I specify instead." This requires having a know external IP address and port, which we definitely won't
		//  have with a network running in Kurtosis.
		//    "--disable-enr-auto-update",
		//    "--enr-address=" + externalIpAddress,
		//    "--enr-udp-port",
		//    fmt.Sprintf("%v", enrPortNum),
		//    "--enr-tcp-port",
		//    fmt.Sprintf("%v", enrPortNum),
		cmdArgs := []string{
			"lighthouse",
			"--debug-level=info",
			"--datadir=" + consensusDataDirpathOnServiceContainer,
			"--testnet-dir=" + configDataDirpathOnService,
			"bn",
			"--eth1",
			"--boot-nodes=" + bootNodeEnr,
			"--http",
			"--http-port=4000",
			"--merge",
			"--http-allow-sync-stalled",
			"--disable-packet-filter",
			fmt.Sprintf("--execution-endpoints=http://%v:%v", elClientPrivateIpAddr, elClientPrivateRpcPortNum),
			fmt.Sprintf("--eth1-endpoints=http://%v:%v", elClientPrivateIpAddr, elClientPrivateRpcPortNum),
			fmt.Sprintf("--terminal-total-difficulty-override=%v", totalTerminalDiffulty),
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
}
