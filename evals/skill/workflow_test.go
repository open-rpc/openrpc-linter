package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Controlled commands exercise the workflow without invoking an agent service.
func TestEvalWorkflow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("workflow stubs use POSIX shell")
	}
	for _, scenario := range []string{"success", "build", "prepare", "skill", "prompt", "agent", "check", "lint"} {
		t.Run(scenario, func(t *testing.T) {
			root, _ := evalFixture(t)
			tools := t.TempDir()
			artifacts := t.TempDir()
			t.Chdir(root)
			t.Setenv("TMPDIR", artifacts)
			t.Setenv("EVAL_SCENARIO", scenario)
			t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
			evalWrite(t, filepath.Join(tools, "go"), `#!/bin/sh
if [ "$EVAL_SCENARIO" = build ]; then exit 1; fi
cat > "$3" <<'BIN'
#!/bin/sh
if [ "$1" = --skill ]; then
 if [ "$EVAL_SCENARIO" = skill ]; then exit 1; fi
 echo '# Controlled skill'
 exit 0
fi
[ "$EVAL_SCENARIO" != lint ]
BIN
chmod +x "$3"
if [ "$EVAL_SCENARIO" = prepare ]; then rm evals/skill/fixtures/openrpc.json; fi
if [ "$EVAL_SCENARIO" = prompt ]; then mkdir "$(dirname "$(dirname "$3")")/prompt.md"; fi
`)
			evalWrite(t, filepath.Join(tools, "codex"), `#!/bin/sh
if [ "$EVAL_SCENARIO" = agent ]; then exit 1; fi
if [ "$EVAL_SCENARIO" = check ]; then exit 0; fi
printf '%s' '{"methods":[{"name":"ping","description":"Returns pong."}]}' > openrpc.json
`)
			old := os.Args
			t.Cleanup(func() { os.Args = old })
			os.Args = []string{"eval"}
			if scenario == "success" {
				main()
				return
			}
			err := run()
			want := map[string]string{"build": "build:", "prepare": "openrpc.json", "skill": "read skill:", "prompt": "prompt.md", "agent": "agent:", "check": "mentioning pong", "lint": "final lint:"}[scenario]
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: got %v, want %q", scenario, err, want)
			}
		})
	}
}

func TestEvalRemovedWorkspaceAndSetupError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not permit removing the working directory")
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Error(err)
		}
	})
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("expected working directory error")
	}
	if err := os.Chdir(original); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	if err := run(); err == nil {
		t.Fatal("expected setup error")
	}
}
