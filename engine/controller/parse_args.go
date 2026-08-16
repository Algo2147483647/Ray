package controller

import (
	"flag"
	"fmt"
	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"io"
)

func (h *Handler) ParseArgs(args []string) *Handler {
	if h.err != nil {
		return h
	}

	scriptPaths := stringListFlag{}

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(&scriptPaths, "script", "path to a canonical scene script")

	if err := flagSet.Parse(args); err != nil {
		h.err = err
		return h
	}

	scriptPaths = append(scriptPaths, flagSet.Args()...)
	if len(scriptPaths) == 0 {
		scriptPaths = append(scriptPaths, defaultScriptPath)
	}
	if len(scriptPaths) != 1 {
		h.err = fmt.Errorf("engine accepts exactly one --script; use studio to merge multiple scripts")
		return h
	}
	h.ScriptPath = scriptPaths[0]
	return h
}

func (h *Handler) LoadScript() *Handler {
	if h.err != nil {
		return h
	}

	fmt.Printf("Loading scene from: %s\n", h.ScriptPath)

	script, err := parser.ReadScriptFile(h.ScriptPath)
	if err != nil {
		h.err = err
		return h
	}

	h.Script = script
	if err := factory.LoadSceneFromScript(script, h.Scene); err != nil {
		h.err = err
		return h
	}

	return h
}

type stringListFlag []string

func (s *stringListFlag) String() string {
	return fmt.Sprint([]string(*s))
}

func (s *stringListFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
