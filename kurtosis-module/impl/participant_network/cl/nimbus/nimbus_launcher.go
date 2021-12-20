package nimbus

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
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
	// TODO change
	imageName = "parithoshj/nimbus:merge-e623091"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000

	configDataDirpathRelToSharedDirRoot = "config-data"

	maxNumHealthcheckRetries = 15
	timeBetweenHealthcheckRetries = 1 * time.Second
)
var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

type NimbusLauncher struct {
	// The dirpath on the module container where the config data directory exists
	configDataDirpathOnModuleContainer string
}

func NewNimbusLauncher(configDataDirpathOnModuleContainer string) *NimbusLauncher {
	return &NimbusLauncher{configDataDirpathOnModuleContainer: configDataDirpathOnModuleContainer}
}

func (launcher NimbusLauncher) Launch(enclaveCtx *enclaves.EnclaveContext, serviceId services.ServiceID, bootnodeContext *cl.CLClientContext, elClientContext *el.ELClientContext, nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths) (resultClientCtx *cl.CLClientContext, resultErr error) {
	containerConfigSupplier := launcher.getContainerConfigSupplier(
		bootnodeContext,
		elClientContext,
	)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Nimbus CL client with service ID '%v'", serviceId)
	}

	httpPort, found := serviceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Nimbus service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client2.NewCLClientRESTClient(serviceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := waitForAvailability(restClient); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Nimbus node to become available")
	}

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Nimbus node's identity, which is necessary to retrieve its ENR")
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
func (launcher *NimbusLauncher) getContainerConfigSupplier(
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
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

		elClientWsUrlStr := fmt.Sprintf(
			"ws://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetWSPortNum(),
		)

		cmdArgs := []string{
			"beacon_node",
			"--non-interactive=true",
			"--network=" + configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer(),
			"--data-dir=" + consensusDataDirpathOnServiceContainer,
			"--web3-url=" + elClientWsUrlStr,
			"--nat=extip:" + privateIpAddr,
			"--enr-auto-update=false",
			"--rest",
			fmt.Sprintf("--rest-port=%v", httpPortNum),
			// TODO validator keys
		}
		if bootnodeContext != nil {
			cmdArgs = append(cmdArgs, "--bootstrap-node=" + bootnodeContext.GetENR())
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
	return containerConfigSupplier
}

func waitForAvailability(restClient *cl_client_rest_client2.CLClientRESTClient) error {
	for i := 0; i < maxNumHealthcheckRetries; i++ {
		_, err := restClient.GetHealth()
		if err == nil {
			// TODO check the healthstatus???
			return nil
		}
		time.Sleep(timeBetweenHealthcheckRetries)
	}
	return stacktrace.NewError(
		"Nimbus node didn't become available even after %v retries with %v between retries",
		maxNumHealthcheckRetries,
		timeBetweenHealthcheckRetries,
	)
}
