package forkmon

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"text/template"
)

const (
	serviceID = "forkmon"
	imageName = "ralexstokes/ethereum_consensus_monitor:latest"

	httpPortId = "http"
	httpPortNumber = uint16(80)

	// The filepath, relative to the root of the shared dir, where we'll generate the config file
	configRelFilepathInSharedDir = "config.toml"
)
var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

type clClientInfo struct {
	IPAddr string
	PortNum uint16
}

type configTemplateData struct {
	ListenPortNum uint16
	CLClientInfo []*clClientInfo
	SecondsPerSlot uint32
	SlotsPerEpoch uint32
	GenesisUnixTimestamp uint64
}

func LaunchForkmon(
	enclaveCtx *enclaves.EnclaveContext,
	configTemplate *template.Template,
	clClientContexts []*cl.CLClientContext,
	genesisUnixTimestamp uint64,
	secondsPerSlot uint32,
	slotsPerEpoch uint32,
) (string, error) {
	containerConfigSupplier, err := getContainerConfigSupplier(
		configTemplate,
		clClientContexts,
		genesisUnixTimestamp,
		secondsPerSlot,
		slotsPerEpoch,
	)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}

	serviceCtx, err := enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the forkmon service")
	}

	publicIpAddr := serviceCtx.GetMaybePublicIPAddress()
	publicHttpPort, found := serviceCtx.GetPublicPorts()[httpPortId]
	if !found {
		return "", stacktrace.NewError("Expected the newly-started forkmon service to have a port with ID '%v' but none was found", httpPortId)
	}

	publicUrl := fmt.Sprintf("http://%v:%v", publicIpAddr, publicHttpPort.GetNumber())
	return publicUrl, nil
}

func getContainerConfigSupplier(
	configTemplate *template.Template,
	clClientContexts []*cl.CLClientContext,
	genesisUnixTimestamp uint64,
	secondsPerSlot uint32,
	slotsPerEpoch uint32,
) (
	func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error),
	error,
) {
	allClClientInfo := []*clClientInfo{}
	for _, clClientCtx := range clClientContexts {
		info := &clClientInfo{
			IPAddr:  clClientCtx.GetIPAddress(),
			PortNum: clClientCtx.GetHTTPPortNum(),
		}
		allClClientInfo = append(allClClientInfo, info)
	}
	templateData := configTemplateData{
		ListenPortNum:        httpPortNumber,
		CLClientInfo:         allClClientInfo,
		SecondsPerSlot:       secondsPerSlot,
		SlotsPerEpoch:        slotsPerEpoch,
		GenesisUnixTimestamp: genesisUnixTimestamp,
	}

	containerConfigSupplier := func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		configFileSharedPath := sharedDir.GetChildPath(configRelFilepathInSharedDir)
		if err := service_launch_utils.FillTemplateToSharedPath(configTemplate, templateData, configFileSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the config file template")
		}

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride([]string{
			"--config-path",
			configFileSharedPath.GetAbsPathOnServiceContainer(),
		}).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier, nil
}