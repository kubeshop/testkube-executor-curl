package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/process"
	"go.uber.org/zap"
)

const CurlAdditionalFlags = "-is"

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

func (r *CurlRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult) {
	var runnerInput CurlRunnerInput
	err := json.Unmarshal([]byte(execution.ScriptContent), &runnerInput)
	if err != nil {
		return result.Err(err)
	}

	err = FillCommandTemplates(runnerInput.Command, execution.Params)
	if err != nil {
		return result.Err(err)
	}

	command := runnerInput.Command[0]
	runnerInput.Command[0] = CurlAdditionalFlags
	output, err := process.Execute(command, runnerInput.Command...)
	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return result.Err(err)
	}

	outputString := string(output)
	result.Output = outputString
	responseStatus, err := getResponseCode(outputString)
	if err != nil {
		return result.Err(err)
	}
	if responseStatus != runnerInput.ExpectedStatus {
		return result.Err(fmt.Errorf("response statut don't match expected %d got %d", runnerInput.ExpectedStatus, responseStatus))
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		return result.Err(fmt.Errorf("response doesn't contain body: %s", runnerInput.ExpectedBody))
	}

	return testkube.ExecutionResult{
		Status: testkube.StatusPtr(testkube.SUCCESS_ExecutionStatus),
		Output: outputString,
	}
}

func getResponseCode(curlOutput string) (int, error) {
	re := regexp.MustCompile(`\A\S*\s(\d+)`)
	matches := re.FindStringSubmatch(curlOutput)
	if len(matches) == 0 {
		return -1, fmt.Errorf("could not find a response status in the command output")
	}
	return strconv.Atoi(matches[1])
}
