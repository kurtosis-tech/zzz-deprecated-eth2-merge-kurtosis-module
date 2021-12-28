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
	"strings"
	"time"
)

const (
	// imageName = "parithoshj/nimbus:merge-e623091"
	imageName = "statusim/nimbus-eth2:amd64-latest"

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
	defaultImageEntrypoint = "/home/user/nimbus-eth2/build/nimbus_beacon_node"

	validatorKeysDirpathRelToSharedDirRoot = "validator-keys"
	validatorSecretsDirpathRelToSharedDirRoot = "validator-secrets"
	validatorSecretsDirPerms = 0600	// If we don't set these when we copy, Nimbus will burn a bunch of time doing it for us

	// Nimbus needs write access to the validator keys/secrets directories, and b/c the module container runs as root
	//  while the Nimbus container does not, we can't just point the Nimbus binary to the paths in the shared dir because
	//  it won't be able to open them. To get around this, we copy the validator keys/secrets to a path inside the Nimbus
	//  container that is owned by the container's user
	validatorKeysDirpathOnServiceContainer = "$HOME/validator-keys"
	validatorSecretsDirpathOnServiceContainer = "$HOME/validator-secrets"

	maxNumHealthcheckRetries = 15
	timeBetweenHealthcheckRetries = 1 * time.Second
)
var usedPorts = map[string]*services.PortSpec{
	tcpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_TCP),
	udpDiscoveryPortID: services.NewPortSpec(discoveryPortNum, services.PortProtocol_UDP),
	httpPortID:         services.NewPortSpec(httpPortNum, services.PortProtocol_TCP),
	metricsPortID:         services.NewPortSpec(metricsPortNum, services.PortProtocol_TCP),
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
		nodeKeystoreDirpaths.NimbusKeysDirpath,
		nodeKeystoreDirpaths.RawSecretsDirpath,
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
	bootnodeContext *cl.CLClientContext, // If this is empty, the node will be launched as a bootnode
	elClientContext *el.ELClientContext,
	validatorKeysDirpathOnModuleContainer string,
	validatorSecretsDirpathOnModuleContainer string,
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
			defaultImageEntrypoint,
			"--non-interactive=true",
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
			"--log-level=info", // TODO make configurable
			// There's a bug where if we don't set this flag, the Nimbus nodes won't work:
			// https://discord.com/channels/641364059387854899/674288681737256970/922890280120750170
			// https://github.com/status-im/nimbus-eth2/issues/2451
			"--doppelganger-detection=false",
		}
		if bootnodeContext == nil {
			// Copied from https://github.com/status-im/nimbus-eth2/blob/67ab477a27e358d605e99bffeb67f98d18218eca/scripts/launch_local_testnet.sh#L417
			// See explanation there
			cmdArgs = append(cmdArgs, "--subscribe-all-subnets")
		} else {
			cmdArgs = append(cmdArgs, "--bootstrap-node=" + bootnodeContext.GetENR())
		}
		cmdStr := strings.Join(cmdArgs, " ")

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
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
