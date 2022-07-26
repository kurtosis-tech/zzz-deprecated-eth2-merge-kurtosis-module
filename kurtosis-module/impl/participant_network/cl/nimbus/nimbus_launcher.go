package nimbus

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	cl_client_rest_client2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_genesis"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"strings"
	"time"
)

const (
	genesisDataMountpointOnClient = "/genesis-data"

	validatorKeysMountpointOnClient = "/validator-keys"

	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"
	metricsPortID      = "metrics"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000
	metricsPortNum          = 8008

	// Nimbus requires that its data directory already exists (because it expects you to bind-mount it), so we
	//  have to to create it
	consensusDataDirpathInServiceContainer = "$HOME/consensus-data"
	consensusDataDirPermsStr               = "0700" // Nimbus wants the data dir to have these perms

	// The entrypoint the image normally starts with (we need to override the entrypoint to create the
	//  consensus data directory on the image before it starts)
	defaultImageEntrypoint = "/home/user/nimbus-eth2/build/nimbus_beacon_node"

	// Nimbus needs write access to the validator keys/secrets directories, and b/c the module container runs as root
	//  while the Nimbus container does not, we can't just point the Nimbus binary to the paths in the shared dir because
	//  it won't be able to open them. To get around this, we copy the validator keys/secrets to a path inside the Nimbus
	//  container that is owned by the container's user
	validatorKeysDirpathOnServiceContainer    = "$HOME/validator-keys"
	validatorSecretsDirpathOnServiceContainer = "$HOME/validator-secrets"

	maxNumHealthcheckRetries      = 60
	timeBetweenHealthcheckRetries = 1 * time.Second

	metricsPath = "/metrics"
)

var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	metricsPortID:      services.NewPortSpec(metricsPortNum, services.PortProtocol_TCP),
}
var nimbusLogLevels = map[module_io.GlobalClientLogLevel]string{
	module_io.GlobalClientLogLevel_Error: "ERROR",
	module_io.GlobalClientLogLevel_Warn:  "WARN",
	module_io.GlobalClientLogLevel_Info:  "INFO",
	module_io.GlobalClientLogLevel_Debug: "DEBUG",
	module_io.GlobalClientLogLevel_Trace: "TRACE",
}

type NimbusLauncher struct {
	genesisData *cl_genesis.CLGenesisData

	// NOTE: This launcher does NOT take in the expected number of peers because doing so causes the Beacon node not to peer at all
	// See: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/issues/26
}

func NewNimbusLauncher(genesisData *cl_genesis.CLGenesisData) *NimbusLauncher {
	return &NimbusLauncher{genesisData: genesisData}
}

func (launcher NimbusLauncher) Launch(
	enclaveCtx *enclaves.EnclaveContext,
	serviceId services.ServiceID,
	image string,
	participantLogLevel string,
	globalLogLevel module_io.GlobalClientLogLevel,
	bootnodeContext *cl.CLClientContext,
	elClientContext *el.ELClientContext,
	keystoreFiles *cl2.KeystoreFiles,
	extraBeaconParams []string,
	extraValidatorParams []string,
) (resultClientCtx *cl.CLClientContext, resultErr error) {

	logLevel, err := module_io.GetClientLogLevelStrOrDefault(participantLogLevel, globalLogLevel, nimbusLogLevels)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the client log level using participant log level '%v' and global log level '%v'", participantLogLevel, globalLogLevel)
	}

	extraParams := append(extraBeaconParams, extraValidatorParams...)
	containerConfigSupplier := launcher.getContainerConfigSupplier(
		image,
		bootnodeContext,
		elClientContext,
		logLevel,
		keystoreFiles,
		extraParams,
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

	// TODO LAUNCH VALIDATOR NODE

	nodeIdentity, err := restClient.GetNodeIdentity()
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred getting the new Nimbus node's identity, which is necessary to retrieve its ENR")
	}

	metricsPort, found := serviceCtx.GetPrivatePorts()[metricsPortID]
	if !found {
		return nil, stacktrace.NewError("Expected new Nimbus service to have port with ID '%v', but none was found", metricsPortID)
	}
	metricsUrl := fmt.Sprintf("%v:%v", serviceCtx.GetPrivateIPAddress(), metricsPort.GetNumber())

	nodeMetricsInfo := cl.NewCLNodeMetricsInfo(string(serviceId), metricsPath, metricsUrl)
	nodesMetricsInfo := []*cl.CLNodeMetricsInfo{nodeMetricsInfo}

	result := cl.NewCLClientContext(
		"nimbus",
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
func (launcher *NimbusLauncher) getContainerConfigSupplier(
	image string,
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	logLevel string,
	keystoreFiles *cl2.KeystoreFiles,
	extraParams []string,
) func(string) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string) (*services.ContainerConfig, error) {
		elClientEngineRpcUrlStr := fmt.Sprintf(
			"http://%v:%v",
			elClientContext.GetIPAddress(),
			elClientContext.GetEngineRPCPortNum(),
		)

		// For some reason, Nimbus takes in the parent directory of the config file (rather than the path to the config file itself)
		genesisConfigParentDirpathOnClient := path.Join(genesisDataMountpointOnClient, path.Dir(launcher.genesisData.GetConfigYMLRelativeFilepath()))
		jwtSecretFilepath := path.Join(genesisDataMountpointOnClient, launcher.genesisData.GetJWTSecretRelativeFilepath())
		validatorKeysDirpath := path.Join(validatorKeysMountpointOnClient, keystoreFiles.NimbusKeysRelativeDirpath)
		validatorSecretsDirpath := path.Join(validatorKeysMountpointOnClient, keystoreFiles.RawSecretsRelativeDirpath)

		// Sources for these flags:
		//  1) https://github.com/status-im/nimbus-eth2/blob/stable/scripts/launch_local_testnet.sh
		//  2) https://github.com/status-im/nimbus-eth2/blob/67ab477a27e358d605e99bffeb67f98d18218eca/scripts/launch_local_testnet.sh#L417
		// WARNING: Do NOT set the --max-peers flag here, as doing so to the exact number of nodes seems to mess things up!
		// See: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/issues/26
		cmdArgs := []string{
			"mkdir",
			consensusDataDirpathInServiceContainer,
			"-m",
			consensusDataDirPermsStr,
			"&&",
			// TODO COMMENT THIS OUT?
			"cp",
			"-R",
			validatorKeysDirpath,
			validatorKeysDirpathOnServiceContainer,
			"&&",
			"cp",
			"-R",
			validatorSecretsDirpath,
			validatorSecretsDirpathOnServiceContainer,
			"&&",
			// If we don't do this chmod, Nimbus will spend a crazy amount of time manually correcting them
			//  before it starts
			"chmod",
			"600",
			validatorSecretsDirpathOnServiceContainer + "/*",
			"&&",
			defaultImageEntrypoint,
			"--non-interactive=true",
			"--log-level=" + logLevel,
			"--network=" + genesisConfigParentDirpathOnClient,
			"--data-dir=" + consensusDataDirpathInServiceContainer,
			"--web3-url=" + elClientEngineRpcUrlStr,
			"--nat=extip:" + privateIpAddr,
			"--enr-auto-update=false",
			"--rest",
			"--rest-address=0.0.0.0",
			fmt.Sprintf("--rest-port=%v", httpPortNum),
			"--validators-dir=" + validatorKeysDirpathOnServiceContainer,
			"--secrets-dir=" + validatorSecretsDirpathOnServiceContainer,
			// There's a bug where if we don't set this flag, the Nimbus nodes won't work:
			// https://discord.com/channels/641364059387854899/674288681737256970/922890280120750170
			// https://github.com/status-im/nimbus-eth2/issues/2451
			"--doppelganger-detection=false",
			// Set per Pari's recommendation to reduce noise in the logs
			"--subscribe-all-subnets=true",
			// Nimbus can handle a max of 256 threads, if the host has more then nimbus crashes. Setting it to 4 so it doesn't crash on build servers
			"--num-threads=4",
			fmt.Sprintf("--jwt-secret=%v", jwtSecretFilepath),
			// vvvvvvvvvvvvvvvvvvv METRICS CONFIG vvvvvvvvvvvvvvvvvvvvv
			"--metrics",
			"--metrics-address=0.0.0.0",
			fmt.Sprintf("--metrics-port=%v", metricsPortNum),
			// ^^^^^^^^^^^^^^^^^^^ METRICS CONFIG ^^^^^^^^^^^^^^^^^^^^^
		}
		if bootnodeContext == nil {
			// Copied from https://github.com/status-im/nimbus-eth2/blob/67ab477a27e358d605e99bffeb67f98d18218eca/scripts/launch_local_testnet.sh#L417
			// See explanation there
			cmdArgs = append(cmdArgs, "--subscribe-all-subnets")
		} else {
			cmdArgs = append(cmdArgs, "--bootstrap-node="+bootnodeContext.GetENR())
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
			launcher.genesisData.GetFilesArtifactUUID(): genesisDataMountpointOnClient,
			keystoreFiles.FilesArtifactUUID:             validatorKeysMountpointOnClient,
		}).Build()

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
