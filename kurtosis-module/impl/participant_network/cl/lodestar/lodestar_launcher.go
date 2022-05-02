package lodestar

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_genesis"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"time"
)

const (
	consensusDataDirpathOnServiceContainer = "/consensus-data"
	genesisDataMountDirpathOnServiceContainer = "/genesis"

	// Port IDs
	tcpDiscoveryPortID = "tcpDiscovery"
	udpDiscoveryPortID = "udpDiscovery"
	httpPortID         = "http"
	metricsPortID      = "metrics"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000
	metricsPortNum   uint16 = 8008

	genesisConfigYmlRelFilepathInSharedDir = "genesis-config.yml"
	genesisSszRelFilepathInSharedDir       = "genesis.ssz"
	jwtSecretRelFilepathInSharedDir        = "jwtsecret"

	maxNumHealthcheckRetries      = 30
	timeBetweenHealthcheckRetries = 2 * time.Second

	beaconSuffixServiceId    = "beacon"
	validatorSuffixServiceId = "validator"

	metricsPath = "/metrics"
)

var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	metricsPortID:      services.NewPortSpec(metricsPortNum, services.PortProtocol_TCP),
}
var lodestarLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "error",
	module_io.GlobalClientLogLevel_Warn:  "warn",
	module_io.GlobalClientLogLevel_Info:  "info",
	module_io.GlobalClientLogLevel_Debug: "debug",
	module_io.GlobalClientLogLevel_Trace: "silly",
}

type LodestarClientLauncher struct {
	genesisData *cl_genesis.CLGenesisData
}

func NewLodestarClientLauncher(genesisData *cl_genesis.CLGenesisData) *LodestarClientLauncher {
	return &LodestarClientLauncher{genesisData: genesisData}
}

func (launcher *LodestarClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	nodeKeystoreDirpaths *cl2.NodeTypeKeystoreDirpaths,
	extraBeaconParams []string,
	extraValidatorParams []string,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	beaconNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, beaconSuffixServiceId))
	validatorNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, validatorSuffixServiceId))

	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, lodestarLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	beaconContainerConfigSupplier := launcher.getBeaconContainerConfigSupplier(
		image,
		bootnodeContext,
		elClientContext,
		logLevel,
		extraBeaconParams,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconNodeServiceId, beaconContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lodestar CL beacon client with service ID '%v'", serviceId)
	}

	httpPort, found := beaconServiceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lodestar beacon service to have port with ID '%v', but none was found", httpPortID)
	}

	beaconRestClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), httpPort.GetNumber())
	if err := cl.WaitForBeaconClientAvailability(beaconRestClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Lodestar beacon node to become available")
	}

	nodeIdentity, err := beaconRestClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Lodestar beacon node's identity, which is necessary to retrieve its ENR")
	}

	beaconHttpUrl := fmt.Sprintf("http://%v:%v", beaconServiceCtx.GetPrivateIPAddress(), httpPortNum)

	validatorContainerConfigSupplier := launcher.getValidatorContainerConfigSupplier(
		validatorNodeServiceId,
		image,
		logLevel,
		beaconHttpUrl,
		nodeKeystoreDirpaths.RawKeysDirpath,
		nodeKeystoreDirpaths.LodestarSecretsDirpath,
		extraValidatorParams,
	)
	_, err = enclaveCtx.AddService(validatorNodeServiceId, validatorContainerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Lodestar CL validator client with service ID '%v'", serviceId)
	}

	metricsPort, found := beaconServiceCtx.GetPrivatePorts()[metricsPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Lodestar service to have port with ID '%v', but none was found", metricsPortID)
	}
	metricsUrl := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), metricsPort.GetNumber())

	nodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(serviceId), metricsPath, metricsUrl)
	nodesMetricsInfo := []*cl.CLNodeMetricsInfo{nodeMetricsInfo}

	result := cl.NewCLClientContext(
		"lodestar",
		nodeIdentity.ENR,
		beaconServiceCtx.GetPrivateIPAddress(),
		httpPortNum,
		nodesMetricsInfo,
		beaconRestClient,
	)

	return result, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *LodestarClientLauncher) getBeaconContainerConfigSupplier(
	image string,
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	logLevel string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		/*
		genesisConfigYmlSharedPath := sharedDir.GetChildPath(genesisConfigYmlRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisConfigYmlFilepathOnModuleContainer, genesisConfigYmlSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis config YML from '%v' to shared dir relative path '%v'",
				launcher.genesisConfigYmlFilepathOnModuleContainer,
				genesisConfigYmlRelFilepathInSharedDir,
			)
		}

		genesisSszSharedPath := sharedDir.GetChildPath(genesisSszRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.genesisSszFilepathOnModuleContainer, genesisSszSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis SSZ from '%v' to shared dir relative path '%v'",
				launcher.genesisSszFilepathOnModuleContainer,
				genesisSszRelFilepathInSharedDir,
			)
		}

		jwtSecretSharedPath := sharedDir.GetChildPath(jwtSecretRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(launcher.jwtSecretFilepathOnModuleContainer, jwtSecretSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying JWT secret file '%v' into shared directory path '%v'", launcher.jwtSecretFilepathOnModuleContainer, jwtSecretRelFilepathInSharedDir)
		}

		 */

		elClientRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetRPCPortNum(),
		)

		elClientEngineRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetEngineRPCPortNum(),
		)

		genesisConfigFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetConfigYMLRelativeFilepath())
		genesisSszFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetGenesisSSZRelativeFilepath())
		jwtSecretFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetJWTSecretRelativeFilepath())
		cmdArgs := []string{
			"beacon",
			"--logLevel=" + logLevel,
			fmt.Sprintf("--port=%v", discoveryPortNum),
			fmt.Sprintf("--discoveryPort=%v", discoveryPortNum),
			"--rootDir=" + consensusDataDirpathOnServiceContainer,
			"--paramsFile=" + genesisConfigFilepath,
			"--genesisStateFile=" + genesisSszFilepath,
			"--eth1.depositContractDeployBlock=0",
			"--network.connectToDiscv5Bootnodes=true",
			"--network.discv5.enabled=true",
			"--eth1.enabled=true",
			"--eth1.providerUrls=" + elClientRpcUrlStr,
			"--execution.urls=" + elClientEngineRpcUrlStr,
			"--api.rest.enabled=true",
			"--api.rest.host=0.0.0.0",
			"--api.rest.api=*",
			fmt.Sprintf("--api.rest.port=%v", httpPortNum),
			"--enr.ip=" + privateIpAddr,
			fmt.Sprintf("--enr.tcp=%v", discoveryPortNum),
			fmt.Sprintf("--enr.udp=%v", discoveryPortNum),
			// Set per Pari's recommendation to reduce noise in the logs
			"--network.subscribeAllSubnets=true",
			fmt.Sprintf("--jwt-secret=%v", jwtSecretFilepath),
			// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
			"--metrics.enabled",
			"--metrics.listenAddr=" + privateIpAddr,
			fmt.Sprintf("--metrics.serverPort=%v", metricsPortNum),
			// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
		}
		if bootnodeContext != nil {
			cmdArgs = append(cmdArgs, "--network.discv5.bootEnrs="+bootnodeContext.GetENR())
		}
		if len(extraParams) > 0 {
			cmdArgs = append(cmdArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return containerConfigSupplier
}

func (launcher *LodestarClientLauncher) getValidatorContainerConfigSupplier(
	serviceId services.ServiceID,
	image string,
	logLevel string,
	beaconEndpoint string,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
	extraParams []string,
) func(string, *services.SharedPath) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		/*
		genesisConfigYmlSharedPath := sharedDir.GetChildPath(genesisConfigYmlRelFilepathInSharedDir)
		if err := service_launch_utils.CopyFileToSharedPath(genesisConfigYmlFilepathOnModuleContainer, genesisConfigYmlSharedPath); err != nil {
			return nil, stacktrace.Propagate(
				err,
				"An error occurred copying the genesis config YML from '%v' to shared dir relative path '%v'",
				genesisConfigYmlFilepathOnModuleContainer,
				genesisConfigYmlRelFilepathInSharedDir,
			)
		}

		 */

		rootDirpath := path.Join(consensusDataDirpathOnServiceContainer, string(serviceId))

		genesisConfigFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetConfigYMLRelativeFilepath())
		cmdArgs := []string{
			"validator",
			"--logLevel=" + logLevel,
			"--rootDir=" + rootDirpath,
			"--paramsFile=" + genesisConfigFilepath,
			"--server=" + beaconEndpoint,
			"--keystoresDir=" + validatorKeysDirpathOnModuleContainer,
			"--secretsDir=" + validatorSecretsDirpathOnModuleContainer,
		}
		if len(cmdArgs) > 0 {
			cmdArgs = append(cmdArgs, extraParams...)
		}

		containerConfig := services.NewContainerConfigBuilder(
			image,
		).WithUsedPorts(
			usedPorts,
		).WithCmdOverride(
			cmdArgs,
		).Build()

		return containerConfig, nil
	}
	return containerConfigSupplier
}
