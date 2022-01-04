package prysm

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"time"
)

const (
	beaconNodeImageName    = "prysmaticlabs/prysm-beacon-chain:stable"
	validatorNodeImageName = "prysmaticlabs/prysm-validator:stable"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"

	// Port nums
	discoveryTCPPortNum uint16 = 13000
	discoveryUDPPortNum uint16 = 12000
	httpPortNum         uint16 = 3500

	genesisConfigYmlRelFilepathInSharedDir = "genesis-config.yml"
	genesisSszRelFilepathInSharedDir       = "genesis.ssz"

	maxNumHealthcheckRetries      = 20
	timeBetweenHealthcheckRetries = 1 * time.Second

	maxNumSyncCheckRetries      = 30
	timeBetweenSyncCheckRetries = 1 * time.Second

	beaconSuffixServiceId    = "beacon"
	validatorSuffixServiceId = "validator"
)

var usedPorts = map[string]*services.PortSpec{
	// TODO Add metrics port
	tcpDiscoveryPortID: services.NewPortSpec(discoveryTCPPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryUDPPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

type PrysmClientLauncher struct {
	genesisConfigYmlFilepathOnModuleContainer string
	genesisSszFilepathOnModuleContainer       string
}

func NewPrysmCLCLientLauncher(genesisConfigYmlFilepathOnModuleContainer string, genesisSszFilepathOnModuleContainer string) *PrysmClientLauncher {
	return &PrysmClientLauncher{genesisConfigYmlFilepathOnModuleContainer: genesisConfigYmlFilepathOnModuleContainer, genesisSszFilepathOnModuleContainer: genesisSszFilepathOnModuleContainer}
}

func (launcher *PrysmClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconServiceId := serviceId + "-" + beaconSuffixServiceId
	//validatorServiceId := serviceId + "-" + validatorSuffixServiceId

	beaconContainerConfigSupplier := getBeaconContainerConfigSupplier(
		bootnodeContext,
		elClientContext,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		launcher.genesisSszFilepathOnModuleContainer,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconServiceId, beaconContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Prysm CL beacon client with service ID '%v'", serviceId)
	}

	httpPort, found := beaconServiceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Prysm beacon service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := waitForAvailability(restClient); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Prysm beacon node to become available")
	}

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Prysm beacon node's identity, which is necessary to retrieve its ENR")
	}

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		httpPortNum,
		restClient,
	)

	return result, nil

}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func getBeaconContainerConfigSupplier(
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	genesisConfigYmlFilepathOnModuleContainer string,
	genesisSszFilepathOnModuleContainer string,
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

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetRPCPortNum(),
		)

		cmdArgs := []string{
			"--accept-terms-of-use=true", //it's mandatory in order to run the node
			"--prater",                   //it's a tesnet setup, it's mandatory to set a network (https://docs.prylabs.network/docs/install/install-with-script#before-you-begin-pick-your-network-1)
			"--datadir=" + consensusDataDirpathOnServiceContainer,
			"--chain-config-file=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--genesis-state=" + genesisSszSharedPath.GetAbsPathOnServiceContainer(),
			"--http-web3provider=" + elClientRpcUrlStr,
			"--http-modules=prysm,eth",
			"--rpc-host=" + privateIpAddr,
			fmt.Sprintf("--rpc-port=%v", discoveryTCPPortNum),
			"--grpc-gateway-host=0.0.0.0",
			fmt.Sprintf("--grpc-gateway-port=%v", httpPortNum),
			//"--monitoring-host=0.0.0.0",
			//fmt.Sprintf("--monitoring-port=%v", httpPortNum),
			fmt.Sprintf("--p2p-tcp-port=%v", discoveryTCPPortNum),
			fmt.Sprintf("--p2p-udp-port=%v", discoveryUDPPortNum),
		}
		if bootnodeContext != nil {
			cmdArgs = append(cmdArgs, "--peer="+bootnodeContext.GetENR())
		}

		containerConfig := services.NewContainerConfigBuilder(
			beaconNodeImageName,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return containerConfigSupplier
}

func getValidatorContainerConfigSupplier(
	serviceId services.ServiceID,
	beaconEndpoint string,
	genesisConfigYmlFilepathOnModuleContainer string,
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

		rootDirpath := path.Join(consensusDataDirpathOnServiceContainer, string(serviceId))

		cmdArgs := []string{
			"--accept-terms-of-use=true", //it's mandatory in order to run the node
			"--prater",                   //it's a tesnet setup, it's mandatory to set a network (https://docs.prylabs.network/docs/install/install-with-script#before-you-begin-pick-your-network-1)
			"--datadir=" + rootDirpath,
			"--chain-config-file=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--beacon-rpc-gateway-provider=" + beaconEndpoint,
			"--grpc-gateway-host=0.0.0.0",
			fmt.Sprintf("--grpc-gateway-port=%v", httpPortNum),
			"--monitoring-host=0.0.0.0",
			fmt.Sprintf("--monitoring-port=%v", httpPortNum),
			"--enable-doppelganger=false",
			/*"validator",
			"--rootDir=" + rootDirpath,
			"--paramsFile=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--server=" + beaconEndpoint,
			"--keystoresDir=" + validatorKeysDirpathOnModuleContainer,
			"--secretsDir=" + validatorSecretsDirpathOnModuleContainer,
			"--logLevel=debug",*/
		}

		containerConfig := services.NewContainerConfigBuilder(
			validatorNodeImageName,
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
		"Prysm node didn't become available even after %v retries with %v between retries",
		maxNumHealthcheckRetries,
		timeBetweenHealthcheckRetries,
	)
}
