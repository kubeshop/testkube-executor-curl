package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
	"github.com/kubeshop/kubtest/pkg/log"
	"github.com/kubeshop/kubtest/pkg/process"
	"go.uber.org/zap"
)

const CurlAdditionalFlags = "-Is"

type CurlRunnerInput struct {
	Command        []string `json:"command"`
	ExpectedStatus int      `json:"expected_status"`
	ExpectedBody   string   `json:"expected_body"`
}

// CurlRunner is used to run curl commands.
type CurlRunner struct {
	Log *zap.SugaredLogger
}

func NewCurlRunner(logger *zap.SugaredLogger) *CurlRunner {
	r := CurlRunner{Log: log.DefaultLogger}
	if logger != nil {
		r.Log = logger
	}
	return &r
}

func (r *CurlRunner) Run(execution kubtest.Execution) kubtest.ExecutionResult {
	var runnerInput CurlRunnerInput
	err := json.Unmarshal([]byte(execution.ScriptContent), &runnerInput)
	if err != nil {
		return kubtest.ExecutionResult{
			Status: kubtest.ExecutionStatusError,
		}
	}
	command := runnerInput.Command[0]
	runnerInput.Command[0] = CurlAdditionalFlags
	output, err := process.Execute(command, runnerInput.Command...)
	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return kubtest.ExecutionResult{
			Status:       kubtest.ExecutionStatusError,
			ErrorMessage: fmt.Sprintf("Error occured when running a command %s", err),
		}
	}

	outputString := string(output)
	responseStatus := getResponseCode(outputString)
	if responseStatus != runnerInput.ExpectedStatus {
		return kubtest.ExecutionResult{
			Status:       kubtest.ExecutionStatusError,
			RawOutput:    outputString,
			ErrorMessage: fmt.Sprintf("Response statut don't match expected %d got %d", runnerInput.ExpectedStatus, responseStatus),
		}
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		return kubtest.ExecutionResult{
			Status:       kubtest.ExecutionStatusError,
			RawOutput:    outputString,
			ErrorMessage: fmt.Sprintf("Response doesn't contain body: %s", runnerInput.ExpectedBody),
		}
	}

	return kubtest.ExecutionResult{
		Status:    kubtest.ExecutionStatusSuceess,
		RawOutput: outputString,
	}
}

func getResponseCode(curlOutput string) int {
	re := regexp.MustCompile(`\A\S*\s(\d+)`)
	matches := re.FindStringSubmatch(curlOutput)
	result, _ := strconv.Atoi(matches[1])
	return result
}
