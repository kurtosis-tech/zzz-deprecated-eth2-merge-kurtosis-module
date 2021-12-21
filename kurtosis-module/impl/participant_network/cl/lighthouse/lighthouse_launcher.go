package lighthouse

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/availability_waiter"
	cl_client_rest_client2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
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

func (launcher *LighthouseCLClientLauncher) Launch(enclaveCtx *enclaves.EnclaveContext, serviceId services.ServiceID, bootnodeContext *cl.CLClientContext, elClientContext *el.ELClientContext, nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths) (resultClientCtx *cl.CLClientContext, resultErr error) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(bootnodeContext, elClientContext)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lighthouse CL client with service ID '%v'", serviceId)
	}

	httpPort, found := serviceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lighthouse service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client2.NewCLClientRESTClient(serviceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := availability_waiter.WaitForCLClientAvailability(restClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lighthouse node to become available")
	}

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lighthouse node's identity, which is necessary to retrieve its ENR")
	}

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		serviceCtx.GetPrivateIPAddress(),
		httpPortNum,
	)

	return result, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *LighthouseCLClientLauncher) getContainerConfigSupplier(
	bootClClientCtx *cl.CLClientContext,
	elClientCtx *el.ELClientContext,
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

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientCtx.GetIPAddress(),
			elClientCtx.GetRPCPortNum(),
		)

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
			"--execution-endpoints=" + elClientRpcUrlStr,
			"--eth1-endpoints=" + elClientRpcUrlStr,
		}
		if bootClClientCtx != nil {
			cmdArgs = append(cmdArgs, "--boot-nodes=" + bootClClientCtx.GetENR())
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
