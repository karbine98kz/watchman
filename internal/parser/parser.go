// Package parser provides shell command parsing utilities.
package parser

import (
	"regexp"
	"strings"
)

// Command represents a parsed shell command.
type Command struct {
	Raw        string
	Env        map[string]string
	Program    string
	Subcommand string
	Args       []string
	Flags      map[string]string
}

var envVarPattern = regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)=(.*)$`)

// Parse parses a shell command string into its components.
func Parse(cmd string) Command {
	result := Command{
		Raw:   cmd,
		Env:   make(map[string]string),
		Args:  make([]string, 0),
		Flags: make(map[string]string),
	}

	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return result
	}

	tokens := tokenize(cmd)
	if len(tokens) == 0 {
		return result
	}

	idx := 0

	// Extract leading environment variables
	for idx < len(tokens) {
		if match := envVarPattern.FindStringSubmatch(tokens[idx]); match != nil {
			result.Env[match[1]] = match[2]
			idx++
		} else {
			break
		}
	}

	if idx >= len(tokens) {
		return result
	}

	// Program name
	result.Program = tokens[idx]
	idx++

	// Check for subcommand (non-flag argument immediately after program)
	if idx < len(tokens) && !strings.HasPrefix(tokens[idx], "-") {
		if hasSubcommand(result.Program) {
			result.Subcommand = tokens[idx]
			idx++
		}
	}

	// Parse remaining tokens as flags and args
	for idx < len(tokens) {
		token := tokens[idx]
		if strings.HasPrefix(token, "-") {
			key, value := parseFlag(token)
			// Check for value in next token if flag has no embedded value
			if value == "" && idx+1 < len(tokens) && !strings.HasPrefix(tokens[idx+1], "-") {
				next := tokens[idx+1]
				if !strings.HasPrefix(next, ".") && !strings.HasPrefix(next, "/") && !strings.Contains(next, "/") {
					value = next
					idx++
				}
			}
			result.Flags[key] = value
		} else {
			result.Args = append(result.Args, token)
		}
		idx++
	}

	return result
}

// HasFlag returns true if the command has the specified flag.
func (c Command) HasFlag(flag string) bool {
	normalized := strings.TrimLeft(flag, "-")
	for key := range c.Flags {
		if strings.TrimLeft(key, "-") == normalized {
			return true
		}
	}
	return false
}

// FlagValue returns the value of a flag and whether it exists.
func (c Command) FlagValue(flag string) (string, bool) {
	normalized := strings.TrimLeft(flag, "-")
	for key, value := range c.Flags {
		if strings.TrimLeft(key, "-") == normalized {
			return value, true
		}
	}
	return "", false
}

// HasEnv returns true if the command has the specified environment variable.
func (c Command) HasEnv(name string) bool {
	_, ok := c.Env[name]
	return ok
}

// EnvValue returns the value of an environment variable and whether it exists.
func (c Command) EnvValue(name string) (string, bool) {
	value, ok := c.Env[name]
	return value, ok
}

// String returns the original raw command.
func (c Command) String() string {
	return c.Raw
}

// tokenize splits a command string into tokens, respecting quotes.
func tokenize(cmd string) []string {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			if inSingleQuote {
				current.WriteRune(r)
			} else {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			} else {
				current.WriteRune(r)
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			} else {
				current.WriteRune(r)
			}
		case ' ', '\t':
			if inSingleQuote || inDoubleQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseFlag parses a flag token into key and value.
func parseFlag(token string) (string, string) {
	if idx := strings.Index(token, "="); idx != -1 {
		return token[:idx], token[idx+1:]
	}
	return token, ""
}

// hasSubcommand returns true if the program typically has subcommands.
func hasSubcommand(program string) bool {
	programs := map[string]bool{
		"go": true, "git": true, "make": true, "docker": true,
		"kubectl": true, "npm": true, "yarn": true, "cargo": true,
	}
	return programs[program]
}
