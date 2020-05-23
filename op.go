package genetic_sort

// The OPs for Brainfuck. Also in here are OP sets for predefined functions. I
// didn't want to try to evolve (and identify) instructions from random
// permutations. So we have some starter functions for swapping items,
// searching for empty cells, assignment, etc.

// There is one special instruction added: ^
// This will jump the memory cell pointer back to the index before the last contiguous list of moves in a single direction. As an example, consider the following

//  *
// [1][2][3][4][5][6][7][8][9][0]
// [>]
// Search for a zero cell to the right

//                             *
// [1][2][3][4][5][6][7][8][9][0]
// ^[-^+^]
// Now jump back to the last address, enter loop, dec 1, jump forward, inc 1,
// jump back, now zero, break loop

//  *
// [0][2][3][4][5][6][7][8][9][1]
// >[-^+^]
// Move right one, enter loop, dec 1, jump back left, inc 1, loop, etc.

//     *
// [2][0][3][4][5][6][7][8][9][1]

// [-^+^] == very concise 0:N swap

type OP string
type OPS string

const (
	OP_POINTER_LEFT  = OP("<")
	OP_POINTER_RIGHT = OP(">")
	OP_INC           = OP("+")
	OP_DEC           = OP("-")
	OP_WHILE         = OP("[")
	OP_WHILE_END     = OP("]")
	OP_JUMP          = OP("^")
	NO_OP            = OP("#")
)

const (
	SET_TO_ZERO        = OPS(`[-]`)
	FIND_ZERO_RIGHT    = OPS(`[>]`)
	FIND_ZERO_LEFT     = OPS(`[<]`)
	SWAP_RIGHT         = OPS(`[>]^[-^+^]^[-^>+^]`)
	SWAP_LEFT          = OPS(`[>]^[-^+^]^[-^<+^]`)
	MOVE_TO_ZERO_RIGHT = OPS(`[>]^[-^+^]`)
	MOVE_TO_ZERO_LEFT  = OPS(`[<]^[-^+^]`)
)

const OP_SET = []OP{
	OP_POINTER_LEFT,
	OP_POINTER_RIGHT,
	OP_INC,
	OP_DEC,
	OP_WHILE,
	OP_WHILE_END,
	OP_CELL_REF,
	OP_LAST_CELL,
	NO_OP,
}

const PREFAB_OPSETS = []OPS{
	SET_TO_ZERO,
	FIND_ZERO_RIGHT,
	FIND_ZERO_LEFT,
	SWAP_RIGHT,
	SWAP_LEFT,
	MOVE_TO_ZERO_RIGHT,
	MOVE_TO_ZERO_LEFT,
}

func (o OPS) ToOPs() []OP {
	ops := []OP{}
	for _, r := range o {
		ops = append(ops, OP(r))
	}
	return ops
}
