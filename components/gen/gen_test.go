package gen_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goforge.dev/cupel/components/gen"
	"goforge.dev/cupel/components/spec"
)

// fakeCompleter returns a canned reply, satisfying gen.Completer offline.
type fakeCompleter struct{ reply string }

func (f fakeCompleter) Complete(_ context.Context, _, _ string) (string, error) {
	return f.reply, nil
}

func userReq(t *testing.T, root string) gen.Request {
	t.Helper()
	s, err := spec.Parse("user", "", "A user has a non-empty name.")
	if err != nil {
		t.Fatalf("spec.Parse: %v", err)
	}
	return gen.Request{Spec: s, Module: "goforge.dev/demo", Root: root}
}

const validReply = "```go\npackage user_test\nimport \"testing\"\nfunc TestUserName(t *testing.T){_=t}\n```"

// Example test: a full run writes a valid component layout to disk.
func TestGenerate_WritesComponent(t *testing.T) {
	root := t.TempDir()
	res, err := gen.Generate(context.Background(), fakeCompleter{validReply}, userReq(t, root))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	testFile := filepath.Join(root, "components", "user", "user_test.go")
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("reading generated test: %v", err)
	}
	if got := string(data); !contains(got, "package user_test") || !contains(got, "func TestUserName") {
		t.Errorf("unexpected test file:\n%s", got)
	}
	if len(res.Written) == 0 {
		t.Error("nothing reported as written")
	}
}

// Example test: existing scaffolding is preserved, tests still regenerate.
func TestGenerate_SkipsExistingScaffold(t *testing.T) {
	root := t.TempDir()
	ifacePath := filepath.Join(root, "components", "user", "user.go")
	if err := os.MkdirAll(filepath.Dir(ifacePath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "package user // hand written\n"
	if err := os.WriteFile(ifacePath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := gen.Generate(context.Background(), fakeCompleter{validReply}, userReq(t, root))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got, _ := os.ReadFile(ifacePath)
	if string(got) != original {
		t.Error("existing interface file was overwritten")
	}
	if len(res.Skipped) == 0 {
		t.Error("expected the existing interface to be reported as skipped")
	}
}

// Example test: a malformed reply is rejected before anything is written.
func TestGenerate_RejectsBadReply(t *testing.T) {
	root := t.TempDir()
	_, err := gen.Generate(context.Background(), fakeCompleter{"package user\nnot tests"}, userReq(t, root))
	if err == nil {
		t.Fatal("expected error for wrong package")
	}
	if _, statErr := os.Stat(filepath.Join(root, "components", "user")); statErr == nil {
		t.Error("files were written despite invalid output")
	}
}

// Table-driven test: dry-run reports paths but writes nothing.
func TestGenerate_DryRun(t *testing.T) {
	root := t.TempDir()
	req := userReq(t, root)
	req.DryRun = true
	res, err := gen.Generate(context.Background(), fakeCompleter{validReply}, req)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(res.Written) == 0 {
		t.Error("dry-run should still report planned paths")
	}
	if _, statErr := os.Stat(filepath.Join(root, "components", "user")); statErr == nil {
		t.Error("dry-run wrote files to disk")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
