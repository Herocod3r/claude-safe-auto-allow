// Package shell parses POSIX shell commands into structured Command objects
// using the mvdan.cc/sh/v3 AST parser instead of regex-based splitting.
package shell

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Command is a parsed shell command with its statically-resolved name and args.
// Name is empty if the command name contains expansions (e.g. $CMD or $(which rm)).
// Args contains only literal values; non-literal args are excluded but tracked via HasNonLiteralArg.
type Command struct {
	Name            string
	Args            []string
	HasNonLiteralArg bool // true if any argument (not the name) could not be resolved
}

// Redirect represents an output redirection from a shell statement.
type Redirect struct {
	Target string // literal target path; empty if non-literal
}

// ParseResult holds all information extracted from parsing a shell input string.
type ParseResult struct {
	Commands        []Command
	Redirects       []Redirect
	HasSubstitution bool // true if any $() or `` command substitution is present
	HasForkBomb     bool // true if a function named ":" is declared (classic fork bomb)
	ParseError      bool // true if the input could not be parsed as valid shell
}

// Parse parses a shell command string and returns structured information
// about all the commands it contains. On parse failure it returns ParseError=true.
func Parse(input string) ParseResult {
	r := strings.NewReader(input)
	f, err := syntax.NewParser().Parse(r, "")
	if err != nil {
		return ParseResult{ParseError: true}
	}

	var result ParseResult

	syntax.Walk(f, func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.FuncDecl:
			if x.Name.Value == ":" {
				result.HasForkBomb = true
			}
			// Do not descend into the function body: commands inside a
			// function definition are only executed when the function is
			// called, not when the line is evaluated. Appending them to
			// result.Commands would cause false positives (e.g. the body
			// of a fork bomb definition being flagged as a top-level cmd).
			return false
		case *syntax.CmdSubst:
			result.HasSubstitution = true
		case *syntax.ProcSubst:
			result.HasSubstitution = true
		case *syntax.Stmt:
			for _, redir := range x.Redirs {
				if r := extractRedirect(redir); r != nil {
					result.Redirects = append(result.Redirects, *r)
				}
			}
		case *syntax.CallExpr:
			cmd, hasSub := extractCommand(x)
			if hasSub {
				result.HasSubstitution = true
			}
			if cmd.Name != "" {
				result.Commands = append(result.Commands, cmd)
			}
		}
		return true
	})

	return result
}

func extractCommand(call *syntax.CallExpr) (Command, bool) {
	var cmd Command
	var hasSub bool

	for i, word := range call.Args {
		if wordHasCmdSubst(word) {
			hasSub = true
			if i > 0 {
				cmd.HasNonLiteralArg = true
			}
		}
		lit := wordLit(word)
		switch {
		case i == 0:
			cmd.Name = lit
		case lit != "":
			cmd.Args = append(cmd.Args, lit)
		default:
			// Non-literal arg (variable expansion, arithmetic, etc.)
			if i > 0 {
				cmd.HasNonLiteralArg = true
			}
		}
	}

	return cmd, hasSub
}

func extractRedirect(redir *syntax.Redirect) *Redirect {
	switch redir.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrAll, syntax.AppAll:
		return &Redirect{Target: wordLit(redir.Word)}
	}
	return nil
}

// wordHasCmdSubst reports whether a word contains a command substitution ($() or ``)
// or a process substitution (<(...) or >(...)).
func wordHasCmdSubst(word *syntax.Word) bool {
	for _, part := range word.Parts {
		switch x := part.(type) {
		case *syntax.CmdSubst:
			return true
		case *syntax.ProcSubst:
			return true
		case *syntax.DblQuoted:
			for _, dp := range x.Parts {
				if _, ok := dp.(*syntax.CmdSubst); ok {
					return true
				}
			}
		}
	}
	return false
}

// wordLit returns the literal string value of a word when it can be fully resolved
// without shell evaluation. Returns empty string if the word contains any expansion.
func wordLit(word *syntax.Word) string {
	var b strings.Builder
	for _, part := range word.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			b.WriteString(x.Value)
		case *syntax.SglQuoted:
			b.WriteString(x.Value)
		case *syntax.DblQuoted:
			inner := dblQuotedLit(x)
			if inner == "" && len(x.Parts) > 0 {
				return "" // contains expansions inside double quotes
			}
			b.WriteString(inner)
		default:
			return ""
		}
	}
	return b.String()
}

func dblQuotedLit(dq *syntax.DblQuoted) string {
	var b strings.Builder
	for _, part := range dq.Parts {
		switch x := part.(type) {
		case *syntax.Lit:
			b.WriteString(x.Value)
		default:
			return ""
		}
	}
	return b.String()
}
