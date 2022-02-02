package node_exporter

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

const (
	serviceID = "node-exporter"
	imageName = "prom/node-exporter:latest"

	httpPortId = "http"
	httpPortNumber uint16 = 9100

	waitForStartupTimeBetweenPolls = 10
	waitForStartupMaxPolls         = 15
	waitInitialDelayMilliseconds   = 500
	waitForStartupEndpointPath = ""
	waitForStartupBodyText = ""
)

var usedPorts = map[string]*services.PortSpec{
	httpPortId: services.NewPortSpec(httpPortNumber, services.PortProtocol_TCP),
}

func LaunchNodeExporter(enclaveCtx *enclaves.EnclaveContext,) (string, error)  {
	containerConfigSupplier, err := getContainerConfigSupplier()
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred getting the container config supplier")
	}

	serviceCtx, err := enclaveCtx.AddService(serviceID, containerConfigSupplier)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred launching the node exporter service")
	}

	publicIpAddr := serviceCtx.GetMaybePublicIPAddress()
	publicHttpPort, found := serviceCtx.GetPublicPorts()[httpPortId]
	if !found {
		return "", stacktrace.NewError("Expected the newly-started node exporter service to have a port with ID '%v' but none was found", httpPortId)
	}


	if err := enclaveCtx.WaitForHttpGetEndpointAvailability(
		serviceID,
		uint32(httpPortNumber),
		waitForStartupEndpointPath,
		waitInitialDelayMilliseconds,
		waitForStartupMaxPolls,
		waitForStartupTimeBetweenPolls,
		waitForStartupBodyText,
		); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred waiting for node exporter service to start up")
	}

	publicUrl := fmt.Sprintf("http://%v:%v", publicIpAddr, publicHttpPort.GetNumber())

	return publicUrl, nil
}

func getContainerConfigSupplier() (
	func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error),
	error,
) {

	containerConfigSupplier := func (privateIpAddr string, sharedDir *services.SharedPath) (*services.ContainerConfig, error) {

		containerConfig := services.NewContainerConfigBuilder(
			imageName,
		).WithUsedPorts(
			usedPorts,
		).Build()

		return containerConfig, nil
	}

	return containerConfigSupplier, nil
}
