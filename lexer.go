// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Lexer struct {
	src  []rune
	pos  int
	line int
	col  int
}

func NewLexer(src string) *Lexer {
	return &Lexer{src: []rune(src), line: 1, col: 1}
}

func (l *Lexer) Lex() ([]Token, error) {
	var tokens []Token
	for {
		ch := l.peek()
		startLine, startCol := l.line, l.col
		switch {
		case ch == 0:
			tokens = append(tokens, Token{Kind: TokenEOF, Line: startLine, Col: startCol})
			return tokens, nil
		case ch == ' ' || ch == '\t' || ch == '\r':
			l.advance()
		case ch == '/' && l.peekNext() == '*':
			if err := l.skipBlockComment(); err != nil {
				return nil, err
			}
		case ch == '\n':
			l.advance()
			tokens = append(tokens, Token{Kind: TokenNewline, Lexeme: "\n", Line: startLine, Col: startCol})
		case unicode.IsLetter(ch) || ch == '_':
			lexeme := l.readWhile(func(r rune) bool {
				return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
			})
			tokens = append(tokens, Token{Kind: keywordKind(lexeme), Lexeme: lexeme, Line: startLine, Col: startCol})
		case unicode.IsDigit(ch):
			lexeme, isDouble := l.readNumber()
			kind := TokenInt
			if isDouble {
				kind = TokenDouble
			}
			tokens = append(tokens, Token{Kind: kind, Lexeme: lexeme, Line: startLine, Col: startCol})
		case ch == '"':
			lexeme, err := l.readString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{Kind: TokenString, Lexeme: lexeme, Line: startLine, Col: startCol})
		default:
			kind, lexeme, ok := l.operatorToken()
			if !ok {
				return nil, fmt.Errorf("line %d:%d: unexpected character %q", startLine, startCol, ch)
			}
			tokens = append(tokens, Token{Kind: kind, Lexeme: lexeme, Line: startLine, Col: startCol})
		}
	}
}

func (l *Lexer) operatorToken() (TokenKind, string, bool) {
	ch := l.peek()
	next := l.peekNext()
	switch ch {
	case '+':
		l.advance()
		return TokenPlus, "+", true
	case '-':
		l.advance()
		return TokenMinus, "-", true
	case '*':
		l.advance()
		return TokenStar, "*", true
	case '/':
		l.advance()
		return TokenSlash, "/", true
	case '!':
		l.advance()
		if next == '=' {
			l.advance()
			return TokenBangEqual, "!=", true
		}
		return TokenEOF, "", false
	case '%':
		l.advance()
		return TokenPercent, "%", true
	case '^':
		l.advance()
		return TokenCaret, "^", true
	case '(':
		l.advance()
		return TokenLParen, "(", true
	case ')':
		l.advance()
		return TokenRParen, ")", true
	case '{':
		l.advance()
		return TokenLBrace, "{", true
	case '}':
		l.advance()
		return TokenRBrace, "}", true
	case ':':
		l.advance()
		return TokenColon, ":", true
	case '=':
		l.advance()
		if next == '=' {
			l.advance()
			return TokenEqualEqual, "==", true
		}
		return TokenEqual, "=", true
	case '<':
		l.advance()
		if next == '=' {
			l.advance()
			return TokenLessEqual, "<=", true
		}
		return TokenLess, "<", true
	case '>':
		l.advance()
		if next == '=' {
			l.advance()
			return TokenGreaterEqual, ">=", true
		}
		return TokenGreater, ">", true
	default:
		return TokenEOF, "", false
	}
}

func (l *Lexer) skipBlockComment() error {
	startLine, startCol := l.line, l.col
	l.advance()
	l.advance()
	for {
		if l.peek() == 0 {
			return fmt.Errorf("line %d:%d: unterminated block comment", startLine, startCol)
		}
		if l.peek() == '*' && l.peekNext() == '/' {
			l.advance()
			l.advance()
			return nil
		}
		l.advance()
	}
}

func (l *Lexer) readNumber() (string, bool) {
	var b strings.Builder
	isDouble := false
	for unicode.IsDigit(l.peek()) {
		b.WriteRune(l.advance())
	}
	if l.peek() == '.' && unicode.IsDigit(l.peekNext()) {
		isDouble = true
		b.WriteRune(l.advance())
		for unicode.IsDigit(l.peek()) {
			b.WriteRune(l.advance())
		}
	}
	return b.String(), isDouble
}

func (l *Lexer) readString() (string, error) {
	startLine, startCol := l.line, l.col
	l.advance()
	var b strings.Builder
	for {
		ch := l.peek()
		if ch == 0 || ch == '\n' {
			return "", fmt.Errorf("line %d:%d: unterminated string literal", startLine, startCol)
		}
		if ch == '"' {
			l.advance()
			return b.String(), nil
		}
		if ch == '\\' {
			l.advance()
			switch l.peek() {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				return "", fmt.Errorf("line %d:%d: unknown string escape", l.line, l.col)
			}
			l.advance()
			continue
		}
		b.WriteRune(l.advance())
	}
}

func (l *Lexer) readWhile(pred func(rune) bool) string {
	var b strings.Builder
	for pred(l.peek()) {
		b.WriteRune(l.advance())
	}
	return b.String()
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekNext() rune {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}

func (l *Lexer) advance() rune {
	ch := l.peek()
	if ch == 0 {
		return 0
	}
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}
