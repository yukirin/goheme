package main

import (
	"fmt"
	"os"
)
import (
	"bufio"
	"io"
	"strings"
)

type frame struct {
	i            int
	s            string
	line, column int
}
type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch chan frame
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []frame
	stale bool

	// The 'l' and 'c' fields were added for
	// https://github.com/wagerlabs/docker/blob/65694e801a7b80930961d70c69cba9f2465459be/buildfile.nex
	// Since then, I introduced the built-in Line() and Column() functions.
	l, c int

	parseResult interface{}

	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	type dfa struct {
		acc          []bool           // Accepting states.
		f            []func(rune) int // Transitions.
		startf, endf []int            // Transitions at start and end of input.
		nest         []dfa
	}
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan frame)
	var scan func(in *bufio.Reader, ch chan frame, family []dfa, line, column int)
	scan = func(in *bufio.Reader, ch chan frame, family []dfa, line, column int) {
		// Index of DFA and length of highest-precedence match so far.
		matchi, matchn := 0, -1
		var buf []rune
		n := 0
		checkAccept := func(i int, st int) bool {
			// Higher precedence match? DFAs are run in parallel, so matchn is at most len(buf), hence we may omit the length equality check.
			if family[i].acc[st] && (matchn < n || matchi > i) {
				matchi, matchn = i, n
				return true
			}
			return false
		}
		var state [][2]int
		for i := 0; i < len(family); i++ {
			mark := make([]bool, len(family[i].startf))
			// Every DFA starts at state 0.
			st := 0
			for {
				state = append(state, [2]int{i, st})
				mark[st] = true
				// As we're at the start of input, follow all ^ transitions and append to our list of start states.
				st = family[i].startf[st]
				if -1 == st || mark[st] {
					break
				}
				// We only check for a match after at least one transition.
				checkAccept(i, st)
			}
		}
		atEOF := false
		for {
			if n == len(buf) && !atEOF {
				r, _, err := in.ReadRune()
				switch err {
				case io.EOF:
					atEOF = true
				case nil:
					buf = append(buf, r)
				default:
					panic(err)
				}
			}
			if !atEOF {
				r := buf[n]
				n++
				var nextState [][2]int
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if -1 == x[1] {
						continue
					}
					nextState = append(nextState, x)
					checkAccept(x[0], x[1])
				}
				state = nextState
			} else {
			dollar: // Handle $.
				for _, x := range state {
					mark := make([]bool, len(family[x[0]].endf))
					for {
						mark[x[1]] = true
						x[1] = family[x[0]].endf[x[1]]
						if -1 == x[1] || mark[x[1]] {
							break
						}
						if checkAccept(x[0], x[1]) {
							// Unlike before, we can break off the search. Now that we're at the end, there's no need to maintain the state of each DFA.
							break dollar
						}
					}
				}
				state = nil
			}

			if state == nil {
				lcUpdate := func(r rune) {
					if r == '\n' {
						line++
						column = 0
					} else {
						column++
					}
				}
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					lcUpdate(buf[0])
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1
					ch <- frame{matchi, text, line, column}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, family[matchi].nest, line, column)
					}
					if atEOF {
						break
					}
					for _, r := range text {
						lcUpdate(r)
					}
				}
				n = 0
				for i := 0; i < len(family); i++ {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		ch <- frame{-1, "", line, column}
	}
	go scan(bufio.NewReader(in), yylex.ch, []dfa{
		// quote|lambda|if|set!|begin|cond|and|or|case|let|let\*|letrec|do|delay|quasiquote|else|=>|define|unquote|unquote-splicing
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, true, false, false, false, false, false, false, true, true, false, false, true, true, false, false, true, false, false, false, true, true, false, false, true, false, true, false, false, false, true, false, false, true, false, false, false, true, false, true, false, false, false, true, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return 1
				case 62:
					return -1
				case 97:
					return 2
				case 98:
					return 3
				case 99:
					return 4
				case 100:
					return 5
				case 101:
					return 6
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 7
				case 108:
					return 8
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 9
				case 112:
					return -1
				case 113:
					return 10
				case 114:
					return -1
				case 115:
					return 11
				case 116:
					return -1
				case 117:
					return 12
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return 80
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 78
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 74
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return 68
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 69
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 59
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 60
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return 56
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return 55
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return 44
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 45
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return 43
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 31
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 28
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 13
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return 14
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 15
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 16
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 17
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 18
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return 19
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 20
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return 21
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return 22
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 23
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return 24
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 25
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 26
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return 27
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 29
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return 30
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return 32
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 33
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 36
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 34
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 35
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 37
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return 38
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 39
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 40
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 41
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 42
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return 51
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 46
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return 47
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return 48
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 49
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return 50
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return 52
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 53
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return 54
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 57
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 58
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return 61
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return 62
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 65
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return 63
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return 64
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 66
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 67
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 72
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 70
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 71
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 73
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return 75
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 76
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return 77
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 79
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				case 121:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// [a-zA-Z!\$%&\*\/:\<=\>\?\^_~][a-zA-Z0-9!\$%&\*\/:\<=\>\?\^_~\+\-\.@]*|\+|-|\.\.\.
		{[]bool{false, true, true, true, false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 33:
					return 1
				case 36:
					return 1
				case 37:
					return 1
				case 38:
					return 1
				case 42:
					return 1
				case 43:
					return 2
				case 45:
					return 3
				case 46:
					return 4
				case 47:
					return 1
				case 58:
					return 1
				case 60:
					return 1
				case 61:
					return 1
				case 62:
					return 1
				case 63:
					return 1
				case 64:
					return -1
				case 94:
					return 1
				case 95:
					return 1
				case 126:
					return 1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return 1
				case 97 <= r && r <= 122:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return 7
				case 36:
					return 7
				case 37:
					return 7
				case 38:
					return 7
				case 42:
					return 7
				case 43:
					return 7
				case 45:
					return 7
				case 46:
					return 7
				case 47:
					return 7
				case 58:
					return 7
				case 60:
					return 7
				case 61:
					return 7
				case 62:
					return 7
				case 63:
					return 7
				case 64:
					return 7
				case 94:
					return 7
				case 95:
					return 7
				case 126:
					return 7
				}
				switch {
				case 43 <= r && r <= 46:
					return 7
				case 48 <= r && r <= 57:
					return 7
				case 65 <= r && r <= 90:
					return 7
				case 97 <= r && r <= 122:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 36:
					return -1
				case 37:
					return -1
				case 38:
					return -1
				case 42:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 58:
					return -1
				case 60:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 63:
					return -1
				case 64:
					return -1
				case 94:
					return -1
				case 95:
					return -1
				case 126:
					return -1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return -1
				case 97 <= r && r <= 122:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 36:
					return -1
				case 37:
					return -1
				case 38:
					return -1
				case 42:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 58:
					return -1
				case 60:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 63:
					return -1
				case 64:
					return -1
				case 94:
					return -1
				case 95:
					return -1
				case 126:
					return -1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return -1
				case 97 <= r && r <= 122:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 36:
					return -1
				case 37:
					return -1
				case 38:
					return -1
				case 42:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 5
				case 47:
					return -1
				case 58:
					return -1
				case 60:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 63:
					return -1
				case 64:
					return -1
				case 94:
					return -1
				case 95:
					return -1
				case 126:
					return -1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return -1
				case 97 <= r && r <= 122:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 36:
					return -1
				case 37:
					return -1
				case 38:
					return -1
				case 42:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 6
				case 47:
					return -1
				case 58:
					return -1
				case 60:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 63:
					return -1
				case 64:
					return -1
				case 94:
					return -1
				case 95:
					return -1
				case 126:
					return -1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return -1
				case 97 <= r && r <= 122:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 36:
					return -1
				case 37:
					return -1
				case 38:
					return -1
				case 42:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 58:
					return -1
				case 60:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 63:
					return -1
				case 64:
					return -1
				case 94:
					return -1
				case 95:
					return -1
				case 126:
					return -1
				}
				switch {
				case 43 <= r && r <= 46:
					return -1
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 90:
					return -1
				case 97 <= r && r <= 122:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return 7
				case 36:
					return 7
				case 37:
					return 7
				case 38:
					return 7
				case 42:
					return 7
				case 43:
					return 7
				case 45:
					return 7
				case 46:
					return 7
				case 47:
					return 7
				case 58:
					return 7
				case 60:
					return 7
				case 61:
					return 7
				case 62:
					return 7
				case 63:
					return 7
				case 64:
					return 7
				case 94:
					return 7
				case 95:
					return 7
				case 126:
					return 7
				}
				switch {
				case 43 <= r && r <= 46:
					return 7
				case 48 <= r && r <= 57:
					return 7
				case 65 <= r && r <= 90:
					return 7
				case 97 <= r && r <= 122:
					return 7
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// #t|#f
		{[]bool{false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 102:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 102:
					return 2
				case 116:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 102:
					return -1
				case 116:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// #\\.
		{[]bool{false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 92:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 3
				case 92:
					return 3
				}
				return 3
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1}, nil},

		// #\\space
		{[]bool{false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return 2
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return 4
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return 5
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return 6
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return 7
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// #\\newline
		{[]bool{false, false, false, false, false, false, false, false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return 2
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return 3
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return 4
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return 6
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return 7
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return 8
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 92:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 119:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// "([^"\\]|\\\"|\\\\)*"
		{[]bool{false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 34:
					return 1
				case 92:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 2
				case 92:
					return 3
				}
				return 4
			},
			func(r rune) int {
				switch r {
				case 34:
					return -1
				case 92:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 5
				case 92:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 34:
					return 2
				case 92:
					return 3
				}
				return 4
			},
			func(r rune) int {
				switch r {
				case 34:
					return 2
				case 92:
					return 3
				}
				return 4
			},
			func(r rune) int {
				switch r {
				case 34:
					return 2
				case 92:
					return 3
				}
				return 4
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

		// [ \n\r\t]*
		{[]bool{true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 9:
					return 0
				case 10:
					return 0
				case 13:
					return 0
				case 32:
					return 0
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1}, []int{ /* End-of-input transitions */ -1}, nil},

		// ;[^\n\r]*
		{[]bool{false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 10:
					return -1
				case 13:
					return -1
				case 59:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 10:
					return -1
				case 13:
					return -1
				case 59:
					return 2
				}
				return 2
			},
			func(r rune) int {
				switch r {
				case 10:
					return -1
				case 13:
					return -1
				case 59:
					return 2
				}
				return 2
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// [01]+
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 48:
					return 1
				case 49:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 48:
					return 1
				case 49:
					return 1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

		// [0-7]+
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch {
				case 48 <= r && r <= 55:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch {
				case 48 <= r && r <= 55:
					return 1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

		// [0-9]+
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch {
				case 48 <= r && r <= 57:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch {
				case 48 <= r && r <= 57:
					return 1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

		// [0-9a-fA-F]+
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch {
				case 48 <= r && r <= 57:
					return 1
				case 65 <= r && r <= 70:
					return 1
				case 97 <= r && r <= 102:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch {
				case 48 <= r && r <= 57:
					return 1
				case 65 <= r && r <= 70:
					return 1
				case 97 <= r && r <= 102:
					return 1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

		// [\+-]i
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 43:
					return 1
				case 45:
					return 1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 43:
					return -1
				case 45:
					return -1
				case 105:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 43:
					return -1
				case 45:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// #[iIEe]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 69:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 101:
					return 2
				case 105:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// #[bBoOxXdD]
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 66:
					return -1
				case 68:
					return -1
				case 79:
					return -1
				case 88:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 111:
					return -1
				case 120:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 66:
					return 2
				case 68:
					return 2
				case 79:
					return 2
				case 88:
					return 2
				case 98:
					return 2
				case 100:
					return 2
				case 111:
					return 2
				case 120:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 66:
					return -1
				case 68:
					return -1
				case 79:
					return -1
				case 88:
					return -1
				case 98:
					return -1
				case 100:
					return -1
				case 111:
					return -1
				case 120:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// \(|\)|#\(|'|`|,|,@|\.
		{[]bool{false, false, true, true, true, true, true, true, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 39:
					return 2
				case 40:
					return 3
				case 41:
					return 4
				case 44:
					return 5
				case 46:
					return 6
				case 64:
					return -1
				case 96:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return 9
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return 8
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 39:
					return -1
				case 40:
					return -1
				case 41:
					return -1
				case 44:
					return -1
				case 46:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},
	}, 0, 0)
	return yylex
}

func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}

// Text returns the matched text.
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}

// Line returns the current line number.
// The first line is 0.
func (yylex *Lexer) Line() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].line
}

// Column returns the current column number.
// The first column is 0.
func (yylex *Lexer) Column() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].column
}

func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		l, c := 0, 0
		if lvl > 0 {
			l, c = yylex.stack[lvl-1].line, yylex.stack[lvl-1].column
		}
		yylex.stack = append(yylex.stack, frame{0, "", l, c})
	}
	if lvl == len(yylex.stack)-1 {
		p := &yylex.stack[lvl]
		*p = <-yylex.ch
		yylex.stale = false
	} else {
		yylex.stale = true
	}
	return yylex.stack[lvl].i
}
func (yylex *Lexer) pop() {
	yylex.stack = yylex.stack[:len(yylex.stack)-1]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}

// Lex runs the lexer. Always returns 0.
// When the -s option is given, this function is not generated;
// instead, the NN_FUN macro runs the lexer.
func (yylex *Lexer) Lex(lval *yySymType) int {
OUTER0:
	for {
		switch yylex.next(0) {
		case 0:
			{
				fmt.Println("keyword = ", yylex.Text())
			}
		case 1:
			{
				fmt.Println("identifier = ", yylex.Text())
			}
		case 2:
			{
				fmt.Println("bool = ", yylex.Text())
			}
		case 3:
			{
				fmt.Println("char = ", yylex.Text())
			}
		case 4:
			{
				fmt.Println("space = ", yylex.Text())
			}
		case 5:
			{
				fmt.Println("newline = ", yylex.Text())
			}
		case 6:
			{
				fmt.Println("string = ", yylex.Text())
			}
		case 7:
			{
				fmt.Println("atmosphere = ", yylex.Text())
			}
		case 8:
			{
				fmt.Println("comment = ", yylex.Text())
			}
		case 9:
			{
				fmt.Println("bin number = ", yylex.Text())
			}
		case 10:
			{
				fmt.Println("octet number = ", yylex.Text())
			}
		case 11:
			{
				fmt.Println("decmel number = ", yylex.Text())
			}
		case 12:
			{
				fmt.Println("hex number = ", yylex.Text())
			}
		case 13:
			{
				fmt.Println("sign complex = ", yylex.Text())
			}
		case 14:
			{
				fmt.Println("exact = ", yylex.Text())
			}
		case 15:
			{
				fmt.Println("radix = ", yylex.Text())
			}
		case 16:
			{
				fmt.Println("delimiter = ", yylex.Text())
			}
		default:
			break OUTER0
		}
		continue
	}
	yylex.pop()

	return 0
}

type yySymType struct {
}

func main() {
	v := &yySymType{}
	NewLexer(os.Stdin).Lex(v)
}
