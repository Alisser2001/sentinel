package proc

import (
	"fmt"
	"os"
	"io"
	"strings"
	"strconv"
	"os/user"
)

func ReadCmdline(pid int) string {
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
	return strings.TrimSpace(string(data))
}

func ReadStatusUID(pid int) uint32 {
	path := fmt.Sprintf("/proc/%d/status", pid)
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	text := string(buf[:n])

	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.ParseUint(fields[1], 10, 32); err == nil {
					return uint32(v)
				}
			}
		}
	}
	return 0
}

func UIDToName(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return fmt.Sprint(uid)
	}
	return u.Username
}

func ReadProcStat(pid int) (comm string, state byte, utime, stime uint64,
	prio, nicev, nthreads int64, starttime uint64, vsizeKB, rssKB int64, ok bool) {

	path := fmt.Sprintf("/proc/%d/stat", pid)
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return
	}

	line := strings.TrimSpace(string(data))

	l := strings.IndexByte(line, '(')
	r := strings.LastIndexByte(line, ')')
	if l < 0 || r < 0 || r <= l {
		return
	}

	comm = line[l+1 : r]
	rest := strings.TrimSpace(line[r+1:])
	fields := strings.Fields(rest)

	if len(fields) < 24 {
		return
	}

	field := func(i int) string { return fields[i-3] }

	state = field(3)[0]
	utime, _ = strconv.ParseUint(field(14), 10, 64)
	stime, _ = strconv.ParseUint(field(15), 10, 64)
	prio, _ = strconv.ParseInt(field(18), 10, 64)
	nicev, _ = strconv.ParseInt(field(19), 10, 64)
	nthreads, _ = strconv.ParseInt(field(20), 10, 64)
	starttime, _ = strconv.ParseUint(field(22), 10, 64)

	vsize, _ := strconv.ParseInt(field(23), 10, 64)
	rss, _ := strconv.ParseInt(field(24), 10, 64)

	pageKB := int64(os.Getpagesize() / 1024)

	vsizeKB = vsize / 1024
	rssKB = rss * pageKB

	ok = true
	return
}