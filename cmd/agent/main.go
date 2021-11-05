package main

import (
	"os"

	"github.com/kubeshop/testkube-executor-curl/pkg/runner"
	"github.com/kubeshop/testkube/pkg/runner/agent"
)

func main() {
	agent.Run(runner.NewCurlRunner(), os.Args)
}
