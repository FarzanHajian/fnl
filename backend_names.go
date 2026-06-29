// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import "strings"

func sanitizeBackendName(sourceName string) string {
	var b strings.Builder
	for i, r := range sourceName {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	if b.Len() == 0 {
		return "unnamed"
	}
	return b.String()
}
