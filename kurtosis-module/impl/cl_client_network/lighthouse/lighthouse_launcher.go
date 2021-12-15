package lighthouse

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"strings"
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

	// To start a bootnode rather than a child node, we provide this string to the launchNode function
	bootnodeEnrStrForStartingBootnode = ""
)
var usedPorts = map[string]*services.PortSpec{
	enrTcpPortID: services.NewPortSpec(enrPortNum, services.PortProtocol_TCP),
	enrUdpPortID: services.NewPortSpec(enrPortNum, services.PortProtocol_UDP),
	httpPortID:   services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

type LighthouseCLClientLauncher struct {
	// The dirpath on the module container where the config data directory exists
	configDataDirpathOnModuleContainer string
}

func NewLighthouseCLClientLauncher(configDataDirpathOnModuleContainer string) *LighthouseCLClientLauncher {
	return &LighthouseCLClientLauncher{configDataDirpathOnModuleContainer: configDataDirpathOnModuleContainer}
}

func (launcher *LighthouseCLClientLauncher) LaunchBootNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	elClientRpcSockets map[string]bool,
	totalTerminalDifficulty uint32,
) (
	resultClientCtx *cl_client_network.ConsensusLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, bootnodeEnrStrForStartingBootnode, elClientRpcSockets, totalTerminalDifficulty)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting boot Ligthhouse node with service ID '%v'", serviceId)
	}
	return clientCtx, nil
}

func (launcher *LighthouseCLClientLauncher) LaunchChildNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeEnr string,
	elClientRpcSockets map[string]bool,
	totalTerminalDifficulty uint32,
) (
	resultClientCtx *cl_client_network.ConsensusLayerClientContext,
	resultErr error,
) {
	clientCtx, err := launcher.launchNode(enclaveCtx, bootnodeEnr, elClientRpcSockets, totalTerminalDifficulty)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting child Lighthouse node with service ID '%v' connected to boot node with ENR '%v'", serviceId, bootnodeEnr)
	}
	return clientCtx, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *LighthouseCLClientLauncher) launchNode(
	enclaveCtx *enclaves.EnclaveContext,
	bootnodeEnr string,
	elClientRpcSockets map[string]bool,
	totalTerminalDiffulty uint32,
) (
	resultClientCtx *cl_client_network.ConsensusLayerClientContext,
	resultErr error,
) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(bootnodeEnr, elClientRpcSockets, totalTerminalDiffulty)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lighthouse CL client with service ID '%v'", serviceId)
	}

	// TODO FIGURE OUT HOW TO GET ENODE FROM CL CLIENT
	enode := "some-enode"
	/*
	nodeInfo, err := getNodeInfoWithRetry(serviceCtx.GetPrivateIPAddress())
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the newly-started node's info")
	}

	 */

	result := cl_client_network.NewConsensusLayerClientContext(
		serviceCtx,
		enode,
	)

	return result, nil
}

func (launcher *LighthouseCLClientLauncher) getContainerConfigSupplier(
	bootNodeEnr string,
	elClientRpcSockets map[string]bool,
	totalTerminalDiffulty uint32,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	return func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		configDataDirpathOnServiceSharedPath := sharedDir.GetChildPath(configDataDirpathRelToSharedDirRoot)

		destConfigDataDirpathOnModule := configDataDirpathOnServiceSharedPath.GetAbsPathOnThisContainer()
		if err := recursive_copy.Copy(launcher.configDataDirpathOnModuleContainer, destConfigDataDirpathOnModule); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the config data directory on the module, '%v', into the service container, '%v'",
				launcher.configDataDirpathOnModuleContainer,
				destConfigDataDirpathOnModule,
			)
		}

		elClientRpcUrls := []string{}
		for rpcSocketStr := range elClientRpcSockets {
			rpcUrlStr := fmt.Sprintf("http://%v", rpcSocketStr)
			elClientRpcUrls = append(elClientRpcUrls, rpcUrlStr)
		}
		elClientRpcUrlsStr := strings.Join(elClientRpcUrls, ",")

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
			"--http",
			"--http-port=4000",
			"--merge",
			"--http-allow-sync-stalled",
			"--disable-packet-filter",
			"--execution-endpoints=" + elClientRpcUrlsStr,
			"--eth1-endpoints=" + elClientRpcUrlsStr,
			fmt.Sprintf("--terminal-total-difficulty-override=%v", totalTerminalDiffulty),
		}
		if bootNodeEnr != bootnodeEnrStrForStartingBootnode {
			cmdArgs = append(cmdArgs, "--boot-nodes=" + bootNodeEnr)
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
