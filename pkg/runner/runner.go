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
	"github.com/kubeshop/testkube/pkg/executor/secret"
	"github.com/kubeshop/testkube/pkg/log"
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
	var params Params
	err := envconfig.Process("runner", &params)
	if err != nil {
		panic(err.Error())
	}

	return &CurlRunner{
		Log:     log.DefaultLogger,
		Params:  params,
		Fetcher: contentPkg.NewFetcher(""),
	}
}

func (r *CurlRunner) Run(execution testkube.Execution) (result testkube.ExecutionResult, err error) {
	var runnerInput CurlRunnerInput
	if r.Params.GitUsername != "" && r.Params.GitToken != "" {
		if execution.Content != nil && execution.Content.Repository != nil {
			execution.Content.Repository.Username = r.Params.GitUsername
			execution.Content.Repository.Token = r.Params.GitToken
		}
	}

	path, err := r.Fetcher.Fetch(execution.Content)
	if err != nil {
		return result, err
	}

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
	envManager.GetVars(execution.Variables)
	variables := testkube.VariablesToMap(execution.Variables)
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
	if execution.Content.Repository != nil {
		runPath = filepath.Join(r.Params.DataDir, "repo", execution.Content.Repository.WorkingDir)
	}

	output, err := executor.Run(runPath, command, envManager, args...)
	output = envManager.Obfuscate(output)

	if err != nil {
		r.Log.Errorf("Error occured when running a command %s", err)
		return result.Err(err), nil
	}

	outputString := string(output)
	result.Output = outputString
	responseStatus, err := getResponseCode(outputString)
	if err != nil {
		return result.Err(err), nil
	}

	expectedStatus, err := strconv.Atoi(runnerInput.ExpectedStatus)
	if err != nil {
		return result.Err(fmt.Errorf("cannot process expected status %s", runnerInput.ExpectedStatus)), nil
	}

	if responseStatus != expectedStatus {
		return result.Err(fmt.Errorf("response statut don't match expected %d got %d", expectedStatus, responseStatus)), nil
	}

	if !strings.Contains(outputString, runnerInput.ExpectedBody) {
		return result.Err(fmt.Errorf("response doesn't contain body: %s", runnerInput.ExpectedBody)), nil
	}

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
