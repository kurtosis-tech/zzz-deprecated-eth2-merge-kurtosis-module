package grafana

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/static_files"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"io"
	"os"
	"path"
)

const (
	serviceID = "grafana"
	imageName = "grafana/grafana-enterprise:latest" //TODO I'm not sure if we should use latest version or ping an specific version instead

	httpPortId            = "http"
	httpPortNumber uint16 = 3000

	configDirectoriesPermission = 0755

	datasourcesConfigDirname = "datasources"
	datasourceConfigFilename = "datasource.yml"

	dashboardsConfigDirname          = "dashboards"
	dashboardProvidersConfigFilename = "dashboard-providers.yml"
	dashboardConfigFilename          = "dashboard.json"

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
	datasourcesConfigDirpathOnModule := path.Join(grafanaConfigDirpathOnModule, datasourcesConfigDirname)            // /tmp/grafana-config/datasources
	datasourceConfigFilepath := path.Join(datasourcesConfigDirpathOnModule, datasourceConfigFilename)                // /tmp/grafana-config/datasources/datasources.yml
	dashboardsConfigDirpathOnModule := path.Join(grafanaConfigDirpathOnModule, dashboardsConfigDirname)              // /tmp/grafana-config/dashboards/
	dashboardProvidersConfigFilepath := path.Join(dashboardsConfigDirpathOnModule, dashboardProvidersConfigFilename) // /tmp/grafana-config/dashboards/dashboards-provider.yml
	dashboardConfigFilepath := path.Join(dashboardsConfigDirpathOnModule, dashboardConfigFilename)                   // /tmp/grafana-config/dashboards/dashboards.json

	dashboardConfigFilepathOnGrafanaContainer := path.Join(
		grafanaConfigDirpathOnService,
		dashboardsConfigDirname,
		dashboardConfigFilename,
	) // /config/dashboards/dashboards.json

	dirpathsToCreate := []string{
		grafanaConfigDirpathOnModule,     // /tmp/grafana-config
		datasourcesConfigDirpathOnModule, // /tmp/grafana-config/datasources
		dashboardsConfigDirpathOnModule,  // /tmp/grafana-config/dashboards/
	}
	for _, dirpath := range dirpathsToCreate {
		if err := os.Mkdir(dirpath, configDirectoriesPermission); err != nil {
			return "", stacktrace.Propagate(err, "An error occurred creating Grafana config directory '%v'", dirpathsToCreate)
		}
	}

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
	templateAndDataByDestRelFilepath[datasourceConfigFilepath] = datasourceTemplateAndData                 // /tmp/grafana-config/datasources/datasources.yml -> data
	templateAndDataByDestRelFilepath[dashboardProvidersConfigFilepath] = dashboardProvidersTemplateAndData // /tmp/grafana-config/dashboards/dashboards-provider.yml -> data

	// copies the dashboard.json from /static-files/ to /tmp/grafana-config/dashboards/dashboards.json
	if err := addGrafanaDashboardConfigToConfigDir(
		static_files.GrafanaDashboardConfigFilepath,
		dashboardConfigFilepath,
	); err != nil {
		return "", stacktrace.Propagate(
			err,
			"An error occurred copying Grafana dashboard config file '%v' to the Grafana config directory at '%v'",
			static_files.GrafanaDashboardConfigFilepath,
			dashboardConfigFilepath,
		)
	}

	// we used to upload /tmp/grafana-config
	// with the render templates call below we are missing the right structure
	// we are also missing the dashboard.json
	// i still need a call to upload files or i can use an empty template
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

func addGrafanaDashboardConfigToConfigDir(srcFilepath, destFilepath string) error {
	// Copy the config file from the static files
	srcFp, err := os.Open(srcFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred opening Grafana dashboard config file '%v'", srcFilepath)
	}
	defer srcFp.Close()

	destFp, err := os.Create(destFilepath)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred creating dashboard config file '%v'", destFilepath)
	}
	defer destFp.Close()

	if _, err := io.Copy(destFp, srcFp); err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred copying bytes from dashboard config source file '%v' to destination file '%v'",
			srcFilepath,
			destFilepath,
		)
	}
	return nil
}
