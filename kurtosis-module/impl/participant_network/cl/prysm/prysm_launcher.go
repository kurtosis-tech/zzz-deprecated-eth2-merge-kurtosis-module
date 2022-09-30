package prysm

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/mev_boost"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_genesis"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	imageSeparatorDelimiter = ","
	expectedNumImages       = 2

	consensusDataDirpathOnServiceContainer      = "/consensus-data"
	genesisDataMountDirpathOnServiceContainer   = "/genesis"
	validatorKeysMountDirpathOnServiceContainer = "/validator-keys"
	prysmPasswordMountDirpathOnServiceContainer = "/prysm-password"

	// Port IDs
	tcpDiscoveryPortID        = "tcp-discovery"
	udpDiscoveryPortID        = "udp-discovery"
	rpcPortID                 = "rpc"
	httpPortID                = "http"
	beaconMonitoringPortID    = "monitoring"
	validatorMonitoringPortID = "monitoring"

	// Port nums
	discoveryTCPPortNum        uint16 = 13000
	discoveryUDPPortNum        uint16 = 12000
	rpcPortNum                 uint16 = 4000
	httpPortNum                uint16 = 3500
	beaconMonitoringPortNum    uint16 = 8080
	validatorMonitoringPortNum uint16 = 8081

	maxNumHealthcheckRetries      = 100
	timeBetweenHealthcheckRetries = 5 * time.Second

	beaconSuffixServiceId    = "beacon"
	validatorSuffixServiceId = "validator"

	minPeers = 1

	metricsPath = "/metrics"

	privateIPAddressPlaceholder = "KURTOSIS_PRIVATE_IP_ADDR_PLACEHOLDER"
)

var beaconNodeUsedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID:     services.NewPortSpec(discoveryTCPPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID:     services.NewPortSpec(discoveryUDPPortNum, services.PortProtocol_UDP),
	rpcPortID:              services.NewPortSpec(rpcPortNum, services.PortProtocol_TCP),
	httpPortID:             services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	beaconMonitoringPortID: services.NewPortSpec(beaconMonitoringPortNum, services.PortProtocol_TCP),
}

var validatorNodeUsedPorts = map[string]*services.PortSpec{
	validatorMonitoringPortID: services.NewPortSpec(validatorMonitoringPortNum, services.PortProtocol_TCP),
}
var prysmLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "error",
	module_io.GlobalClientLogLevel_Warn:  "warn",
	module_io.GlobalClientLogLevel_Info:  "info",
	module_io.GlobalClientLogLevel_Debug: "debug",
	module_io.GlobalClientLogLevel_Trace: "trace",
}

type PrysmCLClientLauncher struct {
	genesisData                   *cl_genesis.CLGenesisData
	prysmPasswordArtifactUuid     services.FilesArtifactUUID
	prysmPasswordRelativeFilepath string
}

func NewPrysmCLClientLauncher(genesisData *cl_genesis.CLGenesisData, prysmPasswordArtifactUuid services.FilesArtifactUUID, prysmPasswordRelativeFilepath string) *PrysmCLClientLauncher {
	return &PrysmCLClientLauncher{genesisData: genesisData, prysmPasswordArtifactUuid: prysmPasswordArtifactUuid, prysmPasswordRelativeFilepath: prysmPasswordRelativeFilepath}
}

func (launcher *PrysmCLClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	// NOTE: Because Prysm has separate images for Beacon and validator, this string will actually be a delimited
	//  combination of both Beacon & validator images
	delimitedImagesStr string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	mevBoostContext *mev_boost.MEVBoostContext,
	keystoreFiles *cl2.KeystoreFiles,
	extraBeaconParams []string,
	extraValidatorParams []string,
) (resultClientCtx *cl.CLClientContext, resultErr error) {
	imageStrs := strings.Split(delimitedImagesStr, imageSeparatorDelimiter)
	if len(imageStrs) != expectedNumImages {
		return nil, stacktrace.NewError(
			"Expected Prysm image string '%v' to contain %v images - Beacon and validator - delimited by '%v'",
			delimitedImagesStr,
			expectedNumImages,
			imageSeparatorDelimiter,
		)
	}
	beaconImage := imageStrs[0]
	validatorImage := imageStrs[1]
	if len(strings.TrimSpace(beaconImage)) == 0 {
		return nil, stacktrace.NewError("An empty Prysm Beacon image was provided")
	}
	if len(strings.TrimSpace(validatorImage)) == 0 {
		return nil, stacktrace.NewError("An empty Prysm validator image was provided")
	}

	beaconNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, beaconSuffixServiceId))
	validatorNodeServiceId := services.ServiceID(fmt.Sprintf("%v-%v", serviceId, validatorSuffixServiceId))

	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, prysmLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	beaconContainerConfig := launcher.getBeaconContainerConfig(
		beaconImage,
		bootnodeContext,
		elClientContext,
		mevBoostContext,
		logLevel,
		extraBeaconParams,
	)
	beaconServiceCtx, err := enclaveCtx.AddService(beaconNodeServiceId, beaconContainerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Prysm CL beacon client with service ID '%v'", serviceId)
	}

	httpPort, found := beaconServiceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Prysm beacon service to have port with ID '%v', but none was found", httpPortID)
	}

	beaconRestClient := cl_client_rest_client.NewCLClientRESTClient(beaconServiceCtx.GetPrivateIPAddress(), httpPort.GetNumber())
	if err := cl.WaitForBeaconClientAvailability(beaconRestClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Prysm beacon node to become available")
	}

	nodeIdentity, err := beaconRestClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Prysm beacon node's identity, which is necessary to retrieve its ENR")
	}

	beaconRPCEndpoint := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), rpcPortNum)
	beaconHTTPEndpoint := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), httpPortNum)
	validatorContainerConfig := launcher.getValidatorContainerConfig(
		validatorImage,
		validatorNodeServiceId,
		logLevel,
		beaconRPCEndpoint,
		beaconHTTPEndpoint,
		keystoreFiles,
		mevBoostContext,
		extraValidatorParams,
	)
	validatorServiceCtx, err := enclaveCtx.AddService(validatorNodeServiceId, validatorContainerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Prysm CL validator client with service ID '%v'", serviceId)
	}

	beaconMonitoringPort, found := beaconServiceCtx.GetPrivatePorts()[beaconMonitoringPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Prysm Beacon service to have port with ID '%v', but none was found", beaconMonitoringPortID)
	}
	beaconMetricsUrl := fmt.Sprintf("%v:%v", beaconServiceCtx.GetPrivateIPAddress(), beaconMonitoringPort.GetNumber())

	validatorMonitoringPort, found := validatorServiceCtx.GetPrivatePorts()[validatorMonitoringPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Prysm Validator service to have port with ID '%v', but none was found", validatorMonitoringPortID)
	}
	validatorMetricsUrl := fmt.Sprintf("%v:%v", validatorServiceCtx.GetPrivateIPAddress(), validatorMonitoringPort.GetNumber())

	beaconNodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(beaconNodeServiceId), metricsPath, beaconMetricsUrl)
	validatorNodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(validatorNodeServiceId), metricsPath, validatorMetricsUrl)
	nodesMetricsInfo := []*cl.CLNodeMetricsInfo{beaconNodeMetricsInfo, validatorNodeMetricsInfo}

	result := cl.NewCLClientContext(
		"prysm",
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
func (launcher *PrysmCLClientLauncher) getBeaconContainerConfig(
	beaconImage string,
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	mevBoostContext *mev_boost.MEVBoostContext,
	logLevel string,
	extraParams []string,
) *services.ContainerConfig {
	elClientEngineRpcUrlStr := fmt.Sprintf(
		"http://%v:%v",
		elClientContext.GetIPAddress(),
		elClientContext.GetEngineRPCPortNum(),
	)

	genesisConfigFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetConfigYMLRelativeFilepath())
	genesisSszFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetGenesisSSZRelativeFilepath())
	jwtSecretFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.genesisData.GetJWTSecretRelativeFilepath())
	cmdArgs := []string{
		"--accept-terms-of-use=true", //it's mandatory in order to run the node
		"--datadir=" + consensusDataDirpathOnServiceContainer,
		"--chain-config-file=" + genesisConfigFilepath,
		"--genesis-state=" + genesisSszFilepath,
		"--http-web3provider=" + elClientEngineRpcUrlStr,
		"--rpc-host=" + privateIPAddressPlaceholder,
		fmt.Sprintf("--rpc-port=%v", rpcPortNum),
		"--grpc-gateway-host=0.0.0.0",
		fmt.Sprintf("--grpc-gateway-port=%v", httpPortNum),
		fmt.Sprintf("--p2p-tcp-port=%v", discoveryTCPPortNum),
		fmt.Sprintf("--p2p-udp-port=%v", discoveryUDPPortNum),
		fmt.Sprintf("--min-sync-peers=%v", minPeers),
		"--monitoring-host=" + privateIPAddressPlaceholder,
		fmt.Sprintf("--monitoring-port=%v", beaconMonitoringPortNum),
		"--verbosity=" + logLevel,
		// Set per Pari's recommendation to reduce noise
		"--subscribe-all-subnets=true",
		fmt.Sprintf("--jwt-secret=%v", jwtSecretFilepath),
		// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
		"--disable-monitoring=false",
		"--monitoring-host=" + privateIPAddressPlaceholder,
		fmt.Sprintf("--monitoring-port=%v", beaconMonitoringPortNum),
		// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
	}
	if bootnodeContext != nil {
		cmdArgs = append(cmdArgs, "--bootstrap-node="+bootnodeContext.GetENR())
	}
	if mevBoostContext != nil {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--http-mev-relay=%s", mevBoostContext.Endpoint()))
	}
	if len(extraParams) > 0 {
		cmdArgs = append(cmdArgs, extraParams...)
	}

	containerConfig := services.NewContainerConfigBuilder(
		beaconImage,
	).WithUsedPorts(
		beaconNodeUsedPorts,
	).WithCmdOverride(
		cmdArgs,
	).WithFiles(map[services.FilesArtifactUUID]string{
		launcher.genesisData.GetFilesArtifactUUID(): genesisDataMountDirpathOnServiceContainer,
	}).WithPrivateIPAddrPlaceholder(
		privateIPAddressPlaceholder,
	).Build()

	return containerConfig
}

func (launcher *PrysmCLClientLauncher) getValidatorContainerConfig(
	validatorImage string,
	serviceId services.ServiceID,
	logLevel string,
	beaconRPCEndpoint string,
	beaconHTTPEndpoint string,
	keystoreFiles *cl2.KeystoreFiles,
	mevBoostContext *mev_boost.MEVBoostContext,
	extraParams []string,
) *services.ContainerConfig {
	consensusDataDirpath := path.Join(consensusDataDirpathOnServiceContainer, string(serviceId))
	prysmKeystoreDirpath := path.Join(validatorKeysMountDirpathOnServiceContainer, keystoreFiles.PrysmRelativeDirpath)
	prysmPasswordFilepath := path.Join(prysmPasswordMountDirpathOnServiceContainer, launcher.prysmPasswordRelativeFilepath)

	cmdArgs := []string{
		"--accept-terms-of-use=true", //it's mandatory in order to run the node
		"--prater",                   //it's a tesnet setup, it's mandatory to set a network (https://docs.prylabs.network/docs/install/install-with-script#before-you-begin-pick-your-network-1)
		"--beacon-rpc-gateway-provider=" + beaconHTTPEndpoint,
		"--beacon-rpc-provider=" + beaconRPCEndpoint,
		"--wallet-dir=" + prysmKeystoreDirpath,
		"--wallet-password-file=" + prysmPasswordFilepath,
		"--datadir=" + consensusDataDirpath,
		fmt.Sprintf("--monitoring-port=%v", validatorMonitoringPortNum),
		"--verbosity=" + logLevel,
		// TODO SOMETHING ABOUT JWT
		// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
		"--disable-monitoring=false",
		"--monitoring-host=0.0.0.0",
		fmt.Sprintf("--monitoring-port=%v", validatorMonitoringPortNum),
		// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
	}
	if mevBoostContext != nil {
		// TODO required to work?
		// cmdArgs = append(cmdArgs, "--suggested-fee-recipient=0x...")
		cmdArgs = append(cmdArgs, "--enable-builder")
	}
	if len(extraParams) > 0 {
		cmdArgs = append(cmdArgs, extraParams...)
	}

	containerConfig := services.NewContainerConfigBuilder(
		validatorImage,
	).WithUsedPorts(
		validatorNodeUsedPorts,
	).WithCmdOverride(
		cmdArgs,
	).WithFiles(map[services.FilesArtifactUUID]string{
		launcher.genesisData.GetFilesArtifactUUID(): genesisDataMountDirpathOnServiceContainer,
		keystoreFiles.FilesArtifactUUID:             validatorKeysMountDirpathOnServiceContainer,
		launcher.prysmPasswordArtifactUuid:          prysmPasswordMountDirpathOnServiceContainer,
	}).Build()

	return containerConfig
}
