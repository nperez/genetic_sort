package genetic_sort

import "log"

type Tombstone struct {
	ID     uint
	UnitID uint
	Reason SelectFailReason
}

func NewTombstone(u *Unit, reason SelectFailReason) *Tombstone {
	if u.ID == 0 {
		log.Fatalf("Unit must be persisted prior to tombstone creation")
	}
	return &Tombstone{
		UnitID: u.ID,
		Reason: reason,
	}
}
