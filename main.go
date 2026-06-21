// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	opts, err := ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage())
		os.Exit(2)
	}
	if err := Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Run(opts Options) error {
	prog, err := loadProgram(opts.Source)
	if err != nil {
		return err
	}
	if err := NewChecker().Check(prog); err != nil {
		return err
	}
	csrc, err := GenerateC(prog)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(opts.Output)
	if err != nil {
		return err
	}
	outputStem := strings.TrimSuffix(outputAbs, filepath.Ext(outputAbs))

	if opts.EmitAST {
		ast, err := ExportAST(prog)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outputStem+".fnl.ast", ast, 0644); err != nil {
			return err
		}
	}
	if opts.EmitGraph {
		if err := os.WriteFile(outputStem+".fnl.ast.dot", ExportASTGraph(prog), 0644); err != nil {
			return err
		}
	}

	cPath := outputStem + ".c"
	if opts.EmitC {
		if err := os.WriteFile(cPath, []byte(csrc), 0644); err != nil {
			return err
		}
	} else {
		tmp, err := os.CreateTemp("", "fnlc-*.c")
		if err != nil {
			return err
		}
		cPath = tmp.Name()
		if _, err := tmp.WriteString(csrc); err != nil {
			tmp.Close()
			return err
		}
		if err := tmp.Close(); err != nil {
			return err
		}
		defer os.Remove(cPath)
	}

	if opts.EmitLLVM {
		ll, err := GenerateLLVM(prog)
		if err != nil {
			return err
		}
		if err := os.WriteFile(outputStem+".ll", []byte(ll), 0644); err != nil {
			return err
		}
	}

	if err := buildExecutable(opts.Backend, cPath, outputAbs); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", opts.Output)
	if opts.EmitC {
		fmt.Printf("wrote %s\n", outputStem+".c")
	}
	if opts.EmitLLVM {
		fmt.Printf("wrote %s\n", outputStem+".ll")
	}
	if opts.EmitAST {
		fmt.Printf("wrote %s\n", outputStem+".fnl.ast")
	}
	if opts.EmitGraph {
		fmt.Printf("wrote %s\n", outputStem+".fnl.ast.dot")
	}
	return nil
}

func loadProgram(path string) (*Program, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isASTInput(path) {
		return ImportAST(src)
	}
	return ParseSource(string(src))
}

func buildExecutable(backend, cPath, output string) error {
	if err := checkBackendReady(backend); err != nil {
		return err
	}
	cmd := exec.Command(backend, compilerArgs(backend, cPath, output)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s failed: %s", backend, msg)
	}
	return nil
}
