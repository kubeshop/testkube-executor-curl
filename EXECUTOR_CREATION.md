# Creating a new kubtest executor
In order to be able to run tests using some new tools for which there is no executor, there is possibility to create a custom executor from the [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).

## Steps for creating executor

- Fork from [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template).
- Clone the newly created repo.
- Rename the go module from kubetest-executor-template to the new name & run `go mod tidy`.
- Rename TemplateExecutor to the name that will be used for new executor.
- Make the executor from the templated code
  The executor is a http server that serves the routes as defined in OpenApi definitions for [Executor](https://kubeshop.github.io/kubtest/openapi/#operations-tag-executor).
  Routes are bound to the `StartExecution` and `GetExecution` functions from the [executor.go](https://github.com/kubeshop/kubtest-executor-template/blob/main/internal/app/executor/executor.go)
  - `GetExecution` just serves the execution results from the repository used for storing(in the kubetest-executor-template it is implemented using mongo DB).
  - `StartExecution` can be implemented using a worker that will run the executions or simpler with a goroutine can be spawn for each execution and when the execution is done the result is stored in the repository.
    - Worker example can be found in the [kubetest-executor-template](https://github.com/kubeshop/kubtest-executor-template) -> [here](https://github.com/kubeshop/kubtest-executor-template/blob/main/internal/app/executor/executor.go)

    ```go
    func (p *TemplateExecutor) StartExecution() fiber.Handler {
        return func(c *fiber.Ctx) error {

            var request kubtest.ExecutionRequest
            err := json.Unmarshal(c.Body(), &request)
            if err != nil {
                return p.Error(c, http.StatusBadRequest, err)
            }

            execution := kubtest.NewExecution(string(request.Metadata), request.Params)
            err = p.Repository.Insert(context.Background(), execution)
            if err != nil {
                return p.Error(c, http.StatusInternalServerError, err)

            }

            p.Log.Infow("starting new execution", "execution", execution)
            c.Response().Header.SetStatusCode(201)
            return c.JSON(execution)
        }
    }
    ```

    - Goroutine example can be found in the [kubtest-executor-curl-example](https://github.com/kubeshop/kubtest-executor-curl-example) -> [here](https://github.com/kubeshop/kubtest-executor-curl-example/blob/main/internal/app/executor/executor.go) implemented for CURL command

    ```go
    func (p *CurlExecutor) StartExecution() fiber.Handler {
        return func(c *fiber.Ctx) error {

            var request kubtest.ExecutionRequest
            err := json.Unmarshal(c.Body(), &request)
            if err != nil {
                return p.Error(c, http.StatusBadRequest, err)
            }

            execution := kubtest.NewExecution(string(request.Metadata), request.Params)
            err = p.Repository.Insert(context.Background(), execution)
            if err != nil {
                return p.Error(c, http.StatusInternalServerError, err)

            }
            go func(ctx context.Context, e kubtest.Execution) {
                resultExecution, _ := p.RunExecution(ctx, e)
                p.Log.Infof("Execution with Id %s, returned %s", resultExecution.Id, resultExecution.Status)
            }(c.UserContext(), execution)

            p.Log.Infow("starting new execution", "execution", execution)
            c.Response().Header.SetStatusCode(201)
            return c.JSON(execution)
        }
    }
    ```

    Where `ExecutionRequest` is defined as:

    ```go
    // scripts execution request body
    type ExecutionRequest struct {
        // script type
        Type_ string `json:"type,omitempty"`
        // script execution custom name
        Name string `json:"name,omitempty"`
        // execution params passed to executor
        Params map[string]string `json:"params,omitempty"`
        // script content as string (content depends from executor)
        Metadata string `json:"metadata,omitempty"`
    }
    ```

    and the test script will be stored in the `Params` and `Metadata` fields.

- Define a input format for the tests.
  In order to communicate effectivele with executor we need to define a format on how to structure the tests. And bellow is an example for the curl based tests.

    ```js
    {
        "command": ["curl",
            "https://reqbin.com/echo/get/json",
            "-H",
            "'Accept: application/json'"
        ],
        "expected_status": 200,
        "expected_body": "{\"success\":\"true\"}"
    }
    ```

    It will be stored in the `Metadata` field of the request body and the request body will look like:

    ```js
    {
        "type": "curl",
        "name": "test1",
        "metadata": "{\"command\": [\"curl\", \"https://reqbin.com/echo/get/json\", \"-H\", \"'Accept: application/json'\"],\"expected_status\":200,\"expected_body\":\"{\\\"success\\\":\\\"true\\\"}\"}"
    }
    ```

- Create execution repository.
  The repository the interface bellow, it can use inmemory storage,a database or whatever fits the needs.

    ```go
    type Repository interface {
        // Get gets execution result by id
        Get(ctx context.Context, id string) (kubtest.Execution, error)
        // Insert inserts new execution result
        Insert(ctx context.Context, result kubtest.Execution) error
        // Update updates execution result
        Update(ctx context.Context, result kubtest.Execution) error
        // QueuePull pulls from queue and locks other clients to read (changes state from queued->pending)
        QueuePull(ctx context.Context) (kubtest.Execution, error)
    }
    ```

- Prepare docker for the type of the executor.
  In this step the docker should be configured to make sure that the runner has all dependencies installed and ready to use. 
  In the case of the [kubtest-executor-curl-example](https://github.com/kubeshop/kubtest-executor-curl-example) only installing curl was needed.

    ```docker
    FROM alpine
    RUN apk --no-cache add ca-certificates && \
        apk --no-cache add curl
    WORKDIR /root/
    COPY --from=0 /app /bin/app
    EXPOSE 8083
    ENTRYPOINT ["/bin/app"]
    ```

- Create new runner.
  Runner should contain the logic to run the test and to verify the expectations.
  The `StartExecute` function calls the `RunExecution` which calls the runner and stores the results in the repository

    ```go
    func (p CurlExecutor) RunExecution(ctx context.Context, e kubtest.Execution) (kubtest.Execution, error) {
        e.Start()
        runner := runner.CurlRunner{Log: p.Log}
        result := runner.Run(strings.NewReader(e.ScriptContent), e.Params)
        e.Result = &result

        var err error
        if result.ErrorMessage != "" {
            e.Error()
            err = fmt.Errorf("execution error: %s", result.ErrorMessage)
        } else {
            e.Success()
        }

        e.Stop()
        // we want always write even if there is error
        if werr := p.Repository.Update(ctx, e); werr != nil {
            return e, werr
        }

        return e, err
    }
    ```

  For the curl executor there is the struct that matches the exact structure of the test input format which runner will take as the input(described above).

    ```go
    type CurlRunnerInput struct {
        Command        []string `json:"command"`
        ExpectedStatus int      `json:"expected_status"`
        ExpectedBody   string   `json:"expected_body"`
    }
    ```

  And bellow is the business logic for the curl executor and it executes the curl command given as input, takes the output, tests the expectations and returns the result.

    ```go
    func (r *CurlRunner) Run(input io.Reader, params map[string]string) kubtest.ExecutionResult {
        var runnerInput CurlRunnerInput
        err := json.NewDecoder(input).Decode(&runnerInput)
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

    ```
