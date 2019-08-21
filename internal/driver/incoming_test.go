package driver

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"regexp"
	"testing"
)

func init() {
	instance = new(Driver)
	instance.Logger = logger.NewClient("test", false, "", "DEBUG")
}

func TestReplaceMessagePlaceholders(t *testing.T) {
	tests := []struct {
		input       string
		outputRegex string
	}{
		{
			input:       "{{uuid",
			outputRegex: "{{uuid",
		},
		{
			input:       "{{uuidd}}",
			outputRegex: "{{uuidd}}",
		},
		{
			input:       "the {{uuid}}",
			outputRegex: "the [-0-9a-f]{36}",
		},
		{
			input:       "middle{{uuid}}test",
			outputRegex: "middle[-0-9a-f]{36}test",
		},
		{
			input:       "{uuid}",
			outputRegex: "{uuid}",
		},
		{
			input:       "{\"id\":\"{{uuid}}\"}",
			outputRegex: "{\"id\":\"[-0-9a-f]{36}\"}",
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			regex := regexp.MustCompile(fmt.Sprintf("^%s$", test.outputRegex))
			output := replaceMessagePlaceholders(test.input)
			if !regex.MatchString(output) {
				t.Errorf("input: %s, output: %s, did not match expected regex: %s", test.input, output, test.outputRegex)
			}
		})
	}
}
