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

const jsonStr = `
{
    "data": [
        {
            "root": "0x661927326bbe482422ed0a6006226c20e24a0e599e876adb7a10b4d92901a090",
            "canonical": true,
            "header": {
                "message": {
                    "slot": "77",
                    "proposer_index": "32",
                    "parent_root": "0x748762a034cbba922bdf0092d06c22a6a4d4540ba7131d14f151e931c6ceef77",
                    "state_root": "0x9018e4650d8a9bd42ea447e11041b184ae0f31f6a778d6b98351a9edd70d1c0c",
                    "body_root": "0xc281b43687775f9c03b6d54bec04b21e8ba020ff3359392ae0d0ce54fc45a69d"
                },
                "signature": "0xb64225b0f69ed87676dbf244f5af79b07e83eb983bce093756f8339177ba969b36dd8d9d13b67c5cca28df2fd8e384570018b1d9a51fa40efcd4d32a437aed6c9490d0fba4e12fa571b1b9cf66b980e63c8759a88f9c99890c5cbbbfbe6b8ca9"
            }
        }
    ]
}`

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
