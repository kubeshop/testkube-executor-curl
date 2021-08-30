package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

type CurlRunnerInput struct {
	Command        []string `json:"command"`
	ExpectedStatus int      `json:"expected_status"`
	ExpectedBody   string   `json:"expected_body"`
}

// CurlRunner is used to run curl commands.
type CurlRunner struct {
}

func (r *CurlRunner) Run(input io.Reader, params map[string]string) kubtest.ExecutionResult {
	var runnerInput CurlRunnerInput
	err := json.NewDecoder(input).Decode(&runnerInput)
	if err != nil {
		return kubtest.ExecutionResult{
			Status: kubtest.ExecutionStatusError,
		}
	}

	output, err := exec.Command(runnerInput.Command[0], runnerInput.Command[1:]...).Output()
	if err != nil {
		log.Fatal(err)
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
