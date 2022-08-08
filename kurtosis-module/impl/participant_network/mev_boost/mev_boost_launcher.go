package mev_boost

import (
	"fmt"
	"strings"

	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/services"
	"github.com/kurtosis-tech/stacktrace"
)

type MEVBoostLauncher struct {
	RelayCheck bool
	Relays     []string
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
	containerConfigSupplier := func(string) (*services.ContainerConfig, error) {
		relayCheckOption := ""
		if launcher.RelayCheck {
			relayCheckOption = "-relay-check"
		}
		networkName, ok := networkIdToName[networkId]
		if !ok {
			networkName = fmt.Sprintf("network-%s", networkId)
		}
		command := []string{
			"mev-boost",
			fmt.Sprintf("-%s", networkName),
			relayCheckOption,
			fmt.Sprintf("-relays %s", strings.Join(launcher.Relays, "")),
		}
		containerConfig := services.NewContainerConfigBuilder(flashbotsMevBoostImage).WithUsedPorts(usedPorts).WithCmdOverride(command).Build()
		return containerConfig, nil
	}

	serviceCtx, err := enclaveCtx.AddService(serviceId, containerConfigSupplier)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred launching the Geth EL client with service ID '%v'", serviceId)
	}

	return &MEVBoostContext{
		service: serviceCtx,
	}, nil
}
