package nimbus

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/module_io"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	cl_client_rest_client2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl/cl_client_rest_client"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/el"
	cl2 "github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/prelaunch_data_generator/cl_validator_keystores"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	recursive_copy "github.com/otiai10/copy"
	"strings"
	"time"
)

const (
	// Port IDs
	tcpDiscoveryPortID = "tcp-discovery"
	udpDiscoveryPortID = "udp-discovery"
	httpPortID         = "http"
	metricsPortID      = "metrics"

	// Port nums
	discoveryPortNum uint16 = 9000
	httpPortNum             = 4000
	metricsPortNum          = 8008

	configDataDirpathRelToSharedDirRoot = "config-data"

	// Nimbus requires that its data directory already exists (because it expects you to bind-mount it), so we
	//  have to to create it
	consensusDataDirpathInServiceContainer = "$HOME/consensus-data"
	consensusDataDirPermsStr               = "0700" // Nimbus wants the data dir to have these perms

	// The entrypoint the image normally starts with (we need to override the entrypoint to create the
	//  consensus data directory on the image before it starts)
	defaultImageEntrypoint = "/home/user/nimbus-eth2/build/beacon_node"

	validatorKeysDirpathRelToSharedDirRoot    = "validator-keys"
	validatorSecretsDirpathRelToSharedDirRoot = "validator-secrets"
	validatorSecretsDirPerms                  = 0600 // If we don't set these when we copy, Nimbus will burn a bunch of time doing it for us

	// Nimbus needs write access to the validator keys/secrets directories, and b/c the module container runs as root
	//  while the Nimbus container does not, we can't just point the Nimbus binary to the paths in the shared dir because
	//  it won't be able to open them. To get around this, we copy the validator keys/secrets to a path inside the Nimbus
	//  container that is owned by the container's user
	validatorKeysDirpathOnServiceContainer    = "$HOME/validator-keys"
	validatorSecretsDirpathOnServiceContainer = "$HOME/validator-secrets"

	maxNumHealthcheckRetries      = 15
	timeBetweenHealthcheckRetries = 1 * time.Second
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
	// The dirpath on the module container where the config data directory exists
	configDataDirpathOnModuleContainer string

	// NOTE: This launcher does NOT take in the expected number of peers because doing so causes the Beacon node not to peer at all
	// See: https://github.com/kurtosis-tech/eth2-merge-kurtosis-module/issues/26
}

func NewNimbusLauncher(configDataDirpathOnModuleContainer string) *NimbusLauncher {
	return &NimbusLauncher{configDataDirpathOnModuleContainer: configDataDirpathOnModuleContainer}
}

func (launcher NimbusLauncher) Launch(
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
		nodeKeystoreDirpaths.NimbusKeysDirpath,
		nodeKeystoreDirpaths.RawSecretsDirpath,
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

	result := cl.NewCLClientContext(
		nodeIdentity.ENR,
		serviceCtx.GetPrivateIPAddress(),
		httpPortNum,
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
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
	extraParams []string,
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

		validatorKeysSharedPath := sharedDir.GetChildPath(validatorKeysDirpathRelToSharedDirRoot)
		if err := recursive_copy.Copy(validatorKeysDirpathOnModuleContainer, validatorKeysSharedPath.GetAbsPathOnThisContainer()); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying the validator keys into the shared directory so the node can consume them")
		}

		validatorSecretsSharedPath := sharedDir.GetChildPath(validatorSecretsDirpathRelToSharedDirRoot)
		if err := recursive_copy.Copy(
			validatorSecretsDirpathOnModuleContainer,
			validatorSecretsSharedPath.GetAbsPathOnThisContainer(),
			recursive_copy.Options{AddPermission: validatorSecretsDirPerms},
		); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying the validator secrets into the shared directory so the node can consume them")
		}

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
			"cp",
			"-R",
			validatorKeysSharedPath.GetAbsPathOnServiceContainer(),
			validatorKeysDirpathOnServiceContainer,
			"&&",
			"cp",
			"-R",
			validatorSecretsSharedPath.GetAbsPathOnServiceContainer(),
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
			"--network=" + configDataDirpathOnServiceSharedPath.GetAbsPathOnServiceContainer(),
			"--data-dir=" + consensusDataDirpathInServiceContainer,
			"--web3-url=" + elClientWsUrlStr,
			"--nat=extip:" + privateIpAddr,
			"--enr-auto-update=false",
			"--rest",
			"--rest-address=0.0.0.0",
			fmt.Sprintf("--rest-port=%v", httpPortNum),
			"--validators-dir=" + validatorKeysDirpathOnServiceContainer,
			"--secrets-dir=" + validatorSecretsDirpathOnServiceContainer,
			"--metrics",
			"--metrics-address=0.0.0.0",
			fmt.Sprintf("--metrics-port=%v", metricsPortNum),
			// There's a bug where if we don't set this flag, the Nimbus nodes won't work:
			// https://discord.com/channels/641364059387854899/674288681737256970/922890280120750170
			// https://github.com/status-im/nimbus-eth2/issues/2451
			"--doppelganger-detection=false",
			// Set per Pari's recommendation to reduce noise in the logs
			"--subscribe-all-subnets=true",
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
