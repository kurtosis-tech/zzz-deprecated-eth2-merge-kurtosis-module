package grafana

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
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
	dashboardConfigRelFilepath          = "dashboard.json"

	configDirpathEnvVar = "GF_PATHS_PROVISIONING"

	grafanaConfigDirpathOnService  = "/config"
	grafanaDashboardsPathOnService = "/dashboards"
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
	artifactUuid, uploadArtifactUuid, err := getGrafanaConfigDirArtifactUuid(
		enclaveCtx,
		datasourceConfigTemplate,
		dashboardProvidersConfigTemplate,
		prometheusPrivateUrl,
	)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred getting the Grafana config directory files artifact")
	}

	containerConfigSupplier := getContainerConfigSupplier(artifactUuid, uploadArtifactUuid)
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
) (services.FilesArtifactUUID, services.FilesArtifactUUID, error) {
	dashboardConfigFilepathOnGrafanaContainer := path.Join(
		grafanaDashboardsPathOnService,
		dashboardConfigRelFilepath,
	)

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

	templateAndDataByDestRelFilepath := make(map[string]*enclaves.TemplateAndData)
	templateAndDataByDestRelFilepath[datasourceConfigRelFilepath] = datasourceTemplateAndData                 // /tmp/grafana-config/datasources/datasources.yml -> data
	templateAndDataByDestRelFilepath[dashboardProvidersConfigRelFilepath] = dashboardProvidersTemplateAndData // /tmp/grafana-config/dashboards/dashboards-provider.yml -> data

	renderedTemplateArtifactUuid, err := enclaveCtx.RenderTemplates(templateAndDataByDestRelFilepath)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred rendering Grafana templates")
	}

	uploadArtifactUuid, err := enclaveCtx.UploadFiles(static_files.GrafanaDashboardsConfigDirpath)
	if err != nil {
		return "", "", stacktrace.Propagate(err, "An error occurred uploading Grafana dashboard.json")
	}

	return renderedTemplateArtifactUuid, uploadArtifactUuid, nil
}

func getContainerConfigSupplier(
	renderTemplateArtifactUuid services.FilesArtifactUUID,
	uploadArtifactUuid services.FilesArtifactUUID,
) func(privateIpAddr string) (*services.ContainerConfig, error) {
	containerConfigSupplier := func(privateIpAddr string) (*services.ContainerConfig, error) {
		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).WithEnvironmentVariableOverrides(map[string]string{
			configDirpathEnvVar: grafanaConfigDirpathOnService,
		}).WithFiles(map[services.FilesArtifactUUID]string{
			renderTemplateArtifactUuid: grafanaConfigDirpathOnService,
			uploadArtifactUuid:         grafanaDashboardsPathOnService,
		}).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier
}
