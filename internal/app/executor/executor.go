package executor

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/repository/result"
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/worker"

	// TODO move server to kubtest/pkg
	"github.com/kubeshop/kubtest-executor-curl-example/internal/pkg/server"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

// ConcurrentExecutions per node
const ConcurrentExecutions = 4

// NewCurlExecutor returns new CurlExecutor instance
func NewCurlExecutor() CurlExecutor {
	var httpConfig server.Config
	envconfig.Process("EXECUTOR", &httpConfig)

	e := CurlExecutor{
		HTTPServer: server.NewServer(httpConfig),
		Worker:     worker.NewWorker(),
	}

	return e
}

type CurlExecutor struct {
	server.HTTPServer
	Repository result.Repository
	Worker     worker.Worker
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
	executionsQueue := p.Worker.PullExecutions()
	p.Worker.Run(executionsQueue)

	return p.HTTPServer.Run()
}
