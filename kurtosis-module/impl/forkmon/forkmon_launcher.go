package forkmon

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/cl_client_network"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
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
}

func LaunchForkmon(
	enclaveCtx *enclaves.EnclaveContext,
	configTemplate *template.Template,
	clClientContexts []*cl_client_network.ConsensusLayerClientContext,
	secondsPerSlot uint32,
) (string, error) {
	containerConfigSupplier, err := getContainerConfigSupplier(
		configTemplate,
		clClientContexts,
		secondsPerSlot,
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
	clClientContexts []*cl_client_network.ConsensusLayerClientContext,
	secondsPerSlot uint32,
) (
	func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error),
	error,
) {
	allClClientInfo := []*clClientInfo{}
	for _, clClientCtx := range clClientContexts {
		clClientHttpPortId := clClientCtx.GetHTTPPortID()
		serviceCtx := clClientCtx.GetServiceContext()
		httpPort, found := serviceCtx.GetPrivatePorts()[clClientHttpPortId]
		if !found {
			return nil, stacktrace.NewError(
				"Expected CL client '%v' to have HTTP port with ID '%v', but none was found",
				serviceCtx.GetServiceID(),
				clClientHttpPortId,
			)
		}
		info := &clClientInfo{
			IPAddr:  serviceCtx.GetPrivateIPAddress(),
			PortNum: httpPort.GetNumber(),
		}
		allClClientInfo = append(allClClientInfo, info)
	}
	templateData := configTemplateData{
		ListenPortNum:  httpPortNumber,
		CLClientInfo:   allClClientInfo,
		SecondsPerSlot: secondsPerSlot,
	}

	containerConfigSupplier := func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		configFileSharedPath := sharedDir.GetChildPath(configRelFilepathInSharedDir)
		configFilepathOnModuleContainer := configFileSharedPath.GetAbsPathOnThisContainer()
		configFpOnModuleContainer, err := os.Create(configFilepathOnModuleContainer)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred opening config filepath '%v' for writing", configFilepathOnModuleContainer)
		}

		if err := configTemplate.Execute(configFpOnModuleContainer, templateData); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the config file template to output file '%v'", configFilepathOnModuleContainer)
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