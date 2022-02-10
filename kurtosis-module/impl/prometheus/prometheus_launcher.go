package prometheus

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
	serviceID = "prometheus"
	imageName = "prom/prometheus:latest"//TODO I'm not sure if we should use latest version or ping an specific version instead

	httpPortId = "http"
	httpPortNumber uint16 = 9090

	// The filepath, relative to the root of the shared dir, where we'll generate the config file
	configRelFilepathInSharedDir = "prometheus.yml"
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

type clClientInfo struct {
	clNodesMetricsInfo []*cl.CLNodeMetricsInfo
}

type configTemplateData struct {
	CLNodesMetricsInfo []*cl.CLNodeMetricsInfo
}

func LaunchPrometheus(
	enclaveCtx *enclaves.EnclaveContext,
	configTemplate *template.Template,
	clClientContexts []*cl.CLClientContext,
) (string, string, error) {
	containerConfigSupplier, err := getContainerConfigSupplier(clClientContexts, configTemplate)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}
	serviceCtx, err := enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred launching the prometheus service")
	}

	publicIpAddr := serviceCtx.GetMaybePublicIPAddress()
	publicHttpPort, found := serviceCtx.GetPublicPorts()[httpPortId]
	if !found {
		return "", "", stacktrace.NewError("Expected the newly-started prometheus service to have a public port with ID '%v' but none was found", httpPortId)
	}

	publicUrl := fmt.Sprintf("http://%v:%v", publicIpAddr, publicHttpPort.GetNumber())

	privateIpAddr := serviceCtx.GetPrivateIPAddress()
	privateHttpPort, found := serviceCtx.GetPrivatePorts()[httpPortId]
	if !found {
		return "", "", stacktrace.NewError("Expected the newly-started prometheus service to have a private port with ID '%v' but none was found", httpPortId)
	}

	privateUrl := fmt.Sprintf("http://%v:%v", privateIpAddr, privateHttpPort.GetNumber())
	return publicUrl, privateUrl, nil
}

// ====================================================================================================
//                                       Private Helper Functions
// ====================================================================================================
func getContainerConfigSupplier(
	clClientContexts []*cl.CLClientContext,
	configTemplate *template.Template,
) (
	func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error),
	error,
) {
	allCLNodesMetricsInfo := []*cl.CLNodeMetricsInfo{}
	for _, clClientCtx := range clClientContexts {
		clClientMetricsInfo := clClientCtx.GetMetricsInfo()
		if clClientMetricsInfo != nil {
			if clClientMetricsInfo.GetClNodesMetricsInfo() != nil {
				allCLNodesMetricsInfo = append(allCLNodesMetricsInfo, clClientMetricsInfo.GetClNodesMetricsInfo()...)
			}
		}
	}

	templateData := configTemplateData{
		CLNodesMetricsInfo: allCLNodesMetricsInfo,
	}

	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {
		configFileSharedPath := sharedDir.GetChildPath(configRelFilepathInSharedDir)
		if err := service_launch_utils.FillTemplateToSharedPath(configTemplate, templateData, configFileSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the config file template")
		}

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride([]string{
			"--config.file=" + configFileSharedPath.GetAbsPathOnServiceContainer(),
			"--storage.tsdb.path=/prometheus",
			"--web.console.libraries=/etc/prometheus/console_libraries",
			"--web.console.templates=/etc/prometheus/consoles",
			"--web.enable-lifecycle",
		}).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier, nil
}
