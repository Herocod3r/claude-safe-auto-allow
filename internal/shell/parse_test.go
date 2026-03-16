package shell

import (
	"fmt"
	"testing"
)

func TestParseBasic(t *testing.T) {
	cases := []struct {
		input string
		name  string
		args  []string
	}{
		{"rm -rf /", "rm", []string{"-rf", "/"}},
		{"ls -la", "ls", []string{"-la"}},
		{"git status", "git", []string{"status"}},
	}
	for _, tc := range cases {
		r := Parse(tc.input)
		if r.ParseError || len(r.Commands) == 0 {
			t.Errorf("Parse(%q): parse error or no commands", tc.input)
			continue
		}
		cmd := r.Commands[0]
		if cmd.Name != tc.name {
			t.Errorf("Parse(%q).Name = %q, want %q", tc.input, cmd.Name, tc.name)
		}
		if fmt.Sprintf("%v", cmd.Args) != fmt.Sprintf("%v", tc.args) {
			t.Errorf("Parse(%q).Args = %v, want %v", tc.input, cmd.Args, tc.args)
		}
	}
}

func TestParseSubstitution(t *testing.T) {
	r := Parse("echo $(whoami)")
	if !r.HasSubstitution {
		t.Error("should detect $() substitution")
	}
	r2 := Parse("echo `whoami`")
	if !r2.HasSubstitution {
		t.Error("should detect backtick substitution")
	}
	r3 := Parse("echo hello")
	if r3.HasSubstitution {
		t.Error("should not detect substitution in plain command")
	}
}

func TestParseForkBomb(t *testing.T) {
	r := Parse(":(){ :|:& };:")
	if !r.HasForkBomb {
		t.Error("should detect fork bomb function declaration")
	}
}

func TestParseSudo(t *testing.T) {
	r := Parse("sudo rm -rf /")
	if r.ParseError || len(r.Commands) == 0 {
		t.Fatal("failed to parse sudo command")
	}
	cmd := r.Commands[0]
	t.Logf("sudo rm: Name=%q Args=%v", cmd.Name, cmd.Args)
	if cmd.Name != "sudo" {
		t.Errorf("Name = %q, want sudo", cmd.Name)
	}
}

func TestParseQuotedArgs(t *testing.T) {
	r := Parse(`mysql -e "DROP DATABASE prod"`)
	if r.ParseError || len(r.Commands) == 0 {
		t.Fatal("parse error")
	}
	cmd := r.Commands[0]
	t.Logf("mysql: Name=%q Args=%v", cmd.Name, cmd.Args)
}

func TestParseNonLiteralArg(t *testing.T) {
	r := Parse("rm -rf $HOME")
	if r.ParseError || len(r.Commands) == 0 {
		t.Fatal("parse error")
	}
	cmd := r.Commands[0]
	t.Logf("rm $HOME: Name=%q Args=%v HasNonLiteral=%v", cmd.Name, cmd.Args, cmd.HasNonLiteralArg)
	if !cmd.HasNonLiteralArg {
		t.Error("should have HasNonLiteralArg=true for $HOME")
	}
}

func TestParseProcSubstitution(t *testing.T) {
	// <(...) process substitution must be treated the same as $() — we can't
	// statically verify the inner command, so HasSubstitution must be true.
	cases := []string{
		"diff <(ls) <(ls ~/Documents)",
		"cat <(echo hello)",
		"cmd >(tee out.txt)",
	}
	for _, input := range cases {
		r := Parse(input)
		if !r.HasSubstitution {
			t.Errorf("Parse(%q): HasSubstitution = false, want true", input)
		}
	}
}

func TestParseChained(t *testing.T) {
	r := Parse("ls && echo done")
	t.Logf("chained: Commands=%v", r.Commands)
	if len(r.Commands) != 2 {
		t.Errorf("want 2 commands, got %d", len(r.Commands))
	}
}
