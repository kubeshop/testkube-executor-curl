package executor

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kubeshop/kubtest-executor-curl/internal/pkg/storage"
	"github.com/kubeshop/kubtest-executor-curl/pkg/runner"
	"github.com/kubeshop/kubtest/pkg/executor/server"
	"github.com/stretchr/testify/assert"
)

func TestCurlExecutor_StartExecution(t *testing.T) {

	t.Run("Runs Curl executor with a get command.", func(t *testing.T) {
		// given
		executor := GetTestExecutor(t)

		req := httptest.NewRequest(
			"POST",
			"/v1/executions/",
			strings.NewReader(`
			{
				"type": "curl",
				"name": "test1",
				"metadata": "{\"command\": [\"curl\", \"https://reqbin.com/echo/get/json\", \"-H\", \"'Accept: application/json'\"],\"expected_status\":200,\"expected_body\":\"{\\\"success\\\":\\\"true\\\"}\"}"
			}`),
		)

		// when
		resp, err := executor.Mux.Test(req)
		assert.NoError(t, err)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
	})

}

func GetTestExecutor(t *testing.T) server.Executor {
	repo := storage.NewMapRepository()
	runner := runner.NewCurlRunner(nil)
	exec := server.NewExecutor(repo, runner)

	exec.Init()

	return exec
}
