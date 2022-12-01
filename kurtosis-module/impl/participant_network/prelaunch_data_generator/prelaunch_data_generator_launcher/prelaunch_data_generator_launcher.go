package prelaunch_data_generator_launcher

import (
	"fmt"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
	"time"
)

const (
	image = "ethpandaops/ethereum-genesis-generator:1.0.3"

	serviceIdPrefix = "prelaunch-data-generator-"
)

// We use Docker exec commands to run the commands we need, so we override the default
var entrypointArgs = []string{
	"sleep",
	"999999",
}

// Launches a prelaunch data generator image, for use in various of the genesis generation
func LaunchPrelaunchDataGenerator(
	enclaveCtx *enclaves.EnclaveContext,
	filesArtifactMountpoints map[services.FilesArtifactUUID]string,
) (
	*services.ServiceContext,
	error,
) {
	containerConfig := getContainerConfig(filesArtifactMountpoints)

	serviceId := services.ServiceID(fmt.Sprintf(
		"%v%v",
		serviceIdPrefix,
		time.Now().Unix(),
	))

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the prelaunch data generator container with service ID '%v'", serviceIdPrefix)
	}

	return serviceCtx, nil
}

func getContainerConfig(
	filesArtifactMountpoints map[services.FilesArtifactUUID]string,
) *services.ContainerConfig {
	containerConfig := services.NewContainerConfigBuilder(
		image,
	).WithEntrypointOverride(
		entrypointArgs,
	).WithFiles(
		filesArtifactMountpoints,
	).Build()

	return containerConfig
}
