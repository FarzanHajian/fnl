// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenNewline
	TokenIdent
	TokenInt
	TokenDouble
	TokenString
	TokenVar
	TokenIf
	TokenElseIf
	TokenElse
	TokenWhile
	TokenBreak
	TokenPrint
	TokenPrintln
	TokenExit
	TokenStr
	TokenInput
	TokenIsInt64
	TokenToInt64
	TokenIsDouble
	TokenToDouble
	TokenTrue
	TokenFalse
	TokenTypeInt64
	TokenTypeDouble
	TokenTypeBool
	TokenTypeString
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenCaret
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenColon
	TokenEqual
	TokenEqualEqual
	TokenBangEqual
	TokenLess
	TokenLessEqual
	TokenGreater
	TokenGreaterEqual
)

type Token struct {
	Kind   TokenKind
	Lexeme string
	Line   int
	Col    int
}

func keywordKind(s string) TokenKind {
	switch s {
	case "var":
		return TokenVar
	case "if":
		return TokenIf
	case "elseif":
		return TokenElseIf
	case "else":
		return TokenElse
	case "while":
		return TokenWhile
	case "break":
		return TokenBreak
	case "print":
		return TokenPrint
	case "println", "prinln":
		return TokenPrintln
	case "exit":
		return TokenExit
	case "to_str":
		return TokenStr
	case "input":
		return TokenInput
	case "is_int64":
		return TokenIsInt64
	case "to_int64":
		return TokenToInt64
	case "is_double":
		return TokenIsDouble
	case "to_double":
		return TokenToDouble
	case "true":
		return TokenTrue
	case "false":
		return TokenFalse
	case "int64":
		return TokenTypeInt64
	case "double":
		return TokenTypeDouble
	case "bool":
		return TokenTypeBool
	case "string":
		return TokenTypeString
	default:
		return TokenIdent
	}
}
