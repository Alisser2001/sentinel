package model

// Change from const to var so it can be reassigned
var DefaultHZ = 1000

const MaxRows = 100

type ProcRec struct {
	Pid   int
	Uid   uint32
	User  string
	Comm  string // ← NUEVO: nombre del programa desde /proc/<pid>/stat
	State byte
	Prio  int64
	Nice  int64

	PrevProcTime uint64
	CurProcTime  uint64
	CPU          float64

	VSizeKB int64
	RSSKB   int64
	PMem    float64

	Cmd   string // cmdline completo
	Alive bool
}
