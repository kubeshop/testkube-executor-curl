package runner

import (
	"strings"
	"text/template"
)

// FillCommandTemplates fills the command array with the values if they are templated
func FillCommandTemplates(commandParts []string, params map[string]string) error {
	for i := range commandParts {
		finalCommandPart, err := FillCommandTemplate(commandParts[i], params)
		commandParts[i] = finalCommandPart
		if err != nil {
			return err
		}
	}

	return nil
}

// FillCommandTemplate fills a command with the values if they are templated
func FillCommandTemplate(command string, params map[string]string) (string, error) {

	ut, err := template.New("cmd").Parse(command)

	if err != nil {
		return "", err
	}
	writer := new(strings.Builder)
	err = ut.Execute(writer, params)

	if err != nil {
		return "", err
	}
	return writer.String(), nil
}
