package main

import (
	"attribute_parser"
	"lexer"
)

func main() {
	exprLexer := lexer.NewLexer("1+2*(4+3);")
	attributeParser := attribute_parser.NewAttributeParser(exprLexer)
	attributeParser.Parse()
}
