// Copyright (c) 2026 Farzan Hajian
// SPDX-License-Identifier: BSD-3-Clause

package main

import "fmt"

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

func ParseSource(src string) (*Program, error) {
	tokens, err := NewLexer(src).Lex()
	if err != nil {
		return nil, err
	}
	return NewParser(tokens).Parse()
}

func ParseAndCheckSource(src string) (*Program, error) {
	prog, err := ParseSource(src)
	if err != nil {
		return nil, err
	}
	if err := NewChecker().Check(prog); err != nil {
		return nil, err
	}
	return prog, nil
}

func (p *Parser) Parse() (*Program, error) {
	p.skipNewlines()
	prog := &Program{}
	for !p.check(TokenEOF) {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		prog.Statements = append(prog.Statements, stmt)
		if !p.check(TokenEOF) {
			if _, err := p.consume(TokenNewline, "expected newline after statement"); err != nil {
				return nil, err
			}
			p.skipNewlines()
		}
	}
	return prog, nil
}

func (p *Parser) parseStatement() (Stmt, error) {
	switch {
	case p.match(TokenVar):
		return p.parseVarDecl()
	case p.match(TokenPrint):
		return p.parsePrint(false)
	case p.match(TokenPrintln):
		return p.parsePrint(true)
	case p.match(TokenExit):
		return p.parseExit()
	case p.match(TokenBreak):
		return &BreakStmt{Pos: tokenPos(p.previous())}, nil
	case p.match(TokenIf):
		return p.parseIf()
	case p.match(TokenWhile):
		return p.parseWhile()
	case p.check(TokenIdent):
		return p.parseAssign()
	default:
		tok := p.peek()
		return nil, fmt.Errorf("line %d:%d: expected statement", tok.Line, tok.Col)
	}
}

func (p *Parser) parseVarDecl() (Stmt, error) {
	pos := tokenPos(p.previous())
	name, err := p.consume(TokenIdent, "expected variable name after var")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenColon, "expected ':' after variable name"); err != nil {
		return nil, err
	}
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenEqual, "expected '=' after variable type"); err != nil {
		return nil, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &VarDecl{Pos: pos, Name: name.Lexeme, Type: typ, Value: value}, nil
}

func (p *Parser) parseAssign() (Stmt, error) {
	name := p.advance()
	pos := tokenPos(name)
	if _, err := p.consume(TokenEqual, "expected '=' after variable name"); err != nil {
		return nil, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	return &Assign{Pos: pos, Name: name.Lexeme, Value: value}, nil
}

func (p *Parser) parsePrint(newline bool) (Stmt, error) {
	pos := tokenPos(p.previous())
	if _, err := p.consume(TokenLParen, "expected '(' after print"); err != nil {
		return nil, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenRParen, "expected ')' after print expression"); err != nil {
		return nil, err
	}
	return &PrintStmt{Pos: pos, Value: value, Newline: newline}, nil
}

func (p *Parser) parseExit() (Stmt, error) {
	pos := tokenPos(p.previous())
	code, err := p.parseSingleArgCall("exit")
	if err != nil {
		return nil, err
	}
	return &ExitStmt{Pos: pos, Code: code}, nil
}

func (p *Parser) parseIf() (Stmt, error) {
	pos := tokenPos(p.previous())
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	thenBody, err := p.parseBlock("if")
	if err != nil {
		return nil, err
	}
	var elseIf []IfBranch
	for {
		elseSearchPos := p.pos
		p.skipNewlines()
		if !p.match(TokenElseIf) {
			p.pos = elseSearchPos
			break
		}
		branchPos := tokenPos(p.previous())
		branchCond, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		branchBody, err := p.parseBlock("elseif")
		if err != nil {
			return nil, err
		}
		elseIf = append(elseIf, IfBranch{Pos: branchPos, Cond: branchCond, Body: branchBody})
	}
	elseSearchPos := p.pos
	p.skipNewlines()
	var elseBody []Stmt
	if p.match(TokenElse) {
		elseBody, err = p.parseBlock("else")
		if err != nil {
			return nil, err
		}
	} else {
		p.pos = elseSearchPos
	}
	return &IfStmt{Pos: pos, Cond: cond, Then: thenBody, ElseIf: elseIf, Else: elseBody}, nil
}

func (p *Parser) parseWhile() (Stmt, error) {
	pos := tokenPos(p.previous())
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	body, err := p.parseBlock("while")
	if err != nil {
		return nil, err
	}
	return &WhileStmt{Pos: pos, Cond: cond, Body: body}, nil
}

func (p *Parser) parseBlock(name string) ([]Stmt, error) {
	if _, err := p.consume(TokenLBrace, "expected '{' to start "+name+" block"); err != nil {
		return nil, err
	}
	var body []Stmt
	p.skipNewlines()
	for !p.check(TokenRBrace) && !p.check(TokenEOF) {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		body = append(body, stmt)
		if !p.check(TokenRBrace) && !p.check(TokenEOF) {
			if _, err := p.consume(TokenNewline, "expected newline after statement"); err != nil {
				return nil, err
			}
			p.skipNewlines()
		}
	}
	if len(body) == 0 {
		return nil, p.errorAt(p.peek(), name+" block must contain at least one statement")
	}
	if _, err := p.consume(TokenRBrace, "expected '}' to close "+name+" block"); err != nil {
		return nil, err
	}
	return body, nil
}

func (p *Parser) parseType() (Type, error) {
	tok := p.advance()
	switch tok.Kind {
	case TokenTypeInt:
		return TypeInt, nil
	case TokenTypeDouble:
		return TypeDouble, nil
	case TokenTypeBool:
		return TypeBool, nil
	case TokenTypeString:
		return TypeString, nil
	default:
		return "", p.errorAt(tok, "expected type int, double, bool, or string")
	}
}

func (p *Parser) parseExpression() (Expr, error) {
	return p.parseComparison()
}

func (p *Parser) parseComparison() (Expr, error) {
	expr, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if p.match(TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual, TokenEqualEqual, TokenBangEqual) {
		opToken := p.previous()
		op := opToken.Kind
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Pos: tokenPos(opToken), Left: expr, Op: op, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseAdditive() (Expr, error) {
	expr, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.match(TokenPlus, TokenMinus) {
		opToken := p.previous()
		op := opToken.Kind
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Pos: tokenPos(opToken), Left: expr, Op: op, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseMultiplicative() (Expr, error) {
	expr, err := p.parsePower()
	if err != nil {
		return nil, err
	}
	for p.match(TokenStar, TokenSlash, TokenPercent) {
		opToken := p.previous()
		op := opToken.Kind
		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Pos: tokenPos(opToken), Left: expr, Op: op, Right: right}
	}
	return expr, nil
}

func (p *Parser) parsePower() (Expr, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.match(TokenCaret) {
		opToken := p.previous()
		right, err := p.parsePower()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Pos: tokenPos(opToken), Left: expr, Op: TokenCaret, Right: right}
	}
	return expr, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	if p.match(TokenMinus) {
		opToken := p.previous()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Pos: tokenPos(opToken), Op: TokenMinus, Value: value}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	switch {
	case p.match(TokenInt):
		return &LiteralExpr{Pos: tokenPos(p.previous()), Value: p.previous().Lexeme, Type: TypeInt}, nil
	case p.match(TokenDouble):
		return &LiteralExpr{Pos: tokenPos(p.previous()), Value: p.previous().Lexeme, Type: TypeDouble}, nil
	case p.match(TokenString):
		return &LiteralExpr{Pos: tokenPos(p.previous()), Value: p.previous().Lexeme, Type: TypeString}, nil
	case p.match(TokenTrue):
		return &LiteralExpr{Pos: tokenPos(p.previous()), Value: "true", Type: TypeBool}, nil
	case p.match(TokenFalse):
		return &LiteralExpr{Pos: tokenPos(p.previous()), Value: "false", Type: TypeBool}, nil
	case p.match(TokenIdent):
		return &VarExpr{Pos: tokenPos(p.previous()), Name: p.previous().Lexeme}, nil
	case p.match(TokenStr):
		pos := tokenPos(p.previous())
		if _, err := p.consume(TokenLParen, "expected '(' after string conversion function"); err != nil {
			return nil, err
		}
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')' after to_str expression"); err != nil {
			return nil, err
		}
		return &StrCallExpr{Pos: pos, Value: value}, nil
	case p.match(TokenIsInt):
		pos := tokenPos(p.previous())
		value, err := p.parseSingleArgCall("is_int")
		if err != nil {
			return nil, err
		}
		return &IsIntCallExpr{Pos: pos, Value: value}, nil
	case p.match(TokenToInt):
		pos := tokenPos(p.previous())
		value, err := p.parseSingleArgCall("to_int")
		if err != nil {
			return nil, err
		}
		return &ToIntCallExpr{Pos: pos, Value: value}, nil
	case p.match(TokenIsDouble):
		pos := tokenPos(p.previous())
		value, err := p.parseSingleArgCall("is_double")
		if err != nil {
			return nil, err
		}
		return &IsDoubleCallExpr{Pos: pos, Value: value}, nil
	case p.match(TokenToDouble):
		pos := tokenPos(p.previous())
		value, err := p.parseSingleArgCall("to_double")
		if err != nil {
			return nil, err
		}
		return &ToDoubleCallExpr{Pos: pos, Value: value}, nil
	case p.match(TokenMathRandom):
		pos := tokenPos(p.previous())
		if _, err := p.consume(TokenLParen, "expected '(' after math_random"); err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')' after math_random"); err != nil {
			return nil, err
		}
		return &MathRandomCallExpr{Pos: pos}, nil
	case p.match(TokenInput):
		pos := tokenPos(p.previous())
		if _, err := p.consume(TokenLParen, "expected '(' after input"); err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')' after input"); err != nil {
			return nil, err
		}
		return &InputCallExpr{Pos: pos}, nil
	case p.match(TokenLParen):
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenRParen, "expected ')' after expression"); err != nil {
			return nil, err
		}
		return expr, nil
	default:
		return nil, p.errorAt(p.peek(), "expected expression")
	}
}

func (p *Parser) parseSingleArgCall(name string) (Expr, error) {
	if _, err := p.consume(TokenLParen, "expected '(' after "+name); err != nil {
		return nil, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenRParen, "expected ')' after "+name+" expression"); err != nil {
		return nil, err
	}
	return value, nil
}

func (p *Parser) skipNewlines() {
	for p.match(TokenNewline) {
	}
}

func (p *Parser) match(kinds ...TokenKind) bool {
	for _, kind := range kinds {
		if p.check(kind) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) consume(kind TokenKind, msg string) (Token, error) {
	if p.check(kind) {
		return p.advance(), nil
	}
	return Token{}, p.errorAt(p.peek(), msg)
}

func (p *Parser) check(kind TokenKind) bool {
	return p.peek().Kind == kind
}

func (p *Parser) advance() Token {
	if !p.check(TokenEOF) {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *Parser) previous() Token {
	return p.tokens[p.pos-1]
}

func (p *Parser) errorAt(tok Token, msg string) error {
	return fmt.Errorf("line %d:%d: %s", tok.Line, tok.Col, msg)
}

func tokenPos(tok Token) SourcePos {
	return SourcePos{Line: tok.Line, Col: tok.Col}
}
