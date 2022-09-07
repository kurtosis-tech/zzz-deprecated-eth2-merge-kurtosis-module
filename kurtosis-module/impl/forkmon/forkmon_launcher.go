package forkmon

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
)

const (
	serviceID = "forkmon"
	imageName = "ralexstokes/ethereum_consensus_monitor:latest"

	httpPortId     = "http"
	httpPortNumber = uint16(80)

	forkmonConfigFilepathOnModule = "/tmp/forkmon-config.toml"

	forkmonConfigMountDirpathOnService = "/config"
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

type clClientInfo struct {
	IPAddr  string
	PortNum uint16
}

type configTemplateData struct {
	ListenPortNum        uint16
	CLClientInfo         []*clClientInfo
	SecondsPerSlot       uint32
	SlotsPerEpoch        uint32
	GenesisUnixTimestamp uint64
}

func LaunchForkmon(
	enclaveCtx *enclaves.EnclaveContext,
	configTemplate string,
	clClientContexts []*cl.CLClientContext,
	genesisUnixTimestamp uint64,
	secondsPerSlot uint32,
	slotsPerEpoch uint32,
) error {
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

	templateAndData := enclaves.NewTemplateAndData(configTemplate, templateData)
	templateAndDataByDestRelFilepath := make(map[string]*enclaves.TemplateAndData)
	templateAndDataByDestRelFilepath[forkmonConfigFilepathOnModule] = templateAndData

	configArtifactUuid, err := enclaveCtx.RenderTemplates(templateAndDataByDestRelFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error rendering Forkmon config file template to '%v'", forkmonConfigFilepathOnModule)
	}

	containerConfigSupplier := getContainerConfigSupplier(configArtifactUuid)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}

	_, err = enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred launching the forkmon service")
	}
	return nil
}

func getContainerConfigSupplier(
	configArtifactUuid services.FilesArtifactUUID,
) func(privateIpAddr string) (*services.ContainerConfig, error) {
	return func(privateIpAddr string) (*services.ContainerConfig, error) {
		configFilepath := path.Join(forkmonConfigMountDirpathOnService, path.Base(forkmonConfigFilepathOnModule))
		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride([]string{
			"--config-path",
			configFilepath,
		}).WithUsedPorts(
			usedPorts,
		).WithFiles(map[services.FilesArtifactUUID]string{
			configArtifactUuid: forkmonConfigMountDirpathOnService,
		}).Build()

		return containerConfig, nil
	}
}
