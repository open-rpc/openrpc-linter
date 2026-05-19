package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-rpc/openrpc-linter/cmd"
)

type lintScenario struct {
	dir         string
	openRPCFile string
	rulesFile   string
	docData     map[string]any
	output      bytes.Buffer
	err         error
}

// registerRecommendedSteps lets recommended_rules_test.go register additional
// steps on the same scenario without duplicating the godog wiring.
var registerRecommendedSteps func(*godog.ScenarioContext, *lintScenario)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func initializeScenario(sc *godog.ScenarioContext) {
	s := &lintScenario{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		s.dir = ""
		s.openRPCFile = ""
		s.rulesFile = ""
		s.docData = nil
		s.output.Reset()
		s.err = nil
		return ctx, nil
	})

	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if s.dir == "" {
			return ctx, nil
		}
		if err := os.RemoveAll(s.dir); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	sc.Step(`^a rules file with the methods length schema rule$`, func() error {
		name, err := s.writeTempFile("rules-*.yml", `description: "E2E rules"
rules:
  methods:
    description: "OpenRPC documents should define at least one method."
    given: "$.methods"
    severity: "error"
    then:
      function: "schema"
      functionOptions:
        type: "array"
        minItems: 1
`)
		if err != nil {
			return err
		}
		s.rulesFile = name
		return nil
	})

	sc.Step(`^an OpenRPC document with no methods$`, func() error {
		return s.writeOpenRPCDocument(`{
  "openrpc": "1.2.6",
  "info": {"title": "Test API", "version": "1.0.0"},
  "methods": []
}`)
	})

	sc.Step(`^an OpenRPC document with one method$`, func() error {
		return s.writeOpenRPCDocument(`{
  "openrpc": "1.2.6",
  "info": {"title": "Test API", "version": "1.0.0"},
  "methods": [
    {
      "name": "ping",
      "params": [],
      "result": {"name": "pong", "schema": {"type": "string"}}
    }
  ]
}`)
	})

	sc.Step(`^I run the linter$`, func() error {
		if s.docData != nil {
			return s.runWithDoc()
		}
		if s.openRPCFile == "" {
			return fmt.Errorf("OpenRPC document was not created")
		}
		if s.rulesFile == "" {
			return fmt.Errorf("rules file was not created")
		}
		s.output.Reset()
		s.err = cmd.RunLint(cmd.LintOptions{
			OpenRPCFile: s.openRPCFile,
			RulesFile:   s.rulesFile,
			Output:      &s.output,
			Format:      "text",
		})
		return nil
	})

	sc.Step(`^the lint should fail$`, func() error {
		if s.err != nil {
			return nil
		}
		return fmt.Errorf("expected lint to fail, output:\n%s", s.output.String())
	})

	sc.Step(`^the lint should pass$`, func() error {
		if s.err == nil {
			return nil
		}
		return fmt.Errorf("expected lint to pass, got %v\nOutput:\n%s", s.err, s.output.String())
	})

	sc.Step(`^the lint output should mention "([^"]*)"$`, func(expected string) error {
		if strings.Contains(s.output.String(), expected) {
			return nil
		}
		return fmt.Errorf("expected lint output to mention %q, got:\n%s", expected, s.output.String())
	})

	if registerRecommendedSteps != nil {
		registerRecommendedSteps(sc, s)
	}
}

func (s *lintScenario) writeOpenRPCDocument(content string) error {
	name, err := s.writeTempFile("openrpc-*.json", content)
	if err != nil {
		return err
	}
	s.openRPCFile = name
	return nil
}

func (s *lintScenario) writeTempFile(pattern, content string) (string, error) {
	if s.dir == "" {
		dir, err := os.MkdirTemp("", "openrpc-linter-e2e-*")
		if err != nil {
			return "", err
		}
		s.dir = dir
	}

	file, err := os.CreateTemp(s.dir, pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return "", err
	}

	return filepath.Clean(file.Name()), nil
}
