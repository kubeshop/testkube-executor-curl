package main

import (
	"github.com/kubeshop/kubtest-executor-curl-example/internal/app/executor"
)

func main() {

	exec := executor.NewTemplateExecutor()
	exec.Init()
	panic(exec.Run())

}
