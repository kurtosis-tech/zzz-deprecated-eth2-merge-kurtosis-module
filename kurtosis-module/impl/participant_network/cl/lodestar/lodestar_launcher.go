package lodestar

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
	"github.com/sirupsen/logrus"
	"path"
	"time"
)

const (
	imageName = "chainsafe/lodestar:next"

	consensusDataDirpathOnServiceContainer = "/consensus-data"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000


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
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
}

type LodestarClientLauncher struct {
	genesisConfigYmlFilepathOnModuleContainer string
	genesisSszFilepathOnModuleContainer       string
}

func NewLodestarCLClientLauncher(genesisConfigYmlFilepathOnModuleContainer string, genesisSszFilepathOnModuleContainer string) *LodestarClientLauncher {
	return &LodestarClientLauncher{genesisConfigYmlFilepathOnModuleContainer: genesisConfigYmlFilepathOnModuleContainer, genesisSszFilepathOnModuleContainer: genesisSszFilepathOnModuleContainer}
}

func (launcher *LodestarClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *prelaunch_data_generator.NodeTypeKeystoreDirpaths,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconServiceId := serviceId + "-" + beaconSuffixServiceId
	validatorServiceId := serviceId + "-" + validatorSuffixServiceId

	beaconContainerConfigSupplier := getBeaconContainerConfigSupplier(
		bootnodeContext,
		elClientContext,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		launcher.genesisSszFilepathOnModuleContainer,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconServiceId, beaconContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lodestar CL beacon client with service ID '%v'", serviceId)
	}

	httpPort, found := beaconServiceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lodestar beacon service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := waitForAvailability(restClient); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lodestar beacon node to become available")
	}

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lodestar beacon node's identity, which is necessary to retrieve its ENR")
	}

	beaconHttpUrl := fmt.Sprintf("http://%v:%v", beaconServiceCtx.GetPrivateIPAddress(), httpPortNum)

	validatorContainerConfigSupplier := getValidatorContainerConfigSupplier(
		validatorServiceId,
		beaconHttpUrl,
		launcher.genesisConfigYmlFilepathOnModuleContainer,
		nodeKeystoreDirpaths.RawKeysDirpath,
		nodeKeystoreDirpaths.LodestarSecretsDirpath,
	)
	_, err = enclaveCtx.AddService(validatorServiceId, validatorContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lodestar CL validator client with service ID '%v'", serviceId)
	}

	if err := waitForSync(restClient); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lodestar validator node to become synced with the beacon node")
	}

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		httpPortNum,
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
			"beacon",
			fmt.Sprintf("--port=%v", discoveryPortNum),
			fmt.Sprintf("--discoveryPort=%v", discoveryPortNum),
			"--rootDir=" + consensusDataDirpathOnServiceContainer,
			"--paramsFile=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--genesisStateFile=" + genesisSszSharedPath.GetAbsPathOnServiceContainer(),
			"--network.connectToDiscv5Bootnodes=true",
			"--network.discv5.enabled=true",
			"--eth1.enabled=true",
			"--eth1.disableEth1DepositDataTracker=true",
			"--eth1.providerUrls=" + elClientRpcUrlStr,
			"--execution.urls=" + elClientRpcUrlStr,
			"--api.rest.enabled=true",
			"--api.rest.host=0.0.0.0",
			"--api.rest.api=*",
			fmt.Sprintf("--api.rest.port=%v", httpPortNum),
			"--enr.ip=" + privateIpAddr,
			fmt.Sprintf("--enr.tcp=%v", discoveryPortNum),
			fmt.Sprintf("--enr.udp=%v", discoveryPortNum),
			"--logLevel=debug",

		}
		if bootnodeContext != nil {
			cmdArgs = append(cmdArgs, "--network.discv5.bootEnrs=" + bootnodeContext.GetENR())
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

		logrus.Infof("Lodestar Keystore Dirpath: %v", validatorKeysDirpathOnModuleContainer)
		logrus.Infof("Lodestar Secrets Dirpath: %v", validatorSecretsDirpathOnModuleContainer)

		cmdArgs := []string{
			"validator",
			"--rootDir=" + rootDirpath,
			"--paramsFile=" + genesisConfigYmlSharedPath.GetAbsPathOnServiceContainer(),
			"--server=" + beaconEndpoint,
			"--keystoresDir=" + validatorKeysDirpathOnModuleContainer,
			"--secretsDir=" + validatorSecretsDirpathOnModuleContainer,
			"--logLevel=debug",
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
		"Lodestar node didn't become available even after %v retries with %v between retries",
		maxNumHealthcheckRetries,
		timeBetweenHealthcheckRetries,
	)
}

func waitForSync(restClient *cl_client_rest_client.CLClientRESTClient) error {
	for i := 0; i < maxNumSyncCheckRetries; i++ {
		syncingData, err := restClient.GetNodeSyncingData()
		if err == nil && syncingData.IsSyncing {
			return nil
		}
		time.Sleep(timeBetweenSyncCheckRetries)
	}
	return stacktrace.NewError(
		"Lodestar validator node didn't become syncing with beacon node even after %v retries with %v between retries",
		maxNumSyncCheckRetries,
		timeBetweenSyncCheckRetries,
	)
}
