package model

type ProcRec struct {
	Pid          int
	Uid          uint32
	User         string
	State        byte
	Prio         int64
	Nice         int64
	PrevProcTime uint64
	CurProcTime  uint64
	CPU          float64
	VSizeKB      int64
	RSSKB        int64
	PMem         float64
	Cmd          string
	Alive        bool
}

const (
	DefaultHZ   = 100
	MaxRows     = 100
)
