package props

// PropID identifies a property: it is the index of the property in Table.
// The identifiers themselves and the lookups on them are in props_gen.go.
type PropID uint16

type bitset []uint64

func (b bitset) has(id PropID) bool { return b[id/64]&(1<<(id%64)) != 0 }
