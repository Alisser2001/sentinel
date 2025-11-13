// SimpleMonitor en Go - versión inspirada en el código C dado.
// Compila con: go build -o simplemonitor main.go
// Ejecuta con: ./simplemonitor

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	interval    = 400 * time.Millisecond
	maxRows     = 100
	defaultHZ   = 100 // Aproximación para convertir ticks a tiempo humano
)

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

var (
	records []ProcRec
)

// ---------- Utilidades básicas ----------

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func findRecordIndex(pid int) int {
	for i := range records {
		if records[i].Pid == pid {
			return i
		}
	}
	return -1
}

func ensureRecordExists(pid int) int {
	idx := findRecordIndex(pid)
	if idx >= 0 {
		records[idx].Alive = true
		return idx
	}
	rec := ProcRec{
		Pid:          pid,
		User:         "",
		State:        '?',
		Prio:         0,
		Nice:         0,
		PrevProcTime: 0,
		CurProcTime:  0,
		CPU:          0,
		VSizeKB:      0,
		RSSKB:        0,
		PMem:         0,
		Cmd:          "",
		Alive:        true,
	}
	records = append(records, rec)
	return len(records) - 1
}

// ---------- Lectura de /proc global ----------

func readTotalCPUTime() uint64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0
	}

	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "cpu") {
		return 0
	}

	var total uint64
	for _, tok := range fields[1:] {
		val, err := strconv.ParseUint(tok, 10, 64)
		if err != nil {
			continue
		}
		total += val
	}
	return total
}

func readMemTotalKB() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 1
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			for _, tok := range fields {
				if v, err := strconv.ParseInt(tok, 10, 64); err == nil {
					if v > 0 {
						return v
					}
				}
			}
			break
		}
	}
	return 1
}

func readLoadavg() (float64, float64, float64) {
	f, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()

	var l1, l5, l15 float64
	fmt.Fscan(f, &l1, &l5, &l15)
	return l1, l5, l15
}

func readUptime() float64 {
	f, err := os.Open("/proc/uptime")
	if err != nil {
		return 0
	}
	defer f.Close()

	var up float64
	fmt.Fscan(f, &up)
	return up
}

// ---------- Lectura de info por proceso ----------

func readCmdline(pid int) string {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	for i := range data {
		if data[i] == 0 {
			data[i] = ' '
		}
	}
	// Elimina espacios iniciales
	cmd := strings.TrimSpace(string(data))
	return cmd
}

func readStatusUID(pid int) uint32 {
	path := fmt.Sprintf("/proc/%d/status", pid)
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			for _, tok := range fields[1:] {
				if u, err := strconv.ParseUint(tok, 10, 32); err == nil {
					return uint32(u)
				}
			}
			break
		}
	}
	return 0
}

func uidToName(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil || u.Username == "" {
		return strconv.FormatUint(uint64(uid), 10)
	}
	return u.Username
}

func readProcStat(
	pid int,
) (comm string, state byte, utime, stime uint64, prio, nicev, numThreads int64,
	starttime uint64, vsizeKB, rssKB int64, ok bool) {

	path := fmt.Sprintf("/proc/%d/stat", pid)
	f, err := os.Open(path)
	if err != nil {
		return "", '?', 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", '?', 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	line := strings.TrimSpace(string(data))

	l := strings.IndexByte(line, '(')
	r := strings.LastIndexByte(line, ')')
	if l < 0 || r < 0 || r <= l {
		return "", '?', 0, 0, 0, 0, 0, 0, 0, 0, false
	}

	comm = line[l+1 : r]
	rest := strings.TrimSpace(line[r+1:])
	fields := strings.Fields(rest)
	if len(fields) < 24 { // necesitamos al menos hasta el campo 24
		return "", '?', 0, 0, 0, 0, 0, 0, 0, 0, false
	}

	// El primer token en 'fields' es el campo 3 (state)
	// Campos según man proc (stat):
	// 3: state (char)
	// 14: utime, 15: stime, 18: priority, 19: nice, 20: num_threads,
	// 22: starttime, 23: vsize, 24: rss
	var stc byte = '?'
	var ut, st, stt uint64
	var pr, ni, th int64
	var vsize, rssPages int64
	pageKB := int64(os.Getpagesize() / 1024)

	for i, tok := range fields {
		field := i + 3 // fields[0] -> field 3
		switch field {
		case 3:
			if len(tok) > 0 {
				stc = tok[0]
			}
		case 14:
			val, _ := strconv.ParseUint(tok, 10, 64)
			ut = val
		case 15:
			val, _ := strconv.ParseUint(tok, 10, 64)
			st = val
		case 18:
			val, _ := strconv.ParseInt(tok, 10, 64)
			pr = val
		case 19:
			val, _ := strconv.ParseInt(tok, 10, 64)
			ni = val
		case 20:
			val, _ := strconv.ParseInt(tok, 10, 64)
			th = val
		case 22:
			val, _ := strconv.ParseUint(tok, 10, 64)
			stt = val
		case 23:
			val, _ := strconv.ParseInt(tok, 10, 64)
			vsize = val
		case 24:
			val, _ := strconv.ParseInt(tok, 10, 64)
			rssPages = val
		}
	}

	vsizeKB = vsize / 1024
	rssKB = rssPages * pageKB

	return comm, stc, ut, st, pr, ni, th, stt, vsizeKB, rssKB, true
}

// ---------- Formato de tiempo de CPU (ticks -> hh:mm:ss) ----------

func fmtTimeTicks(ticks uint64) string {
	// Similar a la versión en C:
	// total_cs = (ticks * 100) / HZ   (centésimas)
	if defaultHZ <= 0 {
		return "00:00.00"
	}
	totalCS := (ticks * 100) / uint64(defaultHZ)
	h := totalCS / 360000
	m := (totalCS % 360000) / 6000
	s := (totalCS % 6000) / 100
	cs := totalCS % 100

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d.%02d", m, s, cs)
}

// ---------- Ordenamiento ----------

func sortByCPUDesc() {
	sort.Slice(records, func(i, j int) bool {
		a := records[i]
		b := records[j]

		if a.Alive != b.Alive {
			return a.Alive && !b.Alive
		}
		if a.CPU != b.CPU {
			return a.CPU > b.CPU
		}
		if a.RSSKB != b.RSSKB {
			return a.RSSKB > b.RSSKB
		}
		return a.Pid < b.Pid
	})
}

// ---------- Main loop ----------

func main() {
	prevTotal := readTotalCPUTime()
	memTotalKB := readMemTotalKB()

	for {
		// Marcar como no vivos y resetear cur_proc_time
		for i := range records {
			records[i].Alive = false
			records[i].CurProcTime = 0
		}

		// Leer /proc
		entries, err := os.ReadDir("/proc")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error abriendo /proc: %v\n", err)
			return
		}

		var tasks, running int
		for _, ent := range entries {
			name := ent.Name()
			if !isNumeric(name) {
				continue
			}
			pid, err := strconv.Atoi(name)
			if err != nil {
				continue
			}

			idx := ensureRecordExists(pid)
			rec := &records[idx]

			comm, state, ut, st, pr, ni, _, _, vsizeKB, rssKB, ok := readProcStat(pid)
			if !ok {
				rec.Alive = false
				continue
			}

			rec.Alive = true
			rec.State = state
			rec.Prio = pr
			rec.Nice = ni
			rec.VSizeKB = vsizeKB
			rec.RSSKB = rssKB

			uid := readStatusUID(pid)
			rec.Uid = uid
			rec.User = uidToName(uid)

			cmd := readCmdline(pid)
			if cmd == "" {
				cmd = comm
			}
			rec.Cmd = cmd

			rec.CurProcTime = ut + st
			tasks++
			if state == 'R' {
				running++
			}
		}

		// CPU global
		curTotal := readTotalCPUTime()
		sysDelta := uint64(1)
		if curTotal > prevTotal {
			sysDelta = curTotal - prevTotal
		}

		// Calcular %CPU y %MEM
		for i := range records {
			rec := &records[i]
			if !rec.Alive {
				continue
			}
			if rec.PrevProcTime == 0 {
				rec.CPU = 0
			} else {
				procDelta := uint64(0)
				if rec.CurProcTime > rec.PrevProcTime {
					procDelta = rec.CurProcTime - rec.PrevProcTime
				}
				rec.CPU = float64(procDelta) * 100.0 / float64(sysDelta)
			}
			if memTotalKB > 0 {
				rec.PMem = float64(rec.RSSKB) * 100.0 / float64(memTotalKB)
			} else {
				rec.PMem = 0
			}
		}

		// Actualizar prev_proc_time sólo para vivos
		for i := range records {
			if records[i].Alive {
				records[i].PrevProcTime = records[i].CurProcTime
			}
		}
		prevTotal = curTotal
		memTotalKB = readMemTotalKB()

		// Compactar: quedarnos sólo con procesos vivos
		compacted := records[:0]
		for _, r := range records {
			if r.Alive {
				compacted = append(compacted, r)
			}
		}
		records = compacted

		// Ordenar
		if len(records) > 0 {
			sortByCPUDesc()
		}

		// Info global
		l1, l5, l15 := readLoadavg()
		up := readUptime()

		// Limpiar pantalla
		fmt.Print("\033[H\033[J")

		// Header
		now := time.Now()
		fmt.Printf("SimpleMonitor %s\n", now.Format(time.RFC1123))
		fmt.Printf("Tasks: %d, running: %d\n", tasks, running)
		fmt.Printf("Load average: %.2f %.2f %.2f  | Uptime: %.0fs\n", l1, l5, l15, up)
		fmt.Printf("%5s %-15s %3s %3s %1s %6s %6s %8s %8s %9s %s\n",
			"PID", "USER", "PR", "NI", "S", "%CPU", "%MEM", "VIRT(KB)", "RES(KB)", "TIME+", "COMMAND")

		// Listado
		shown := 0
		for _, r := range records {
			if !r.Alive {
				continue
			}
			if shown >= maxRows {
				break
			}
			timeStr := fmtTimeTicks(r.CurProcTime)
			cmd := r.Cmd
			if len(cmd) > 30 {
				cmd = cmd[:30]
			}
			fmt.Printf("%5d %-15s %3d %3d %1c %6.2f %6.2f %8d %8d %9s %s\n",
				r.Pid,
				r.User,
				r.Prio,
				r.Nice,
				r.State,
				r.CPU,
				r.PMem,
				r.VSizeKB,
				r.RSSKB,
				timeStr,
				cmd,
			)
			shown++
		}

		// Flush
		fmt.Print()
		time.Sleep(interval)
	}
}
