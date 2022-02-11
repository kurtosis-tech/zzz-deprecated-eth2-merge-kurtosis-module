package grafana

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/service_launch_utils"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"os"
	"text/template"
)

const (
	serviceID = "grafana"
	imageName = "grafana/grafana-enterprise:latest" //TODO I'm not sure if we should use latest version or ping an specific version instead

	httpPortId = "http"
	httpPortNumber uint16 = 3000

	configDirectoriesPermission = 0755

	datasourcesConfigDirpathInShareDir = "datasources"
	datasourceConfigFileNameInShareDir = "datasource.yml"

	dashboardsConfigDirpahtInShareDir          = "dashboards"
	dashboardProvidersConfigFilenameInShareDir = "dashboard-providers.yml"
	grafanaDashboardConfigFilename             = "dashboard.json"
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

type datasourceConfigTemplateData struct {
	PrometheusURL string
}

type dashboardConfigTemplateData struct {
	DashboardsDirpath string
}

func LaunchGrafana(
	enclaveCtx *enclaves.EnclaveContext,
	datasourceConfigTemplate *template.Template,
	dashboardConfigTemplate *template.Template,
	prometheusUrl string,
	clClientContexts []*cl.CLClientContext,
) (string, error) {
	containerConfigSupplier, err := getContainerConfigSupplier(datasourceConfigTemplate, dashboardConfigTemplate, prometheusUrl, clClientContexts)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}
	serviceCtx, err := enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the grafana service")
	}

	publicIpAddr := serviceCtx.GetMaybePublicIPAddress()
	publicHttpPort, found := serviceCtx.GetPublicPorts()[httpPortId]
	if !found {
		return "", stacktrace.NewError("Expected the newly-started grafana service to have a port with ID '%v' but none was found", httpPortId)
	}

	publicUrl := fmt.Sprintf("http://%v:%v", publicIpAddr, publicHttpPort.GetNumber())

	return publicUrl, nil
}

// ====================================================================================================
//                                       Private Helper Functions
// ====================================================================================================
func getContainerConfigSupplier(
	datasourceConfigTemplate *template.Template,
	dashboardConfigTemplate *template.Template,
	prometheusUrl string,
	clClientContexts []*cl.CLClientContext,
	) (func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error), error,
) {
	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		datasourcesConfigSharedPath := sharedDir.GetChildPath(datasourcesConfigDirpathInShareDir)
		if err := os.Mkdir(datasourcesConfigSharedPath.GetAbsPathOnServiceContainer(), configDirectoriesPermission); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred creating directory '%v' on grafana service container ", datasourcesConfigSharedPath.GetAbsPathOnServiceContainer(), )
		}

		dashboardsConfigSharedPath := sharedDir.GetChildPath(dashboardsConfigDirpahtInShareDir)
		err := os.Mkdir(dashboardsConfigSharedPath.GetAbsPathOnServiceContainer(), configDirectoriesPermission)
		if err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred creating directory '%v' on grafana service container ", dashboardsConfigSharedPath.GetAbsPathOnServiceContainer())
		}

		datasourceConfigFileSharedPath := datasourcesConfigSharedPath.GetChildPath(datasourceConfigFileNameInShareDir)

		datasourceTemplateData := datasourceConfigTemplateData{
			PrometheusURL: prometheusUrl,
		}

		if err := service_launch_utils.FillTemplateToSharedPath(datasourceConfigTemplate, datasourceTemplateData, datasourceConfigFileSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the config file template")
		}

		dashboardTemplateData := dashboardConfigTemplateData{
			DashboardsDirpath: dashboardsConfigSharedPath.GetAbsPathOnServiceContainer(),
		}

		dashboardsConfigFileSharedPath := dashboardsConfigSharedPath.GetChildPath(dashboardProvidersConfigFilenameInShareDir)

		if err := service_launch_utils.FillTemplateToSharedPath(dashboardConfigTemplate, dashboardTemplateData, dashboardsConfigFileSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred filling the config file template")
		}

		if err := copyGrafanaDashboardConfigFileToSharedDir(dashboardsConfigSharedPath); err != nil {
			return nil, stacktrace.Propagate(err, "An error occurred copying grafana-dashboard-config file into the shared directory '%v'", dashboardsConfigSharedPath.GetAbsPathOnServiceContainer())
		}

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithEnvironmentVariableOverrides(map[string]string{
			"GF_PATHS_PROVISIONING": sharedDir.GetAbsPathOnServiceContainer(),
		}).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier, nil
}

func copyGrafanaDashboardConfigFileToSharedDir(
		dashboardsConfigSharedPath *services.SharedPath,
	) error {

	grafanaDashboardConfigFilepathInModuleContainer := static_files.GrafanaDashboardConfigFilepath

	grafanaDashboardConfigSharedPath := dashboardsConfigSharedPath.GetChildPath(grafanaDashboardConfigFilename)

	if err := service_launch_utils.CopyFileToSharedPath(
		grafanaDashboardConfigFilepathInModuleContainer,
		grafanaDashboardConfigSharedPath); err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred copying grafana-dashboard-config file from '%v' into path '%v'",
			grafanaDashboardConfigFilepathInModuleContainer,
			grafanaDashboardConfigSharedPath,
		)
	}
	return nil
}
