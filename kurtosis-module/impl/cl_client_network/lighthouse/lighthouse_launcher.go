package lighthouse

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network/cl_client_rest_client"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"strings"
	"time"
)

const (
	imageName = "sigp/lighthouse:latest-unstable"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	configDataDirpathRelToSharedDirRoot = "config-data"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000

	// To start a bootnode rather than a child node, we provide this string to the launchNode function
	bootnodeEnrStrForStartingBootnode = ""

	maxNumHealthcheckRetries = 10
	timeBetweenHealthcheckRetries = 1 * time.Second
)
var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
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
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, bootnodeEnrStrForStartingBootnode, elClientRpcSockets, totalTerminalDifficulty)
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
	clientCtx, err := launcher.launchNode(enclaveCtx, serviceId, bootnodeEnr, elClientRpcSockets, totalTerminalDifficulty)
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
	serviceId services.ServiceID,
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

	httpPort, found := serviceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lighthouse service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client.NewCLClientRESTClient(serviceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := waitForAvailability(restClient); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lighthouse node to become available")
	}

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lighthouse node's identity, which is necessary to retrieve its ENR")
	}

	result := cl_client_network.NewConsensusLayerClientContext(
		serviceCtx,
		nodeIdentity.ENR,
		httpPortID,
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
		// NOTE: If connecting to the merge devnet remotely we DON'T want the following flags; when they're not set, the node's external IP address is auto-detected
		//  from the peers it communicates with but when they're set they basically say "override the autodetection and
		//  use what I specify instead." This requires having a know external IP address and port, which we definitely won't
		//  have with a network running in Kurtosis.
		//    "--disable-enr-auto-update",
		//    "--enr-address=" + externalIpAddress,
		//    fmt.Sprintf("--enr-udp-port=%v", discoveryPortNum),
		//    fmt.Sprintf("--enr-tcp-port=%v", discoveryPortNum),
		cmdArgs := []string{
			"lighthouse",
			"--debug-level=info",
			"--datadir=" + consensusDataDirpathOnServiceContainer,
			"--testnet-dir=" + configDataDirpathOnService,
			"bn",
			"--eth1",
			// vvvvvvvvvvvvvvvvvvv REMOVE THESE WHEN CONNECTING TO EXTERNAL NET vvvvvvvvvvvvvvvvvvvvv
			"--disable-enr-auto-update",
			"--enr-address=" + privateIpAddr,
			fmt.Sprintf("--enr-udp-port=%v", discoveryPortNum),
			fmt.Sprintf("--enr-tcp-port=%v", discoveryPortNum),
			// ^^^^^^^^^^^^^^^^^^^ REMOVE THESE WHEN CONNECTING TO EXTERNAL NET ^^^^^^^^^^^^^^^^^^^^^
			"--listen-address=0.0.0.0",
			fmt.Sprintf("--port=%v", discoveryPortNum), // NOTE: Remove for connecting to external net!
			"--http",
			"--http-address=0.0.0.0",
			fmt.Sprintf("--http-port=%v", httpPortNum),
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

func waitForAvailability(restClient *cl_client_rest_client.CLClientRESTClient) error {
	for i := 0; i < maxNumHealthcheckRetries; i++ {
		_, err := restClient.GetHealth()
		if err == nil {
			// TODO check the healthstatus???
			return nil
		}
		time.Sleep(timeBetweenHealthcheckRetries)
	}
	return stacktrace.NewError(
		"Lighthouse node didn't become available even after %v retries with %v between retries",
		maxNumHealthcheckRetries,
		timeBetweenHealthcheckRetries,
	)
}
