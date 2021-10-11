```
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop

                    EXECUTOR CURL
```

<!-- try to enable it after snyk resolves https://github.com/snyk/snyk/issues/347

Known vulnerabilities: ![testkube](https://snyk.io/test/github/kubeshop/testkube/badge.svg)
![testkube-operator](https://snyk.io/test/github/kubeshop-operator/testkube/badge.svg)
![helm-charts](https://snyk.io/test/github/kubeshop/helm-charts/badge.svg)
-->

# Welcome to testkube Executor Curl

testkube Executor Curl is the test executor for [testkube](https://testkube.io) that is using [Curl](https://curl.se/).

# Issues and enchancements

Please follow to main testkube repository for reporting any [issues](https://github.com/kubeshop/testkube/issues) or [discussions](https://github.com/kubeshop/testkube/discussions)

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

testkube Executor Curl implements [testkube OpenAPI for executors](https://kubeshop.github.io/testkube/openapi/#operations-tag-executor) (look at executor tag)
