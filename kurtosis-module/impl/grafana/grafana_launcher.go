package grafana

import (
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"path"
)

const (
	serviceID = "grafana"
	imageName = "grafana/grafana-enterprise:latest" //TODO I'm not sure if we should use latest version or ping an specific version instead

	httpPortId            = "http"
	httpPortNumber uint16 = 3000

	datasourceConfigRelFilepath = "datasources/datasource.yml"

	dashboardProvidersConfigRelFilepath = "dashboards/dashboard-providers.yml"
	dashboardConfigRelFilepath          = "dashboards/dashboard.json"

	configDirpathEnvVar = "GF_PATHS_PROVISIONING"

	grafanaConfigDirpathOnModule = "/tmp/grafana-config"

	grafanaConfigDirpathOnService = "/config"
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

type datasourceConfigTemplateData struct {
	PrometheusURL string
}

type dashboardProvidersConfigTemplateData struct {
	DashboardsDirpath string
}

func LaunchGrafana(
	enclaveCtx *enclaves.EnclaveContext,
	datasourceConfigTemplate string,
	dashboardProvidersConfigTemplate string,
	prometheusPrivateUrl string,
) error {
	artifactUuid, err := getGrafanaConfigDirArtifactUuid(
		enclaveCtx,
		datasourceConfigTemplate,
		dashboardProvidersConfigTemplate,
		prometheusPrivateUrl,
	)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred getting the Grafana config directory files artifact")
	}

	containerConfigSupplier := getContainerConfigSupplier(artifactUuid)
	_, err = enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred launching the grafana service")
	}

	return nil
}

// ====================================================================================================
//
//	Private Helper Functions
//
// ====================================================================================================
func getGrafanaConfigDirArtifactUuid(
	enclaveCtx *enclaves.EnclaveContext,
	datasourceConfigTemplate string,
	dashboardProvidersConfigTemplate string,
	prometheusPrivateUrl string,
) (services.FilesArtifactUUID, error) {
	dashboardConfigFilepathOnGrafanaContainer := path.Join(
		grafanaConfigDirpathOnService,
		dashboardConfigRelFilepath,
	) // /config/dashboards/dashboards.json

	datasourceData := datasourceConfigTemplateData{
		PrometheusURL: prometheusPrivateUrl,
	}
	datasourceTemplateAndData := enclaves.NewTemplateAndData(datasourceConfigTemplate, datasourceData)

	dashboardProvidersData := dashboardProvidersConfigTemplateData{
		// Grafana needs to know where the dashboards config file will be on disk, which means we need to feed
		//  it the *mounted* location on disk (on the Grafana container) when we generate this on the module container
		DashboardsDirpath: dashboardConfigFilepathOnGrafanaContainer, // /config/dashboards/dashboards.json
	}
	dashboardProvidersTemplateAndData := enclaves.NewTemplateAndData(dashboardProvidersConfigTemplate, dashboardProvidersData)

	// Even though this is a pure json we treat it as a template
	dashboardConfigFileContent, err := ioutil.ReadFile(static_files.GrafanaDashboardConfigFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred reading the dashboard file content")
	}
	dashboardConfigFileTemplateAndData := enclaves.NewTemplateAndData(string(dashboardConfigFileContent), "")

	templateAndDataByDestRelFilepath := make(map[string]*enclaves.TemplateAndData)
	templateAndDataByDestRelFilepath[datasourceConfigRelFilepath] = datasourceTemplateAndData                 // /tmp/grafana-config/datasources/datasources.yml -> data
	templateAndDataByDestRelFilepath[dashboardProvidersConfigRelFilepath] = dashboardProvidersTemplateAndData // /tmp/grafana-config/dashboards/dashboards-provider.yml -> data
	templateAndDataByDestRelFilepath[dashboardConfigRelFilepath] = dashboardConfigFileTemplateAndData

	artifactUuid, err := enclaveCtx.RenderTemplates(templateAndDataByDestRelFilepath)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred rendering Grafana templates")
	}

	return artifactUuid, nil
}

func getContainerConfigSupplier(
	configDirArtifactUuid services.FilesArtifactUUID,
) func(privateIpAddr string) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string) (*services.ContainerConfig, error) {
		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithEnvironmentVariableOverrides(map[string]string{
			configDirpathEnvVar: grafanaConfigDirpathOnModule,
		}).WithFiles(map[services.FilesArtifactUUID]string{
			configDirArtifactUuid: grafanaConfigDirpathOnService,
		}).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier
}
