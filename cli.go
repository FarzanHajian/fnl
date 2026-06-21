// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type Options struct {
	Source    string
	Output    string
	EmitC     bool
	EmitLLVM  bool
	EmitAST   bool
	EmitGraph bool
	Backend   string
}

func ParseArgs(args []string) (Options, error) {
	opts := Options{
		Backend: "gcc",
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o":
			i++
			if i >= len(args) {
				return opts, errors.New("-o requires an output path")
			}
			opts.Output = args[i]
		case arg == "--emit-c":
			opts.EmitC = true
		case arg == "--emit-llvm":
			opts.EmitLLVM = true
		case arg == "--emit-ast":
			opts.EmitAST = true
		case arg == "--emit-ast-graph":
			opts.EmitGraph = true
		case strings.HasPrefix(arg, "--backend="):
			opts.Backend = strings.TrimPrefix(arg, "--backend=")
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown option %s", arg)
		default:
			if opts.Source != "" {
				return opts, errors.New("expected exactly one source file")
			}
			opts.Source = arg
		}
	}
	if opts.Source == "" {
		return opts, errors.New("missing source file")
	}
	if !isSourceInput(opts.Source) {
		return opts, errors.New("source file must use .fnl or .fnl.ast extension")
	}
	if opts.Backend != "gcc" && opts.Backend != "clang" {
		return opts, errors.New("--backend must be gcc or clang")
	}
	if opts.Output == "" {
		base := inputBaseName(opts.Source)
		opts.Output = base + hostExeExt()
	}
	return opts, nil
}

func usage() string {
	return "usage: fnlc.exe <source.fnl|source.fnl.ast> [-o source.exe] [--emit-c] [--emit-llvm] [--emit-ast] [--emit-ast-graph] [--backend=gcc|clang]"
}

func isSourceInput(path string) bool {
	return filepath.Ext(path) == ".fnl" || strings.HasSuffix(path, ".fnl.ast")
}

func isASTInput(path string) bool {
	return strings.HasSuffix(path, ".fnl.ast")
}

func inputBaseName(path string) string {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".fnl.ast") {
		return strings.TrimSuffix(base, ".fnl.ast")
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
