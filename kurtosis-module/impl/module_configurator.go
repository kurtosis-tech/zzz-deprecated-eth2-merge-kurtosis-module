package impl

import (
	"encoding/json"
	"github.com/kurtosis-tech/kurtosis-module-api-lib/golang/lib/kurtosis_modules"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
)

const(
	defaultLogLevel = "info"
)

// Parameters that the module accepts when loaded, serializeda as JSON
type LoadModuleParams struct {
	// Indicates the log level for this Kurtosis module implementation
	LogLevel string `json:"logLevel"`
}

type ExampleExecutableKurtosisModuleConfigurator struct{}

func NewExampleExecutableKurtosisModuleConfigurator() *ExampleExecutableKurtosisModuleConfigurator {
	return &ExampleExecutableKurtosisModuleConfigurator{}
}

func (t ExampleExecutableKurtosisModuleConfigurator) ParseParamsAndCreateExecutableModule(serializedCustomParamsStr string) (kurtosis_modules.ExecutableKurtosisModule, error) {
	serializedCustomParamsBytes := []byte(serializedCustomParamsStr)
	var args LoadModuleParams
	if err := json.Unmarshal(serializedCustomParamsBytes, &args); err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred deserializing the Kurtosis module serialized custom params with value '%v", serializedCustomParamsStr)
	}

	err := setLogLevel(args.LogLevel)
	if err != nil {
		return nil, stacktrace.Propagate(err, "An error occurred setting the log level")
	}

	module := NewExampleExecutableKurtosisModule()

	return module, nil
}

func setLogLevel(logLevelStr string) error {
	if logLevelStr == "" {
		logLevelStr = defaultLogLevel
	}
	level, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		return stacktrace.Propagate(err, "An error occurred parsing loglevel string '%v'", logLevelStr)
	}

	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	return nil
}
