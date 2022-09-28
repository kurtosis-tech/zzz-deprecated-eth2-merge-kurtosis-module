package mev_boost

import (
	"fmt"
	"strings"

	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-sdk/api/golang/core/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

type MEVBoostLauncher struct {
	ShouldCheckRelay bool
	RelayEndpoints   []string
}

const (
	flashbotsMevBoostImage        = "flashbots/mev-boost"
	flashbotsMevBoostPort  uint16 = 18550
)

var (
	usedPorts = map[string]*services.PortSpec{
		"api": services.NewPortSpec(flashbotsMevBoostPort, services.PortProtocol_TCP),
	}
	networkIdToName = map[string]string{
		"5":        "goerli",
		"11155111": "sepolia",
		"3":        "ropsten",
	}
)

func (launcher *MEVBoostLauncher) Launch(enclaveCtx *enclaves.EnclaveContext, serviceId services.ServiceID, networkId string) (*MEVBoostContext, error) {
	containerConfig := launcher.getContainerConfig(networkId)
	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfig)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the mev-boost instance with service ID '%v'", serviceId)
	}

	privateIPAddress := serviceCtx.GetPrivateIPAddress()
	return &MEVBoostContext{
		privateIPAddress: privateIPAddress,
		port:             flashbotsMevBoostPort,
	}, nil
}

func (launcher *MEVBoostLauncher) getContainerConfig(networkId string) *services.ContainerConfig {
	command := []string{"mev-boost"}
	networkName, ok := networkIdToName[networkId]
	if !ok {
		networkName = fmt.Sprintf("network-%s", networkId)
	}
	command = append(command, fmt.Sprintf("-%s", networkName))
	if launcher.ShouldCheckRelay {
		command = append(command, "-relay-check")
	}
	if len(launcher.RelayEndpoints) != 0 {
		command = append(command, "-relays", strings.Join(launcher.RelayEndpoints, ","))
	}
	containerConfig := services.NewContainerConfigBuilder(flashbotsMevBoostImage).WithUsedPorts(usedPorts).WithCmdOverride(command).Build()
	return containerConfig
}
