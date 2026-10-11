package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

func main() {
	runMain(os.Exit)
}

func runMain(exit func(int)) {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exit(1)
	}
}

// executionEnvironment keeps the working directory and executable search path
// together for all commands in one evaluation workspace.
type executionEnvironment struct {
	directory string
	bin       string
}

func (e executionEnvironment) command(input string, args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = e.directory
	cmd.Env = append(os.Environ(), "PATH="+e.bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Stdin = strings.NewReader(input)
	return cmd.CombinedOutput()
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	directory, workspace, bin, err := setup()
	if err != nil {
		return err
	}
	binary := filepath.Join(bin, "openrpc-linter")
	output, err := (executionEnvironment{directory: root, bin: bin}).command("", "go", "build", "-o", binary, ".")
	if err != nil {
		return fmt.Errorf("build: %w\n%s", err, output)
	}
	if err := prepare(root, workspace); err != nil {
		return err
	}
	environment := executionEnvironment{directory: workspace, bin: bin}
	prompt, err := preparePrompt(environment, directory, binary)
	if err != nil {
		return err
	}
	if err := runAgent(environment, directory, prompt); err != nil {
		return err
	}
	if err := check(root, workspace); err != nil {
		return err
	}
	if err := environment.loggedCommand("", filepath.Join(directory, "lint.log"), "final lint", binary, "lint", "openrpc.json", "-r", "rules.yml"); err != nil {
		return err
	}
	fmt.Println("PASS: description added, other fields and rules preserved, lint passes.")
	fmt.Println("Review agent.log for CLI usage and description accuracy.")
	return nil
}

func setup() (string, string, string, error) {
	directory, err := os.MkdirTemp("", "openrpc-skill-eval-")
	if err != nil {
		return "", "", "", err
	}
	fmt.Println("Eval artifacts:", directory)
	workspace, bin, err := prepareDirectories(directory)
	return directory, workspace, bin, err
}

func prepareDirectories(directory string) (workspace, bin string, err error) {
	workspace = filepath.Join(directory, "workspace")
	bin = filepath.Join(directory, "bin")
	for _, dir := range []string{workspace, bin} {
		err = os.Mkdir(dir, 0700)
		if err != nil {
			return
		}
	}
	return
}

func prepare(root, workspace string) error {
	for _, file := range []string{"openrpc.json", "rules.yml"} {
		content, err := os.ReadFile(filepath.Join(root, "evals", "skill", "fixtures", file))
		if err != nil {
			return err
		}
		err = os.WriteFile(filepath.Join(workspace, file), content, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func runAgent(environment executionEnvironment, directory, prompt string) error {
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"codex", "exec", "--skip-git-repo-check", "--ephemeral", "--sandbox", "workspace-write", "-"}
	}
	return environment.loggedCommand(prompt, filepath.Join(directory, "agent.log"), "agent", args...)
}

func (e executionEnvironment) loggedCommand(input, logPath, label string, args ...string) error {
	output, commandErr := e.command(input, args...)
	err := os.WriteFile(logPath, output, 0600)
	if err != nil {
		return err
	}
	if commandErr != nil {
		return fmt.Errorf("%s: %w; see %s", label, commandErr, filepath.Base(logPath))
	}
	return nil
}

func preparePrompt(environment executionEnvironment, directory, binary string) (string, error) {
	skill, err := environment.command("", binary, "--skill")
	if err != nil {
		return "", fmt.Errorf("read skill: %w", err)
	}
	prompt := string(skill) + "\n\nUse the openrpc-linter skill to fix the lint failure in openrpc.json using the existing rules.yml. The ping method returns pong. Work in the current directory.\n"
	err = os.WriteFile(filepath.Join(directory, "prompt.md"), []byte(prompt), 0600)
	return prompt, err
}

func document(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	err = json.Unmarshal(data, &value)
	return value, err
}

func check(root, workspace string) error {
	fixtures := filepath.Join(root, "evals", "skill", "fixtures")
	original, err := document(filepath.Join(fixtures, "openrpc.json"))
	if err != nil {
		return err
	}
	updated, err := document(filepath.Join(workspace, "openrpc.json"))
	if err != nil {
		return err
	}
	if err := checkDocumentChange(original, updated); err != nil {
		return err
	}
	return checkRules(fixtures, workspace)
}

func checkDocumentChange(original, updated map[string]any) error {
	methods, _ := updated["methods"].([]any)
	if len(methods) != 1 {
		return fmt.Errorf("FAIL: expected the original ping method")
	}
	method, _ := methods[0].(map[string]any)
	description, _ := method["description"].(string)
	if !regexp.MustCompile(`(?i)\bpong\b`).MatchString(description) {
		return fmt.Errorf("FAIL: ping needs a description mentioning pong")
	}
	delete(method, "description")
	if !reflect.DeepEqual(original, updated) {
		return fmt.Errorf("FAIL: unrelated document changes")
	}
	return nil
}

func checkRules(fixtures, workspace string) error {
	before, err := os.ReadFile(filepath.Join(fixtures, "rules.yml"))
	if err != nil {
		return err
	}
	after, err := os.ReadFile(filepath.Join(workspace, "rules.yml"))
	if err != nil {
		return err
	}
	if string(before) != string(after) {
		return fmt.Errorf("FAIL: rules.yml changed")
	}
	return nil
}
