all: parser/parser.go parser/lex.nn.go

parser/parser.go: parser/parser.go.y
	go tool yacc -o $@ -v parser/y.output $<

parser/lex.nn.go: parser/lex.nex
	nex -o $@ $<

clean:
	rm -f parser/parser.go
	rm -f parser/lex.nn.go
	rm -f parser/y.output

.PHONY: clean
