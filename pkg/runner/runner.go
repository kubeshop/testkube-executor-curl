package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor"
	contentPkg "github.com/kubeshop/testkube/pkg/executor/content"
	outputPkg "github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/executor/runner"
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/ui"
	"go.uber.org/zap"
)

// Params ...
type Params struct {
	Endpoint        string // RUNNER_ENDPOINT
	AccessKeyID     string // RUNNER_ACCESSKEYID
	SecretAccessKey string // RUNNER_SECRETACCESSKEY
	Location        string // RUNNER_LOCATION
	Token           string // RUNNER_TOKEN
	Ssl             bool   // RUNNER_SSL
	ScrapperEnabled bool   // RUNNER_SCRAPPERENABLED
	GitUsername     string // RUNNER_GITUSERNAME
	GitToken        string // RUNNER_GITTOKEN
	DataDir         string // RUNNER_DATADIR
}

const CurlAdditionalFlags = "-is"

// CurlRunner is used to run curl commands.
type CurlRunner struct {
	Params  Params
	Fetcher contentPkg.ContentFetcher
	Log     *zap.SugaredLogger
}

func NewCurlRunner() *CurlRunner {
	outputPkg.PrintEvent(fmt.Sprintf("%s Preparing test runner", ui.IconTruck))
	var params Params

	outputPkg.PrintEvent(fmt.Sprintf("%s Reading environment variables...", ui.IconWorld))
	err := envconfig.Process("runner", &params)
	if err != nil {
		outputPkg.PrintEvent(fmt.Sprintf("%s Failed to read environment variables: %s", ui.IconCross, err.Error()))
		panic(err.Error())
	}
	outputPkg.PrintEvent(fmt.Sprintf("%s Environment variables read successfully", ui.IconCheckMark))
	printParams(params)

	return &CurlRunner{
		Log:     log.DefaultLogger,
		Params:  params,
		Fetcher: contentPkg.NewFetcher(""),
	}
}

func (r *CurlRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	outputPkg.PrintEvent(fmt.Sprintf("%s Preparing for test run", ui.IconTruck))
	var runnerInput CurlRunnerInput
	if r.Params.GitUsername != "" || r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	outputPkg.PrintEvent(fmt.Sprintf("%s Fetching test content from %s...", ui.IconBox, execution.Content.Type_))
	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		outputPkg.PrintEvent(fmt.Sprintf("%s Could not fetch test content: %s", ui.IconCross, err.Error()))
		return result, err
	}
	outputPkg.PrintEvent(fmt.Sprintf("%s Test content fetched to path %s", ui.IconCheckMark, path))

	if !execution.Content.IsFile() {
		return result, testkube.ErrTestContentTypeNotFile
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(content, &runnerInput)
	if err != nil {
		return result, err
	}

	envManager := secret.NewEnvManagerWithVars(execution.Variables)
	envManager.GetVars(envManager.Variables)
	variables := testkube.VariablesToMap(envManager.Variables)
	err = runnerInput.FillTemplates(variables)
	if err != nil {
		r.Log.Errorf("Error occured when resolving input templates %s", err)
		return result.Err(err), nil
	}

	command := runnerInput.Command[0]
	if command != "curl" {
		return result, fmt.Errorf("you can run only `curl` commands with this executor but passed: `%s`", command)
	}

	runnerInput.Command[0] = CurlAdditionalFlags

	args := runnerInput.Command
	args = append(args, execution.Args...)

	runPath := ""
	if execution.Content.Repository != nil && execution.Content.Repository.WorkingDir != "" {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	outputPkg.PrintEvent(fmt.Sprintf("%s executing test\n\t$ %s %s", ui.IconMicroscope, command, strings.Join(args, " ")))
	output, err := executor.Run(runPath, command, envManager, args...)
	output = envManager.Obfuscate(output)

	if err != nil {
		outputPkg.PrintEvent(fmt.Sprintf("%s Error occured when running a command %s", ui.IconCross, err.Error()))
		r.Log.Errorf("Error occured when running a command %s", err)
		return result.Err(err), nil
	}

	outputString := string(output)
	result.Output = outputString
	responseStatus, err := getResponseCode(outputString)
	if err != nil {
		outputPkg.PrintEvent(fmt.Sprintf("%s Test run failed: %s", ui.IconCross, err.Error()))
		return result.Err(err), nil
	}

	expectedStatus, err := strconv.Atoi(runnerInput.ExpectedStatus)
	if err != nil {
		outputPkg.PrintEvent(fmt.Sprintf("%s Test run failed: cannot process expected status: %s", ui.IconCross, err.Error()))
		return result.Err(fmt.Errorf("cannot process expected status %s", runnerInput.ExpectedStatus)), nil
	}

	if responseStatus != expectedStatus {
		outputPkg.PrintEvent(fmt.Sprintf("%s Test run failed: cannot process expected status: %s", ui.IconCross, err.Error()))
		return result.Err(fmt.Errorf("response status don't match expected %d got %d", expectedStatus, responseStatus)), nil
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		outputPkg.PrintEvent(fmt.Sprintf("%s Test run failed: response doesn't contain body: %s", ui.IconCross, runnerInput.ExpectedBody))
		return result.Err(fmt.Errorf("response doesn't contain body: %s", runnerInput.ExpectedBody)), nil
	}

	outputPkg.PrintEvent(fmt.Sprintf("%s Test run succeeded", ui.IconCheckMark))

	return testkube.ExecutionResult{
		Status: testkube.ExecutionStatusPassed,
		Output: outputString,
	}, nil
}

func getResponseCode(curlOutput string) (int, error) {
	re := regexp.MustCompile(`\A\S*\s(\d+)`)
	matches := re.FindStringSubmatch(curlOutput)
	if len(matches) == 0 {
		return -1, fmt.Errorf("could not find a response status in the command output")
	}
	return strconv.Atoi(matches[1])
}

// GetType returns runner type
func (r *CurlRunner) GetType() runner.Type {
	return runner.TypeMain
}

// printParams shows the read parameters in logs
func printParams(params Params) {
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_ENDPOINT=\"%s\"", params.Endpoint))
	printSensitiveParam("RUNNER_ACCESSKEYID", params.AccessKeyID)
	printSensitiveParam("RUNNER_SECRETACCESSKEY", params.SecretAccessKey)
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_LOCATION=\"%s\"", params.Location))
	printSensitiveParam("RUNNER_TOKEN", params.Token)
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_SSL=%t", params.Ssl))
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_SCRAPPERENABLED=\"%t\"", params.ScrapperEnabled))
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_GITUSERNAME=\"%s\"", params.GitUsername))
	printSensitiveParam("RUNNER_GITTOKEN", params.GitToken)
	outputPkg.PrintLog(fmt.Sprintf("RUNNER_DATADIR=\"%s\"", params.DataDir))
}

// printSensitiveParam shows in logs if a parameter is set or not
func printSensitiveParam(name string, value string) {
	if len(value) == 0 {
		outputPkg.PrintLog(fmt.Sprintf("%s=\"\"", name))
	} else {
		outputPkg.PrintLog(fmt.Sprintf("%s=\"********\"", name))
	}
}
