package static_files

import (
	"github.com/kurtosis-tech/stacktrace"
	"path"
	"text/template"
)

const (
	// The path on the module container where static files are housed
	staticFilesDirpath = "/static-files"

	// Geth + CL genesis generation
	genesisGenerationConfigDirpath = staticFilesDirpath + "/genesis-generation-config"

	elGenesisGenerationConfigDirpath          = genesisGenerationConfigDirpath + "/el"
	ELGenesisGenerationConfigTemplateFilepath = elGenesisGenerationConfigDirpath + "/genesis-config.yaml.tmpl"

	clGenesisGenerationConfigDirpath             = genesisGenerationConfigDirpath + "/cl"
	CLGenesisGenerationConfigTemplateFilepath    = clGenesisGenerationConfigDirpath + "/config.yaml.tmpl"
	CLGenesisGenerationMnemonicsTemplateFilepath = clGenesisGenerationConfigDirpath + "/mnemonics.yaml.tmpl"

	// Prefunded keys
	prefundedKeysDirpath = staticFilesDirpath + "/genesis-prefunded-keys"
	GethPrefundedKeysDirpath = prefundedKeysDirpath + "/geth"

	// Forkmon config
	ForkmonConfigTemplateFilepath = staticFilesDirpath + "/forkmon-config/config.toml.tmpl"

	//Prometheus config
	PrometheusConfigTemplateFilepath = staticFilesDirpath + "/prometheus-config/prometheus.yml.tmpl"

	//Grafana config
	grafanaConfigDirpath                            = "/grafana-config"
	GrafanaDatasourceConfigTemplateFilepath         = staticFilesDirpath + grafanaConfigDirpath + "/datasource.yml.tmpl"
	GrafanaDashboardsConfigDirpath                  = staticFilesDirpath + grafanaConfigDirpath + "/dashboards"
	GrafanaDashboardProvidersConfigTemplateFilepath = GrafanaDashboardsConfigDirpath + "/dashboard-providers.yml.tmpl"
	GrafanaDashboardConfigFilepath                  = GrafanaDashboardsConfigDirpath + "/dashboard.json"
)

func ParseTemplate(filepath string) (*template.Template, error) {
	tmpl, err := template.New(
		// For some reason, the template name has to match the basename of the file:
		//  https://stackoverflow.com/questions/49043292/error-template-is-an-incomplete-or-empty-template
		path.Base(filepath),
	).ParseFiles(
		filepath,
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred parsing template file '%v'", filepath)
	}
	return tmpl, nil
}
