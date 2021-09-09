package main

import (
	"github.com/kubeshop/kubtest-executor-curl/internal/pkg/storage"
	"github.com/kubeshop/kubtest-executor-curl/pkg/runner"
	"github.com/kubeshop/kubtest/pkg/executor/server"
	"github.com/kubeshop/kubtest/pkg/ui"
)

func main() {

	repo := storage.NewMapRepository()
	runner := runner.NewCurlRunner(nil)
	exec := server.NewExecutor(repo, runner)

	ui.ExitOnError("Running executor", exec.Init().Run())
}
