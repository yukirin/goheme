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
		// quote|lambda|if|set!|begin|cond|and|or|case|let|let\*|letrec|do|delay|quasiquote
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, true, false, false, false, false, false, false, true, true, false, false, true, true, false, false, true, false, false, false, true, true, false, true, false, false, true, false, false, false, true, false, true, false, false, false, true, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
					return -1
				case 97:
					return 1
				case 98:
					return 2
				case 99:
					return 3
				case 100:
					return 4
				case 101:
					return -1
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return 5
				case 108:
					return 6
				case 109:
					return -1
				case 110:
					return -1
				case 111:
					return 7
				case 113:
					return 8
				case 114:
					return -1
				case 115:
					return 9
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
					return 53
				case 111:
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
				case 97:
					return 43
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
					return 44
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 38
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
					return 39
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
					return 37
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
				case 97:
					return 26
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 27
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
				case 113:
					return -1
				case 114:
					return 25
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 13
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 10
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 11
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
					return 12
				case 42:
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
				case 97:
					return 14
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
					return 15
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 18
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 16
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 17
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
					return 19
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
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
				case 113:
					return 20
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 21
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
					return 22
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 23
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 24
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
					return 33
				case 110:
					return -1
				case 111:
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return -1
				case 116:
					return 28
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
					return 29
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
				case 113:
					return -1
				case 114:
					return 30
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 31
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return 32
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
				case 97:
					return -1
				case 98:
					return 34
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 35
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
				case 97:
					return 36
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
					return 40
				case 109:
					return -1
				case 110:
					return -1
				case 111:
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
				case 97:
					return 41
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
					return 42
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 33:
					return -1
				case 42:
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
				case 113:
					return -1
				case 114:
					return -1
				case 115:
					return 47
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
					return 45
				case 111:
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 46
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 48
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
					return 50
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
					return 51
				case 108:
					return -1
				case 109:
					return -1
				case 110:
					return -1
				case 111:
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
					return 52
				case 111:
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
				case 97:
					return -1
				case 98:
					return -1
				case 99:
					return -1
				case 100:
					return 54
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
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// else|=>|define|unquote|unquote-splicing
		{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return 1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return 2
				case 101:
					return 3
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return 28
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 23
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
					return 20
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return 5
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return 6
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return 8
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return 9
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 10
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return 11
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return 12
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return 13
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
					return 14
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
					return 15
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return 16
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
					return 17
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return 18
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
					return 19
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return 21
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 22
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return 24
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return 26
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
					return -1
				case 99:
					return -1
				case 100:
					return -1
				case 101:
					return 27
				case 102:
					return -1
				case 103:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 45:
					return -1
				case 61:
					return -1
				case 62:
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
				case 110:
					return -1
				case 111:
					return -1
				case 112:
					return -1
				case 113:
					return -1
				case 115:
					return -1
				case 116:
					return -1
				case 117:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

		// #\\(.|space|newline)
		{[]bool{false, false, false, true, true, true, false, false, false, true, false, false, false, false, false, true}, []func(rune) int{ // Transitions
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
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 119:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 3
				case 92:
					return 3
				case 97:
					return 3
				case 99:
					return 3
				case 101:
					return 3
				case 105:
					return 3
				case 108:
					return 3
				case 110:
					return 4
				case 112:
					return 3
				case 115:
					return 5
				case 119:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return 6
				case 115:
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
				case 97:
					return 7
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return 8
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 119:
					return 11
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
				case 105:
					return -1
				case 108:
					return 12
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return 13
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return 14
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return 15
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
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
				case 97:
					return -1
				case 99:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 110:
					return -1
				case 112:
					return -1
				case 115:
					return -1
				case 119:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

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

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)([\+\-])?([01]+|[01]+\/[01]+)
		{[]bool{false, false, false, false, false, false, false, true, false, true, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 10
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 8
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 9
				case 49:
					return 9
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 9
				case 49:
					return 9
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return 11
				case 73:
					return 11
				case 98:
					return -1
				case 101:
					return 11
				case 105:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)([\+\-])?([01]+|[01]+\/[01]+)@([\+\-])?([01]+|[01]+\/[01]+)
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, true, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 64:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 64:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 8
				case 48:
					return 7
				case 49:
					return 7
				case 64:
					return 9
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return 10
				case 45:
					return 10
				case 47:
					return -1
				case 48:
					return 11
				case 49:
					return 11
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 11
				case 49:
					return 11
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 12
				case 48:
					return 11
				case 49:
					return 11
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 13
				case 49:
					return 13
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 13
				case 49:
					return 13
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 64:
					return 9
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return 16
				case 73:
					return 16
				case 98:
					return -1
				case 101:
					return 16
				case 105:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 64:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)([\+\-])?([01]+|[01]+\/[01]+)\+([01]+|[01]+\/[01]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return 8
				case 45:
					return -1
				case 47:
					return 9
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 11
				case 49:
					return 11
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return 8
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 12
				case 48:
					return 11
				case 49:
					return 11
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return 16
				case 73:
					return 16
				case 98:
					return -1
				case 101:
					return 16
				case 105:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)([\+\-])?([01]+|[01]+\/[01]+)-([01]+|[01]+\/[01]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return 8
				case 47:
					return 9
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 11
				case 49:
					return 11
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return 8
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 12
				case 48:
					return 11
				case 49:
					return 11
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 14
				case 49:
					return 14
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return 16
				case 73:
					return 16
				case 98:
					return -1
				case 101:
					return 16
				case 105:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)([\+\-])?([01]+|[01]+\/[01]+)[\+\-]i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, true, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 12
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return 8
				case 45:
					return 8
				case 47:
					return 9
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return 8
				case 45:
					return 8
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return 13
				case 73:
					return 13
				case 98:
					return -1
				case 101:
					return 13
				case 105:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[bB])|(#[bB])(#[eEiI])?)[\+\-]([01]+|[01]+\/[01]+)?i
		{[]bool{false, false, false, false, false, false, false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 2
				case 69:
					return 3
				case 73:
					return 3
				case 98:
					return 2
				case 101:
					return 3
				case 105:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 11
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
					return 4
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return 5
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return 5
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
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 9
				case 48:
					return 7
				case 49:
					return 7
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
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
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return 10
				case 49:
					return 10
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return 12
				case 73:
					return 12
				case 98:
					return -1
				case 101:
					return 12
				case 105:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 6
				case 45:
					return 6
				case 47:
					return -1
				case 48:
					return -1
				case 49:
					return -1
				case 66:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 98:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)([\+\-])?([0-7]+|[0-7]+\/[0-7]+)
		{[]bool{false, false, false, false, false, false, true, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 10
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return 9
				case 79:
					return -1
				case 101:
					return 9
				case 105:
					return 9
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 7
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 11
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 11
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)([\+\-])?([0-7]+|[0-7]+\/[0-7]+)@([\+\-])?([0-7]+|[0-7]+\/[0-7]+)
		{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 79:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 7
				case 64:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 16
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)([\+\-])?([0-7]+|[0-7]+\/[0-7]+)\+([0-7]+|[0-7]+\/[0-7]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 79:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return -1
				case 47:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 12
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 12
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 16
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)([\+\-])?([0-7]+|[0-7]+\/[0-7]+)-([0-7]+|[0-7]+\/[0-7]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 79:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 7
				case 47:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 7
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 12
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 12
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 16
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)([\+\-])?([0-7]+|[0-7]+\/[0-7]+)[\+\-]i
		{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 12
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 11
				case 73:
					return 11
				case 79:
					return -1
				case 101:
					return 11
				case 105:
					return 11
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return 7
				case 47:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 10
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return 7
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 13
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 13
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[oO])|(#[oO])(#[eEiI])?)[\+\-]([0-7]+|[0-7]+\/[0-7]+)?i
		{[]bool{false, false, false, false, false, false, true, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 79:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 111:
					return 3
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 11
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 10
				case 73:
					return 10
				case 79:
					return -1
				case 101:
					return 10
				case 105:
					return 10
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 6
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 8
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 6
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 7
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return 6
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return 12
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return 12
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 79:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 111:
					return -1
				}
				switch {
				case 48 <= r && r <= 55:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)
		{[]bool{false, false, false, false, false, false, true, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 10
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return 9
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return 9
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 7
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 8
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 8
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 8
				case 65 <= r && r <= 70:
					return 8
				case 97 <= r && r <= 102:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 8
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 8
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 8
				case 65 <= r && r <= 70:
					return 8
				case 97 <= r && r <= 102:
					return 8
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 11
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 11
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)@([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)
		{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 88:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 7
				case 64:
					return 8
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 64:
					return -1
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 12
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 12
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 12
				case 65 <= r && r <= 70:
					return 12
				case 97 <= r && r <= 102:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 12
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 12
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 12
				case 65 <= r && r <= 70:
					return 12
				case 97 <= r && r <= 102:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return 8
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 64:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)\+([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 88:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return -1
				case 47:
					return 8
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return 12
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return 12
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)-([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 15
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 14
				case 73:
					return 14
				case 88:
					return -1
				case 101:
					return 14
				case 105:
					return 14
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 7
				case 47:
					return 8
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 7
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 11
				case 69:
					return 10
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return 12
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				case 65 <= r && r <= 70:
					return 10
				case 97 <= r && r <= 102:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 13
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 13
				case 105:
					return 12
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				case 65 <= r && r <= 70:
					return 13
				case 97 <= r && r <= 102:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 16
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)([\+\-])?([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)[\+\-]i
		{[]bool{false, false, false, false, false, false, false, false, false, false, true, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 12
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 11
				case 73:
					return 11
				case 88:
					return -1
				case 101:
					return 11
				case 105:
					return 11
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return 7
				case 47:
					return 8
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return 10
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 7
				case 45:
					return 7
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 13
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 13
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[xX])|(#[xX])(#[eEiI])?)[\+\-]([0-9a-fA-F]+|[0-9a-fA-F]+\/[0-9a-fA-F]+)?i
		{[]bool{false, false, false, false, false, false, false, true, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 2
				case 73:
					return 2
				case 88:
					return 3
				case 101:
					return 2
				case 105:
					return 2
				case 120:
					return 3
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 11
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 4
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 10
				case 73:
					return 10
				case 88:
					return -1
				case 101:
					return 10
				case 105:
					return 10
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return 7
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return 8
				case 69:
					return 6
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 6
				case 105:
					return 7
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 6
				case 65 <= r && r <= 70:
					return 6
				case 97 <= r && r <= 102:
					return 6
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return 9
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return 9
				case 105:
					return 7
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				case 65 <= r && r <= 70:
					return 9
				case 97 <= r && r <= 102:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return 12
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 47:
					return -1
				case 69:
					return -1
				case 73:
					return -1
				case 88:
					return -1
				case 101:
					return -1
				case 105:
					return -1
				case 120:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				case 65 <= r && r <= 70:
					return -1
				case 97 <= r && r <= 102:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))
		{[]bool{false, false, false, false, true, true, false, false, false, true, true, false, true, false, true, true, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 16
				case 69:
					return 17
				case 70:
					return -1
				case 73:
					return 17
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 16
				case 101:
					return 17
				case 102:
					return -1
				case 105:
					return 17
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 5
				case 47:
					return 6
				case 68:
					return 7
				case 69:
					return 7
				case 70:
					return 7
				case 73:
					return -1
				case 76:
					return 7
				case 83:
					return 7
				case 100:
					return 7
				case 101:
					return 7
				case 102:
					return 7
				case 105:
					return -1
				case 108:
					return 7
				case 115:
					return 7
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 11
				case 69:
					return 11
				case 70:
					return 11
				case 73:
					return -1
				case 76:
					return 11
				case 83:
					return 11
				case 100:
					return 11
				case 101:
					return 11
				case 102:
					return 11
				case 105:
					return -1
				case 108:
					return 11
				case 115:
					return 11
				}
				switch {
				case 48 <= r && r <= 57:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 8
				case 45:
					return 8
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 9
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 13
				case 45:
					return 13
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 11
				case 69:
					return 11
				case 70:
					return 11
				case 73:
					return -1
				case 76:
					return 11
				case 83:
					return 11
				case 100:
					return 11
				case 101:
					return 11
				case 102:
					return 11
				case 105:
					return -1
				case 108:
					return 11
				case 115:
					return 11
				}
				switch {
				case 48 <= r && r <= 57:
					return 12
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 14
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 7
				case 69:
					return 7
				case 70:
					return 7
				case 73:
					return -1
				case 76:
					return 7
				case 83:
					return 7
				case 100:
					return 7
				case 101:
					return 7
				case 102:
					return 7
				case 105:
					return -1
				case 108:
					return 7
				case 115:
					return 7
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 20
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 18
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 19
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 19
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return 21
				case 70:
					return -1
				case 73:
					return 21
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 21
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))@([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, true, true, false, false, false, true, true, false, true, false, true, true, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return 31
				case 69:
					return 32
				case 70:
					return -1
				case 73:
					return 32
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 31
				case 101:
					return 32
				case 102:
					return -1
				case 105:
					return 32
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 5
				case 47:
					return 6
				case 64:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return 26
				case 69:
					return 26
				case 70:
					return 26
				case 73:
					return -1
				case 76:
					return 26
				case 83:
					return 26
				case 100:
					return 26
				case 101:
					return 26
				case 102:
					return 26
				case 105:
					return -1
				case 108:
					return 26
				case 115:
					return 26
				}
				switch {
				case 48 <= r && r <= 57:
					return 27
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 25
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 11
				case 45:
					return 11
				case 46:
					return 12
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 12
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 14
				case 47:
					return 15
				case 64:
					return -1
				case 68:
					return 16
				case 69:
					return 16
				case 70:
					return 16
				case 73:
					return -1
				case 76:
					return 16
				case 83:
					return 16
				case 100:
					return 16
				case 101:
					return 16
				case 102:
					return 16
				case 105:
					return -1
				case 108:
					return 16
				case 115:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return -1
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 21
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 19
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 17
				case 45:
					return 17
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 18
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 18
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 18
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 19
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 22
				case 45:
					return 22
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return -1
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 21
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return 16
				case 69:
					return 16
				case 70:
					return 16
				case 73:
					return -1
				case 76:
					return 16
				case 83:
					return 16
				case 100:
					return 16
				case 101:
					return 16
				case 102:
					return 16
				case 105:
					return -1
				case 108:
					return 16
				case 115:
					return 16
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 25
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 28
				case 45:
					return 28
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return 26
				case 69:
					return 26
				case 70:
					return 26
				case 73:
					return -1
				case 76:
					return 26
				case 83:
					return 26
				case 100:
					return 26
				case 101:
					return 26
				case 102:
					return 26
				case 105:
					return -1
				case 108:
					return 26
				case 115:
					return 26
				}
				switch {
				case 48 <= r && r <= 57:
					return 27
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 35
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 33
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return 34
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 34
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return 36
				case 70:
					return -1
				case 73:
					return 36
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 36
				case 102:
					return -1
				case 105:
					return 36
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 64:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))\+([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 31
				case 69:
					return 32
				case 70:
					return -1
				case 73:
					return 32
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 31
				case 101:
					return 32
				case 102:
					return -1
				case 105:
					return 32
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return 6
				case 47:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 16
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 14
				case 45:
					return 14
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 18
				case 47:
					return 19
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return 21
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 25
				case 69:
					return 25
				case 70:
					return 25
				case 73:
					return -1
				case 76:
					return 25
				case 83:
					return 25
				case 100:
					return 25
				case 101:
					return 25
				case 102:
					return 25
				case 105:
					return 21
				case 108:
					return 25
				case 115:
					return 25
				}
				switch {
				case 48 <= r && r <= 57:
					return 26
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 22
				case 45:
					return 22
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 27
				case 45:
					return 27
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 25
				case 69:
					return 25
				case 70:
					return 25
				case 73:
					return -1
				case 76:
					return 25
				case 83:
					return 25
				case 100:
					return 25
				case 101:
					return 25
				case 102:
					return 25
				case 105:
					return 21
				case 108:
					return 25
				case 115:
					return 25
				}
				switch {
				case 48 <= r && r <= 57:
					return 26
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return 21
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 35
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 33
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 34
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 34
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return 36
				case 70:
					return -1
				case 73:
					return 36
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 36
				case 102:
					return -1
				case 105:
					return 36
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))-([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 31
				case 69:
					return 32
				case 70:
					return -1
				case 73:
					return 32
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 31
				case 101:
					return 32
				case 102:
					return -1
				case 105:
					return 32
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return 6
				case 47:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 16
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 14
				case 45:
					return 14
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 18
				case 47:
					return 19
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return 21
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 25
				case 69:
					return 25
				case 70:
					return 25
				case 73:
					return -1
				case 76:
					return 25
				case 83:
					return 25
				case 100:
					return 25
				case 101:
					return 25
				case 102:
					return 25
				case 105:
					return 21
				case 108:
					return 25
				case 115:
					return 25
				}
				switch {
				case 48 <= r && r <= 57:
					return 26
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 22
				case 45:
					return 22
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 23
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 24
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 27
				case 45:
					return 27
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 25
				case 69:
					return 25
				case 70:
					return 25
				case 73:
					return -1
				case 76:
					return 25
				case 83:
					return 25
				case 100:
					return 25
				case 101:
					return 25
				case 102:
					return 25
				case 105:
					return 21
				case 108:
					return 25
				case 115:
					return 25
				}
				switch {
				case 48 <= r && r <= 57:
					return 26
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 21
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 28
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 20
				case 69:
					return 20
				case 70:
					return 20
				case 73:
					return -1
				case 76:
					return 20
				case 83:
					return 20
				case 100:
					return 20
				case 101:
					return 20
				case 102:
					return 20
				case 105:
					return 21
				case 108:
					return 20
				case 115:
					return 20
				}
				switch {
				case 48 <= r && r <= 57:
					return 29
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 30
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 35
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 33
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 34
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 34
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return 36
				case 70:
					return -1
				case 73:
					return 36
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 36
				case 102:
					return -1
				case 105:
					return 36
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)([\+\-])?([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))[\+\-]i
		{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 18
				case 69:
					return 19
				case 70:
					return -1
				case 73:
					return 19
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 18
				case 101:
					return 19
				case 102:
					return -1
				case 105:
					return 19
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return 6
				case 47:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 16
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 14
				case 45:
					return 14
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return -1
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 5
				case 45:
					return 5
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return -1
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 17
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 22
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 20
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 21
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 21
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return 23
				case 70:
					return -1
				case 73:
					return 23
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 23
				case 102:
					return -1
				case 105:
					return 23
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 4
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// ((#[eEiI])?(#[dD])?|(#[dD])?(#[eEiI])?)[\+\-]([0-9]+|[0-9]+\/[0-9]+|(\.?[0-9]+([eEsSfFdDlL][\+\-]?[0-9]+)?|[0-9]+\.[0-9]*([eEsSfFdDlL][\+\-]?[0-9]+)?))?i
		{[]bool{false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 17
				case 69:
					return 18
				case 70:
					return -1
				case 73:
					return 18
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 17
				case 101:
					return 18
				case 102:
					return -1
				case 105:
					return 18
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 3
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 4
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return 6
				case 47:
					return 7
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return 4
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 5
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return 4
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 9
				case 45:
					return 9
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 4
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 10
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 4
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 11
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 14
				case 45:
					return 14
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 12
				case 69:
					return 12
				case 70:
					return 12
				case 73:
					return -1
				case 76:
					return 12
				case 83:
					return 12
				case 100:
					return 12
				case 101:
					return 12
				case 102:
					return 12
				case 105:
					return 4
				case 108:
					return 12
				case 115:
					return 12
				}
				switch {
				case 48 <= r && r <= 57:
					return 13
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return 4
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return 15
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 8
				case 69:
					return 8
				case 70:
					return 8
				case 73:
					return -1
				case 76:
					return 8
				case 83:
					return 8
				case 100:
					return 8
				case 101:
					return 8
				case 102:
					return 8
				case 105:
					return 4
				case 108:
					return 8
				case 115:
					return 8
				}
				switch {
				case 48 <= r && r <= 57:
					return 16
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 21
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return 19
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return 20
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return 20
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return -1
				case 45:
					return -1
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return 22
				case 70:
					return -1
				case 73:
					return 22
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return 22
				case 102:
					return -1
				case 105:
					return 22
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 43:
					return 2
				case 45:
					return 2
				case 46:
					return -1
				case 47:
					return -1
				case 68:
					return -1
				case 69:
					return -1
				case 70:
					return -1
				case 73:
					return -1
				case 76:
					return -1
				case 83:
					return -1
				case 100:
					return -1
				case 101:
					return -1
				case 102:
					return -1
				case 105:
					return -1
				case 108:
					return -1
				case 115:
					return -1
				}
				switch {
				case 48 <= r && r <= 57:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

		// '|`|,|,@
		{[]bool{false, true, true, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 39:
					return 1
				case 44:
					return 2
				case 64:
					return -1
				case 96:
					return 3
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return -1
				case 44:
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
				case 39:
					return -1
				case 44:
					return -1
				case 64:
					return 4
				case 96:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 39:
					return -1
				case 44:
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
				case 39:
					return -1
				case 44:
					return -1
				case 64:
					return -1
				case 96:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

		// #\(
		{[]bool{false, false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 35:
					return 1
				case 40:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 40:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 35:
					return -1
				case 40:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},

		// \.
		{[]bool{false, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 46:
					return 1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 46:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

		// \(|\)
		{[]bool{false, true, true}, []func(rune) int{ // Transitions
			func(r rune) int {
				switch r {
				case 40:
					return 1
				case 41:
					return 2
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 40:
					return -1
				case 41:
					return -1
				}
				return -1
			},
			func(r rune) int {
				switch r {
				case 40:
					return -1
				case 41:
					return -1
				}
				return -1
			},
		}, []int{ /* Start-of-input transitions */ -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1}, nil},
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
				fmt.Println("expr keyword = ", yylex.Text())
			}
		case 1:
			{
				fmt.Println("stat keyword = ", yylex.Text())
			}
		case 2:
			{
				fmt.Println("identifier = ", yylex.Text())
			}
		case 3:
			{
				fmt.Println("bool = ", yylex.Text())
			}
		case 4:
			{
				fmt.Println("char = ", yylex.Text())
			}
		case 5:
			{
				fmt.Println("string = ", yylex.Text())
			}
		case 6:
			{
			}
		case 7:
			{
				fmt.Println("comment = ", yylex.Text())
			}
		case 8:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 9:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 10:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 11:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 12:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 13:
			{
				fmt.Println("bin num = ", yylex.Text())
			}
		case 14:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 15:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 16:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 17:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 18:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 19:
			{
				fmt.Println("octet num = ", yylex.Text())
			}
		case 20:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 21:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 22:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 23:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 24:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 25:
			{
				fmt.Println("hex num = ", yylex.Text())
			}
		case 26:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 27:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 28:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 29:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 30:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 31:
			{
				fmt.Println("decimal num = ", yylex.Text())
			}
		case 32:
			{
				fmt.Println("short stat = ", yylex.Text())
			}
		case 33:
			{
				fmt.Println("vector = ", yylex.Text())
			}
		case 34:
			{
				fmt.Println("connma = ", yylex.Text())
			}
		case 35:
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
