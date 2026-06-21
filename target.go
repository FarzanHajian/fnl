// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func hostExeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func hostLLVMTriple() string {
	if runtime.GOOS == "windows" {
		return "x86_64-w64-windows-gnu"
	}
	return "x86_64-pc-linux-gnu"
}

func compilerArgs(backend, cPath, output string) []string {
	args := []string{"-m64", "-O2", "-std=c11", cPath}
	if !(backend == "clang" && runtime.GOOS == "windows") {
		args = append(args, "-lm")
	}
	return append(args, "-o", output)
}

func checkBackendReady(backend string) error {
	if backend != "clang" || runtime.GOOS != "windows" {
		return nil
	}
	if hasMSVCToolchain(os.Getenv, exec.LookPath) {
		return nil
	}
	return fmt.Errorf("clang backend on Windows requires a configured MSVC toolchain; run fnlc from a Visual Studio Developer Command Prompt, or install MSVC Build Tools and ensure VCToolsInstallDir/INCLUDE/LIB are configured")
}

func hasMSVCToolchain(getenv func(string) string, lookPath func(string) (string, error)) bool {
	if getenv("VCToolsInstallDir") != "" || getenv("VCINSTALLDIR") != "" {
		return true
	}
	include := strings.ToLower(getenv("INCLUDE"))
	lib := strings.ToLower(getenv("LIB"))
	if strings.Contains(include, "microsoft visual studio") && strings.Contains(lib, "microsoft visual studio") {
		return true
	}
	if _, err := lookPath("cl"); err != nil {
		return false
	}
	if _, err := lookPath("link"); err != nil {
		return false
	}
	return true
}
