package main

import (
	"github.com/kubeshop/testkube-executor-curl/internal/pkg/storage"
	"github.com/kubeshop/testkube-executor-curl/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/server"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {

	repo := storage.NewMapRepository()
	runner := runner.NewCurlRunner(nil)
	exec := server.NewExecutor(repo, runner)

	ui.ExitOnError("Running executor", exec.Init().Run())
}
