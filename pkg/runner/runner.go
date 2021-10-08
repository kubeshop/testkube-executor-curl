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

func (r *CurlRunner) Run(execution testkube.Execution) testkube.ExecutionResult {
	var runnerInput CurlRunnerInput
	err := json.Unmarshal([]byte(execution.ScriptContent), &runnerInput)
	if err != nil {
		return testkube.ExecutionResult{
			Status: testkube.ResultError,
		}
	}
	command := runnerInput.Command[0]
	runnerInput.Command[0] = CurlAdditionalFlags
	output, err := process.Execute(command, runnerInput.Command...)
	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return testkube.ExecutionResult{
			Status:       testkube.ResultError,
			ErrorMessage: fmt.Sprintf("Error occured when running a command %s", err),
		}
	}

	outputString := string(output)
	responseStatus, err := getResponseCode(outputString)
	if err != nil {
		return testkube.ExecutionResult{
			Status:       testkube.ResultError,
			Output:       outputString,
			ErrorMessage: err.Error(),
		}
	}
	if responseStatus != runnerInput.ExpectedStatus {
		return testkube.ExecutionResult{
			Status:       testkube.ResultError,
			Output:       outputString,
			ErrorMessage: fmt.Sprintf("Response statut don't match expected %d got %d", runnerInput.ExpectedStatus, responseStatus),
		}
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		return testkube.ExecutionResult{
			Status:       testkube.ResultError,
			Output:       outputString,
			ErrorMessage: fmt.Sprintf("Response doesn't contain body: %s", runnerInput.ExpectedBody),
		}
	}

	return testkube.ExecutionResult{
		Status: testkube.ResultSuceess,
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
