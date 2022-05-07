package brainfuck

import ()

// The OPs for Brainfuck. Also in here are OP sets for predefined functions. I
// didn't want to try to evolve (and identify) instructions from random
// permutations. So we have some starter functions for swapping items,
// searching for empty cells, assignment, etc.

// There two special instructions added: * ^
// Additionally, there is now a new bookmark register to store/recall a memory tape index. The ^ instruction will recall the stored memory tape index, store the current index, then jump the memory tape pointer to the recalled index. The * instruction will just store the current memory index to the register.

// The uninitialized bookmark is always zero

//  *
// [1][2][3][4][5][6][7][8][9][0]
// *[>]
// Bookmark the start. Search for a zero cell to the right

//                             *
// [1][2][3][4][5][6][7][8][9][0]
// ^[-^+^]
// Bookmark dance. Enter loop, loop eval: now one, dec 1, bookmark dance, inc 1, bookmark dance, loop eval: now zero, break loop

//     *
// [0][2][3][4][5][6][7][8][9][1]
// *>[-^+^]
// Bookmark. Move right one, enter loop, dec 1, jump back left, inc 1, loop, etc.

//     *
// [2][0][3][4][5][6][7][8][9][1]

// [-^+^] == very concise 0:N swap

const (
	OP_POINTER_LEFT  = '<'
	OP_POINTER_RIGHT = '>'
	OP_INC           = '+'
	OP_DEC           = '-'
	OP_WHILE         = '['
	OP_WHILE_END     = ']'
	OP_JUMP          = '^'
	OP_BOOKMARK      = '*'
	NO_OP            = '#'
)

const (
	SET_TO_ZERO        = `[-]`
	FIND_ZERO_RIGHT    = `[>]`
	FIND_ZERO_LEFT     = `[<]`
	SWAP_RIGHT         = `*[>]^[-^+^]>[-<+>]^[-^+^]`
	SWAP_LEFT          = `>*[>]^<[-^+^]>[-<+>]^[-^+^]`
	MOVE_TO_ZERO_RIGHT = `*[>]^[-^+^]`
	MOVE_TO_ZERO_LEFT  = `*[<]^[-^+^]`
)

var OP_SET [9]byte = [...]byte{
	OP_POINTER_LEFT,
	OP_POINTER_RIGHT,
	OP_INC,
	OP_DEC,
	OP_WHILE,
	OP_WHILE_END,
	OP_JUMP,
	OP_BOOKMARK,
	NO_OP,
}

var PREFAB_OPSETS [7]string = [...]string{
	SET_TO_ZERO,
	FIND_ZERO_RIGHT,
	FIND_ZERO_LEFT,
	SWAP_RIGHT,
	SWAP_LEFT,
	MOVE_TO_ZERO_RIGHT,
	MOVE_TO_ZERO_LEFT,
}
