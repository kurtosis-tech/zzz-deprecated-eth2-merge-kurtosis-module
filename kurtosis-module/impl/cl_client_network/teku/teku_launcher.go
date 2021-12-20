package teku

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"strings"
	"time"
)

const (
	imageName = "consensys/teku:latest"

	// The Docker container runs as the "teku" user so we can't write to root
	consensusDataDirpathOnServiceContainer = "/opt/teku/consensus-data"

	// TODO Get rid of this being hardcoded; should be shared
	validatingRewardsAccount = "0x0000000000000000000000000000000000000001"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000

	// To start a bootnode rather than a child node, we provide this string to the launchNode function
	bootnodeEnrStrForStartingBootnode = ""

	genesisConfigYmlRelFilepathInSharedDir = "genesis-config.yml"
	genesisSszRelFilepathInSharedDir = "genesis.ssz"

	// Teku nodes take quite a while to start
	maxNumHealthcheckRetries = 60
	timeBetweenHealthcheckRetries = 1 * time.Second
)
var usedPorts = map[string]*services.PortSpec{
	// TODO Add metrics port
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

type TekuCLClientLauncher struct {
	genesisConfigYmlFilepathOnModuleContainer string
	genesisSszFilepathOnModuleContainer string
}

func NewTekuCLClientLauncher(genesisConfigYmlFilepathOnModuleContainer string, genesisSszFilepathOnModuleContainer string) *TekuCLClientLauncher {
	return &TekuCLClientLauncher{genesisConfigYmlFilepathOnModuleContainer: genesisConfigYmlFilepathOnModuleContainer, genesisSszFilepathOnModuleContainer: genesisSszFilepathOnModuleContainer}
}

func (launcher *TekuCLClientLauncher) LaunchBootNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	elClientRpcSockets map[string]bool,
	nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl_client_network.ConsensusLayerClientContext, resultErr error) {
	clientCtx, err := launcher.launchNode(
		enclaveCtx,
		serviceId,
		bootnodeEnrStrForStartingBootnode,
		elClientRpcSockets,
		nodeKeystoreDirpaths.TekuKeysDirpath,
		nodeKeystoreDirpaths.TekuSecretsDirpath,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting boot Teku node with service ID '%v'", serviceId)
	}
	return clientCtx, nil
}

func (launcher *TekuCLClientLauncher) LaunchChildNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeEnr string,
	elClientRpcSockets map[string]bool,
	nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl_client_network.ConsensusLayerClientContext, resultErr error) {
	clientCtx, err := launcher.launchNode(
		enclaveCtx,
		serviceId,
		bootnodeEnr,
		elClientRpcSockets,
		nodeKeystoreDirpaths.TekuKeysDirpath,
		nodeKeystoreDirpaths.TekuSecretsDirpath,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred starting child Teku node with service ID '%v' connected to boot node with ENR '%v'", serviceId, bootnodeEnr)
	}
	return clientCtx, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *TekuCLClientLauncher) launchNode(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeEnr string,
	elClientRpcSockets map[string]bool,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
) (
	resultClientCtx *cl_client_network.ConsensusLayerClientContext,
	resultErr error,
) {
	containerConfigSupplier := getContainerConfigSupplier(
		bootnodeEnr,
		elClientRpcSockets,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		launcher.genesisSszFilepathOnModuleContainer,
		validatorKeysDirpathOnModuleContainer,
		validatorSecretsDirpathOnModuleContainer,
	)
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

func getContainerConfigSupplier(
	bootNodeEnr string,
	elClientRpcSockets map[string]bool,
	genesisConfigYmlFilepathOnModuleContainer string,
	genesisSszFilepathOnModuleContainer string,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		genesisConfigYmlSharedPath := sharedDir.GetChildPath(genesisConfigYmlRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisConfigYmlFilepathOnModuleContainer, genesisConfigYmlSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis config YML from '%v' to shared dir relative path '%v'",
				genesisConfigYmlFilepathOnModuleContainer,
				genesisConfigYmlRelFilepathInSharedDir,
			)
		}

		genesisSszSharedPath := sharedDir.GetChildPath(genesisSszRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisSszFilepathOnModuleContainer, genesisSszSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis SSZ from '%v' to shared dir relative path '%v'",
				genesisSszFilepathOnModuleContainer,
				genesisSszRelFilepathInSharedDir,
			)
		}

		elClientRpcUrls := []string{}
		for rpcSocketStr := range elClientRpcSockets {
			rpcUrlStr := fmt.Sprintf("http://%v", rpcSocketStr)
			elClientRpcUrls = append(elClientRpcUrls, rpcUrlStr)
		}
		elClientRpcUrlsStr := strings.Join(elClientRpcUrls, ",")

		cmdArgs := []string{
			"--network=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--initial-state=" + genesisSszSharedPath.GetAbsPathOnServiceContainer(),
			"--data-path=" + consensusDataDirpathOnServiceContainer,
			"--data-storage-mode=PRUNE",
			"--p2p-enabled=true",
			"--eth1-endpoints=" + elClientRpcUrlsStr,
			"--Xee-endpoint=" + elClientRpcUrlsStr,
			"--p2p-advertised-ip=" + privateIpAddr,
			"--rest-api-enabled=true",
			"--rest-api-docs-enabled=true",
			"--rest-api-interface=0.0.0.0",
			fmt.Sprintf("--rest-api-port=%v", httpPortNum),
			"--rest-api-host-allowlist=*",
			"--data-storage-non-canonical-blocks-enabled=true",
			"--log-destination=CONSOLE",
			fmt.Sprintf(
				"--validator-keys=%v:%v",
				validatorKeysDirpathOnModuleContainer,
				validatorSecretsDirpathOnModuleContainer,
			),
			"--Xvalidators-suggested-fee-recipient-address=" + validatingRewardsAccount,
		}
		if bootNodeEnr != bootnodeEnrStrForStartingBootnode {
			cmdArgs = append(cmdArgs, "--p2p-discovery-bootnodes=" + bootNodeEnr)
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
		"Teku node didn't become available even after %v retries with %v between retries",
		maxNumHealthcheckRetries,
		timeBetweenHealthcheckRetries,
	)
}