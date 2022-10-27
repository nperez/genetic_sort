package genetic_sort

const (
	DEBUG                                       = true
	Alive                                       = 1
	Dead                                        = 2
	FailedMachineRun           SelectFailReason = 1
	FailedSetFidelity          SelectFailReason = 2
	FailedSortedness           SelectFailReason = 3
	FailedInstructionCount     SelectFailReason = 4
	FailedInstructionsExecuted SelectFailReason = 5
	PUSH_OP                    byte             = iota
	POP_OP
	SHIFT_OP
	UNSHIFT_OP
	INSERT_OP
	DELETE_OP
	SWAP_OP
	REPLACE_OP
	META_NO_OP
)
