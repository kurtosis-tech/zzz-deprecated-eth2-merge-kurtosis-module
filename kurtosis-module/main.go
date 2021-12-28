package main

import (
	"fmt"
	"github.com/kurtosis-tech/eth2-merge-kurtosis-module/kurtosis-module/impl"
	"github.com/kurtosis-tech/kurtosis-module-api-lib/golang/lib/execution"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	successExitCode = 0
	failureExitCode = 1
)

func main() {
	configurator := impl.NewEth2KurtosisModuleConfigurator()
	executor := execution.NewKurtosisModuleExecutor(configurator)
	if err := executor.Run(); err != nil {
		logrus.Errorf("An error occurred running the Kurtosis module executor:")
		fmt.Fprintln(logrus.StandardLogger().Out, err)
		os.Exit(failureExitCode)
	}
	os.Exit(successExitCode)
}
