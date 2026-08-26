package controller

import (
	"flag"
	"fmt"
	"io"

	"github.com/Algo2147483647/ray/engine/controller/factory"
	"github.com/Algo2147483647/ray/engine/controller/parser"
)

func (h *Handler) ParseArgs(args []string) *Handler {
	if h.err != nil {
		return h
	}

	var scriptPath string
	scriptSet := false

	flagSet := flag.NewFlagSet("ray", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Func("script", "path to a canonical scene script", func(value string) error {
		if scriptSet {
			return fmt.Errorf("--script may be specified only once")
		}
		scriptSet = true
		scriptPath = value
		return nil
	})

	if err := flagSet.Parse(args); err != nil {
		h.err = err
		return h
	}

	if len(flagSet.Args()) != 0 {
		h.err = fmt.Errorf("engine does not accept positional arguments; use --script PATH")
		return h
	}
	if !scriptSet || scriptPath == "" {
		h.err = fmt.Errorf("engine requires --script PATH")
		return h
	}
	h.ScriptPath = scriptPath
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
