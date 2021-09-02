package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/repository/result"
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/storage"
	"github.com/kubeshop/kubtest-executor-curl-example/pkg/runner"

	// TODO move server to kubtest/pkg
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/server"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewCurlExecutor returns new CurlExecutor instance
func NewCurlExecutor() CurlExecutor {
	var httpConfig server.Config
	envconfig.Process("EXECUTOR_PORT", &httpConfig)

	e := CurlExecutor{
		HTTPServer: server.NewServer(httpConfig),
		Repository: &storage.MapRepository{},
	}

	return e
}

type CurlExecutor struct {
	server.HTTPServer
	Repository result.Repository
}

func (p *CurlExecutor) Init() {
	executions := p.Routes.Group("/executions")
	executions.Post("/", p.StartExecution())
	executions.Get("/:id", p.GetExecution())
}

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

func (p CurlExecutor) GetExecution() fiber.Handler {
	return func(c *fiber.Ctx) error {
		execution, err := p.Repository.Get(context.Background(), c.Params("id"))
		if err != nil {
			return p.Error(c, http.StatusInternalServerError, err)
		}

		return c.JSON(execution)
	}
}

func (p CurlExecutor) Run() error {
	return p.HTTPServer.Run()
}

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
