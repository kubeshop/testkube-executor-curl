```
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                               /kʌb tɛst/ by Kubeshop
                    EXECUTOR CURL
```

<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347

Known vulnerabilities: ![kubtest](https://snyk.io/test/github/kubeshop/kubtest/badge.svg)
![kubtest-operator](https://snyk.io/test/github/kubeshop-operator/kubtest/badge.svg)
![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)
-->

# Welcome to Kubtest Executor Curl

Kubtest Executor Curl is the test executor for [Kubtest](https://kubtest.io) that is using [Curl](https://curl.se/).

# Issues and enchancements

Please follow to main kubtest repository for reporting any [issues](https://github.com/kubeshop/kubtest/issues) or [discussions](https://github.com/kubeshop/kubtest/discussions)

## Details

Curl executor is a very simple one, it runs a curl command given as the input and check the response for expected status and body, the input is of form

```js
{
  "command": [
    "curl",
    "https://reqbin.com/echo/get/json",
    "-H",
    "'Accept: application/json'"
  ],
  "expected_status": 200,
  "expected_body": "{\"success\":\"true\"}"
}
```

the executor will check if the response has `expected_status` and if body of the response contains the `expected_body`.

The type of the test CRD should be `curl/test`.

## API

Kubtest Executor Curl implements [Kubtest OpenAPI for executors](https://kubeshop.github.io/kubtest/openapi/#operations-tag-executor) (look at executor tag)
