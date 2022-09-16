package teku

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
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	tekuBinaryFilepathInImage = "/opt/teku/bin/teku"

	genesisDataMountDirpathOnServiceContainer = "/genesis"

	// The Docker container runs as the "teku" user so we can't write to root
	consensusDataDirpathOnServiceContainer = "/opt/teku/consensus-data"

	// These will get mounted as root and Teku needs directory write permissions, so we'll copy this
	//  into the Teku user's home directory to get around it
	validatorKeysDirpathOnServiceContainer = "/validator-keys"

	// TODO Get rid of this being hardcoded; should be shared
	validatingRewardsAccount = "0x0000000000000000000000000000000000000000"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"
	metricsPortID      = "metrics"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000
	metricsPortNum   uint16 = 8008

	// 1) The Teku container runs as the "teku" user
	// 2) Teku requires write access to the validator secrets directory, so it can write a lockfile into it as it uses the keys
	// 3) The module container runs as 'root'
	// With these three things combined, it means that when the module container tries to write the validator keys/secrets into
	//  the shared directory, it does so as 'root'. When Teku tries to consum the same files, it will get a failure because it
	//  doesn't have permission to write to the 'validator-secrets' directory.
	// To get around this, we copy the files AGAIN from
	destValidatorKeysDirpathInServiceContainer    = "$HOME/validator-keys"
	destValidatorSecretsDirpathInServiceContainer = "$HOME/validator-secrets"

	// Teku nodes take ~35s to bring their HTTP server up
	maxNumHealthcheckRetries      = 100
	timeBetweenHealthcheckRetries = 2 * time.Second

	minPeers = 1

	metricsPath = "/metrics"

	privateIPAddressPlaceholder = "KURTOSIS_PRIVATE_IP_ADDR_PLACEHOLDER"
)

var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	metricsPortID:      services.NewPortSpec(metricsPortNum, services.PortProtocol_TCP),
}
var tekuLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "ERROR",
	module_io.GlobalClientLogLevel_Warn:  "WARN",
	module_io.GlobalClientLogLevel_Info:  "INFO",
	module_io.GlobalClientLogLevel_Debug: "DEBUG",
	module_io.GlobalClientLogLevel_Trace: "TRACE",
}

type TekuCLClientLauncher struct {
	clGenesisData          *cl_genesis.CLGenesisData
	expectedNumBeaconNodes uint32
}

func NewTekuCLClientLauncher(clGenesisData *cl_genesis.CLGenesisData) *TekuCLClientLauncher {
	return &TekuCLClientLauncher{clGenesisData: clGenesisData}
}

func (launcher *TekuCLClientLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	// TODO move to launcher param
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	mevBoostContext *mev_boost.MEVBoostContext,
	keystoreFiles *cl2.KeystoreFiles,
	extraBeaconParams []string,
	extraValidatorParams []string,
) (resultClientCtx *cl.CLClientContext, resultErr error) {

	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, tekuLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	extraParams := append(extraBeaconParams, extraValidatorParams...)
	containerConfigSupplier := launcher.getContainerConfig(
		image,
		bootnodeContext,
		elClientContext,
		mevBoostContext,
		logLevel,
		keystoreFiles,
		extraParams,
	)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Teku CL client with service ID '%v'", serviceId)
	}

	httpPort, found := serviceCtx.GetPrivatePorts()[httpPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Teku service to have port with ID '%v', but none was found", httpPortID)
	}

	restClient := cl_client_rest_client.NewCLClientRESTClient(serviceCtx.GetPrivateIPAddress(), httpPort.GetNumber())

	if err := cl.WaitForBeaconClientAvailability(restClient, maxNumHealthcheckRetries, timeBetweenHealthcheckRetries); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred waiting for the new Teku node to become available")
	}

	// TODO add validator availability using teh validator API: https://ethereum.github.io/beacon-APIs/?urls.primaryName=v1#/ValidatorRequiredApi

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Teku node's identity, which is necessary to retrieve its ENR")
	}

	metricsPort, found := serviceCtx.GetPrivatePorts()[metricsPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Teku service to have port with ID '%v', but none was found", metricsPortID)
	}
	metricsUrl := fmt.Sprintf("%v:%v", serviceCtx.GetPrivateIPAddress(), metricsPort.GetNumber())

	nodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(serviceId), metricsPath, metricsUrl)
	nodesMetricsInfo := []*cl.CLNodeMetricsInfo{nodeMetricsInfo}

	result := cl.NewCLClientContext(
		"teku",
		nodeIdentity.ENR,
		serviceCtx.GetPrivateIPAddress(),
		httpPortNum,
		nodesMetricsInfo,
		restClient,
	)

	return result, nil
}

// ====================================================================================================
//                                   Private Helper Methods
// ====================================================================================================
func (launcher *TekuCLClientLauncher) getContainerConfig(
	image string,
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	mevBoostContext *mev_boost.MEVBoostContext,
	logLevel string,
	keystoreFiles *cl2.KeystoreFiles,
	extraParams []string,
) *services.ContainerConfig {
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

	genesisConfigFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.clGenesisData.GetConfigYMLRelativeFilepath())
	genesisSszFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.clGenesisData.GetGenesisSSZRelativeFilepath())
	jwtSecretFilepath := path.Join(genesisDataMountDirpathOnServiceContainer, launcher.clGenesisData.GetJWTSecretRelativeFilepath())
	validatorKeysDirpath := path.Join(validatorKeysDirpathOnServiceContainer, keystoreFiles.TekuKeysRelativeDirpath)
	validatorSecretsDirpath := path.Join(validatorKeysDirpathOnServiceContainer, keystoreFiles.TekuSecretsRelativeDirpath)
	cmdArgs := []string{
		// Needed because the generated keys are owned by root and the Teku image runs as the 'teku' user
		"cp",
		"-R",
		validatorKeysDirpath,
		destValidatorKeysDirpathInServiceContainer,
		"&&",
		// Needed because the generated keys are owned by root and the Teku image runs as the 'teku' user
		"cp",
		"-R",
		validatorSecretsDirpath,
		destValidatorSecretsDirpathInServiceContainer,
		"&&",
		tekuBinaryFilepathInImage,
		"--Xee-version kilnv2",
		"--logging=" + logLevel,
		"--log-destination=CONSOLE",
		"--network=" + genesisConfigFilepath,
		"--initial-state=" + genesisSszFilepath,
		"--data-path=" + consensusDataDirpathOnServiceContainer,
		"--data-storage-mode=PRUNE",
		"--p2p-enabled=true",
		// Set per Pari's recommendation, to reduce noise in the logs
		"--p2p-subscribe-all-subnets-enabled=true",
		fmt.Sprintf("--p2p-peer-lower-bound=%v", minPeers),
		"--eth1-endpoints=" + elClientRpcUrlStr,
		"--p2p-advertised-ip=" + privateIPAddressPlaceholder,
		"--rest-api-enabled=true",
		"--rest-api-docs-enabled=true",
		"--rest-api-interface=0.0.0.0",
		fmt.Sprintf("--rest-api-port=%v", httpPortNum),
		"--rest-api-host-allowlist=*",
		"--data-storage-non-canonical-blocks-enabled=true",
		fmt.Sprintf(
			"--validator-keys=%v:%v",
			destValidatorKeysDirpathInServiceContainer,
			destValidatorSecretsDirpathInServiceContainer,
		),
		fmt.Sprintf("--ee-jwt-secret-file=%v", jwtSecretFilepath),
		"--ee-endpoint=" + elClientEngineRpcUrlStr,
		"--validators-proposer-default-fee-recipient=" + validatingRewardsAccount,
		// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
		"--metrics-enabled",
		"--metrics-interface=0.0.0.0",
		"--metrics-host-allowlist='*'",
		"--metrics-categories=BEACON,PROCESS,LIBP2P,JVM,NETWORK,PROCESS",
		fmt.Sprintf("--metrics-port=%v", metricsPortNum),
		// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
	}
	if bootnodeContext != nil {
		cmdArgs = append(cmdArgs, "--p2p-discovery-bootnodes="+bootnodeContext.GetENR())
	}
	if mevBoostContext != nil {
		cmdArgs = append(cmdArgs, "--validators-builder-registration-default-enabled=true")
		cmdArgs = append(cmdArgs, fmt.Sprintf("--builder-endpoint='%s'", mevBoostContext.Endpoint()))
	}
	if len(extraParams) > 0 {
		cmdArgs = append(cmdArgs, extraParams...)
	}
	cmdStr := strings.Join(cmdArgs, " ")

	containerConfig := services.NewContainerConfigBuilder(
		image,
	).WithUsedPorts(
		usedPorts,
	).WithEntrypointOverride([]string{
		"sh", "-c",
	}).WithCmdOverride([]string{
		cmdStr,
	}).WithFiles(map[services.FilesArtifactUUID]string{
		launcher.clGenesisData.GetFilesArtifactUUID(): genesisDataMountDirpathOnServiceContainer,
		keystoreFiles.FilesArtifactUUID:               validatorKeysDirpathOnServiceContainer,
	}).Build()

	return containerConfig
}
