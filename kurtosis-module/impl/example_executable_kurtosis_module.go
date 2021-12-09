package impl

import (
	"encoding/json"
	"github.com/kurtosis-tech/kurtosis-core-api-lib/api/golang/lib/enclaves"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

var (
	tipsRepository = []string{
		"Everything not saved will be lost.",
		"Don't pet a burning dog.",
		"Even a broken clock is right twice a day.",
		"If no one comes from the future to stop you from doing it, then how bad of a decision can it really be?",
		"Never fall in love with a tennis player. Love means nothing to them.",
		"If you ever get caught sleeping on the job, slowly raise your head and say 'In Jesus' name, Amen'",
		"Never trust in an electrician with no eyebrows",
		"If you sleep until lunch time, you can save the breakfast money.",
	}
)

// Parameters that the execute command accepts, serialized as JSON
type ExecuteParams struct {
	IWantATip bool `json:"iWantATip"`
}

// Result that the execute command returns, serialized as JSON
type ExecuteResult struct {
	Tip string `json:"tip"`
}

type ExampleExecutableKurtosisModule struct {
}

func NewExampleExecutableKurtosisModule() *ExampleExecutableKurtosisModule {
	return &ExampleExecutableKurtosisModule{}
}

func (e ExampleExecutableKurtosisModule) Execute(enclaveCtx *enclaves.EnclaveContext, serializedParams string) (serializedResult string, resultError error) {
	logrus.Infof("Received serialized execute params '%v'", serializedParams)
	serializedParamsBytes := []byte(serializedParams)
	var params ExecuteParams
	if err := json.Unmarshal(serializedParamsBytes, &params); err != nil {
		return "", stacktrace.Propagate(err, "An error occurred deserializing the serialized execute params string '%v'", serializedParams)
	}

	resultObj := &ExecuteResult{
		Tip: getRandomTip(params.IWantATip),
	}

	result, err := json.Marshal(resultObj)
	if err != nil {
		return "", stacktrace.Propagate(err, "An error occurred serializing the result object '%+v'", resultObj)
	}
	stringResult := string(result)

	logrus.Info("Execution successful")
	return stringResult, nil
}

func getRandomTip(shouldGiveAdvice bool) string {
	var tip string
	if shouldGiveAdvice {
		rand.Seed(time.Now().Unix())
		tip = tipsRepository[rand.Intn(len(tipsRepository))]
	} else {
		tip = "The module won't enlighten you today."
	}
	return tip
}
