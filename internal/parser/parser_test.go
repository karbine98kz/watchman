package parser

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want Command
	}{
		{
			name: "empty command",
			cmd:  "",
			want: Command{Raw: "", Env: map[string]string{}, Args: []string{}, Flags: map[string]string{}},
		},
		{
			name: "whitespace only",
			cmd:  "   ",
			want: Command{Raw: "   ", Env: map[string]string{}, Args: []string{}, Flags: map[string]string{}},
		},
		{
			name: "simple program",
			cmd:  "ls",
			want: Command{Raw: "ls", Env: map[string]string{}, Program: "ls", Args: []string{}, Flags: map[string]string{}},
		},
		{
			name: "program with args",
			cmd:  "ls -la /tmp",
			want: Command{Raw: "ls -la /tmp", Env: map[string]string{}, Program: "ls", Args: []string{"/tmp"}, Flags: map[string]string{"-la": ""}},
		},
		{
			name: "go test with flags",
			cmd:  "go test -race -count=1 ./pkg/...",
			want: Command{Raw: "go test -race -count=1 ./pkg/...", Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./pkg/..."}, Flags: map[string]string{"-race": "", "-count": "1"}},
		},
		{
			name: "go test all packages",
			cmd:  "go test ./...",
			want: Command{Raw: "go test ./...", Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./..."}, Flags: map[string]string{}},
		},
		{
			name: "env var prefix",
			cmd:  "CGO_ENABLED=0 go build ./...",
			want: Command{Raw: "CGO_ENABLED=0 go build ./...", Env: map[string]string{"CGO_ENABLED": "0"}, Program: "go", Subcommand: "build", Args: []string{"./..."}, Flags: map[string]string{}},
		},
		{
			name: "multiple env vars",
			cmd:  "GOMODCACHE=/tmp/mod GOCACHE=/tmp/cache go test ./...",
			want: Command{Raw: "GOMODCACHE=/tmp/mod GOCACHE=/tmp/cache go test ./...", Env: map[string]string{"GOMODCACHE": "/tmp/mod", "GOCACHE": "/tmp/cache"}, Program: "go", Subcommand: "test", Args: []string{"./..."}, Flags: map[string]string{}},
		},
		{
			name: "git commit with message",
			cmd:  "git commit -m 'Fix bug'",
			want: Command{Raw: "git commit -m 'Fix bug'", Env: map[string]string{}, Program: "git", Subcommand: "commit", Args: []string{}, Flags: map[string]string{"-m": "Fix bug"}},
		},
		{
			name: "double quoted argument",
			cmd:  `go test -run "TestFoo" ./pkg`,
			want: Command{Raw: `go test -run "TestFoo" ./pkg`, Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./pkg"}, Flags: map[string]string{"-run": "TestFoo"}},
		},
		{
			name: "go test with bench",
			cmd:  "go test -bench=. -benchmem ./...",
			want: Command{Raw: "go test -bench=. -benchmem ./...", Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./..."}, Flags: map[string]string{"-bench": ".", "-benchmem": ""}},
		},
		{
			name: "long flags",
			cmd:  "go test --verbose --count=5 ./...",
			want: Command{Raw: "go test --verbose --count=5 ./...", Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./..."}, Flags: map[string]string{"--verbose": "", "--count": "5"}},
		},
		{
			name: "make target",
			cmd:  "make test",
			want: Command{Raw: "make test", Env: map[string]string{}, Program: "make", Subcommand: "test", Args: []string{}, Flags: map[string]string{}},
		},
		{
			name: "env var only",
			cmd:  "FOO=bar",
			want: Command{Raw: "FOO=bar", Env: map[string]string{"FOO": "bar"}, Args: []string{}, Flags: map[string]string{}},
		},
		{
			name: "escaped space",
			cmd:  `echo hello\ world`,
			want: Command{Raw: `echo hello\ world`, Env: map[string]string{}, Program: "echo", Args: []string{"hello world"}, Flags: map[string]string{}},
		},
		{
			name: "go test short mode",
			cmd:  "go test -short ./...",
			want: Command{Raw: "go test -short ./...", Env: map[string]string{}, Program: "go", Subcommand: "test", Args: []string{"./..."}, Flags: map[string]string{"-short": ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.cmd)
			if got.Raw != tt.want.Raw {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.want.Raw)
			}
			if got.Program != tt.want.Program {
				t.Errorf("Program = %q, want %q", got.Program, tt.want.Program)
			}
			if got.Subcommand != tt.want.Subcommand {
				t.Errorf("Subcommand = %q, want %q", got.Subcommand, tt.want.Subcommand)
			}
			if !reflect.DeepEqual(got.Env, tt.want.Env) {
				t.Errorf("Env = %v, want %v", got.Env, tt.want.Env)
			}
			if !reflect.DeepEqual(got.Args, tt.want.Args) {
				t.Errorf("Args = %v, want %v", got.Args, tt.want.Args)
			}
			if !reflect.DeepEqual(got.Flags, tt.want.Flags) {
				t.Errorf("Flags = %v, want %v", got.Flags, tt.want.Flags)
			}
		})
	}
}

func TestCommandHasFlag(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		flag string
		want bool
	}{
		{"has flag with dash", Command{Flags: map[string]string{"-race": ""}}, "-race", true},
		{"has flag without dash", Command{Flags: map[string]string{"-race": ""}}, "race", true},
		{"has long flag", Command{Flags: map[string]string{"--verbose": ""}}, "--verbose", true},
		{"missing flag", Command{Flags: map[string]string{"-race": ""}}, "-cover", false},
		{"empty flags", Command{Flags: map[string]string{}}, "-race", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.HasFlag(tt.flag); got != tt.want {
				t.Errorf("HasFlag(%q) = %v, want %v", tt.flag, got, tt.want)
			}
		})
	}
}

func TestCommandFlagValue(t *testing.T) {
	tests := []struct {
		name      string
		cmd       Command
		flag      string
		wantValue string
		wantOK    bool
	}{
		{"flag with value", Command{Flags: map[string]string{"-count": "1"}}, "-count", "1", true},
		{"flag without dash", Command{Flags: map[string]string{"-count": "5"}}, "count", "5", true},
		{"flag no value", Command{Flags: map[string]string{"-race": ""}}, "-race", "", true},
		{"missing flag", Command{Flags: map[string]string{"-race": ""}}, "-count", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := tt.cmd.FlagValue(tt.flag)
			if v != tt.wantValue || ok != tt.wantOK {
				t.Errorf("FlagValue(%q) = (%q, %v), want (%q, %v)", tt.flag, v, ok, tt.wantValue, tt.wantOK)
			}
		})
	}
}

func TestCommandHasEnv(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		env  string
		want bool
	}{
		{"has env", Command{Env: map[string]string{"CGO_ENABLED": "0"}}, "CGO_ENABLED", true},
		{"missing env", Command{Env: map[string]string{"CGO_ENABLED": "0"}}, "GOMODCACHE", false},
		{"empty env", Command{Env: map[string]string{}}, "CGO_ENABLED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.HasEnv(tt.env); got != tt.want {
				t.Errorf("HasEnv(%q) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}

func TestCommandEnvValue(t *testing.T) {
	tests := []struct {
		name      string
		cmd       Command
		env       string
		wantValue string
		wantOK    bool
	}{
		{"env present", Command{Env: map[string]string{"CGO_ENABLED": "0"}}, "CGO_ENABLED", "0", true},
		{"env with path", Command{Env: map[string]string{"GOMODCACHE": "/tmp/mod"}}, "GOMODCACHE", "/tmp/mod", true},
		{"missing env", Command{Env: map[string]string{}}, "CGO_ENABLED", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, ok := tt.cmd.EnvValue(tt.env)
			if v != tt.wantValue || ok != tt.wantOK {
				t.Errorf("EnvValue(%q) = (%q, %v), want (%q, %v)", tt.env, v, ok, tt.wantValue, tt.wantOK)
			}
		})
	}
}

func TestCommandString(t *testing.T) {
	cmd := Command{Raw: "go test ./..."}
	if got := cmd.String(); got != "go test ./..." {
		t.Errorf("String() = %q, want %q", got, "go test ./...")
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		cmd  string
		want []string
	}{
		{"go test ./...", []string{"go", "test", "./..."}},
		{"go\ttest\t./...", []string{"go", "test", "./..."}},
		{"go   test   ./...", []string{"go", "test", "./..."}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			if got := tokenize(tt.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestHasSubcommand(t *testing.T) {
	yes := []string{"go", "git", "make", "docker", "kubectl", "npm", "yarn", "cargo"}
	no := []string{"ls", "cat", "echo", "grep"}

	for _, p := range yes {
		if !hasSubcommand(p) {
			t.Errorf("hasSubcommand(%q) = false, want true", p)
		}
	}
	for _, p := range no {
		if hasSubcommand(p) {
			t.Errorf("hasSubcommand(%q) = true, want false", p)
		}
	}
}
