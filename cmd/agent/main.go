package main

import (
	"fmt"
	"os"

	"github.com/kubeshop/testkube-executor-curl/pkg/runner"
	"github.com/kubeshop/testkube/pkg/executor/agent"
	"github.com/kubeshop/testkube/pkg/ui"
)

func main() {
	r, err := runner.NewCurlRunner()
	if err != nil {
		panic(fmt.Errorf("%s Could not run cURL tests: %w", ui.IconCross, err))
	}

	agent.Run(r, os.Args)
}
