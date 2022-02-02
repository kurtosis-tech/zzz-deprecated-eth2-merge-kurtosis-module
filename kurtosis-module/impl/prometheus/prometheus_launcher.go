package prometheus

import (
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl/participant_network/cl"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"text/template"
)

const (
	serviceID = "prometheus"
	imageName = "prom/prometheus:latest"

	httpPortId = "http"
	httpPortNumber uint16 = 9090

	// The filepath, relative to the root of the shared dir, where we'll generate the config file
	configRelFilepathInSharedDir = "prometheus.yml"
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}




func LaunchPrometheus(
	enclaveCtx *enclaves.EnclaveContext,
	configTemplate *template.Template,
	clClientContexts []*cl.CLClientContext,
) error {
	containerConfigSupplier, err := getContainerConfigSupplier(clClientContexts)
	if err != nil {
		return  stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}
	logrus.Infof("%v", containerConfigSupplier)
	return nil
}

func getContainerConfigSupplier(
	clClientContexts []*cl.CLClientContext,
) (
	func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error),
	error,
) {

	containerConfigSupplier := func(privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithCmdOverride([]string{
			/*"--config-path",
			configFileSharedPath.GetAbsPathOnServiceContainer(),*/
		}).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier, nil
}
