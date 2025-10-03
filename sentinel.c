#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <unistd.h>
#include <ctype.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <time.h>
#include <pwd.h>

#define INTERVAL_USEC 400000
#define MAX_ROWS 100

typedef unsigned long long ull;

typedef struct {
    pid_t pid;
    uid_t uid;
    char user[32];
    char state;
    long prio;
    long nicev;
    ull prev_proc_time;
    ull cur_proc_time;
    double cpu;
    long vsize_kb;
    long rss_kb;
    double pmem;
    char cmd[512];
    int alive;
} proc_rec;

static proc_rec *records = NULL;
static size_t records_len = 0;

static ull read_total_cpu_time() {
    FILE *f = fopen("/proc/stat", "r");
    if (!f) return 0;
    char line[1024];
    ull total = 0;
    if (fgets(line, sizeof(line), f)) {
        char *p = line;
        while (*p && !isspace((unsigned char)*p)) p++;
        while (*p && isspace((unsigned char)*p)) p++;
        char *tok = strtok(p, " ");
        while (tok) {
            total += strtoull(tok, NULL, 10);
            tok = strtok(NULL, " ");
        }
    }
    fclose(f);
    return total;
}

static int is_numeric(const char *s) {
    while (*s) {
        if (!isdigit((unsigned char)*s)) return 0;
        s++;
    }
    return 1;
}

static int find_record(pid_t pid) {
    for (size_t i = 0; i < records_len; ++i) if (records[i].pid == pid) return (int)i;
    return -1;
}

static void ensure_record_exists(pid_t pid) {
    int idx = find_record(pid);
    if (idx >= 0) {
        records[idx].alive = 1;
        return;
    }
    records = realloc(records, (records_len + 1) * sizeof(proc_rec));
    records[records_len].pid = pid;
    records[records_len].uid = 0;
    records[records_len].user[0] = '\0';
    records[records_len].state = '?';
    records[records_len].prio = 0;
    records[records_len].nicev = 0;
    records[records_len].prev_proc_time = 0;
    records[records_len].cur_proc_time = 0;
    records[records_len].cpu = 0.0;
    records[records_len].vsize_kb = 0;
    records[records_len].rss_kb = 0;
    records[records_len].pmem = 0.0;
    records[records_len].cmd[0] = '\0';
    records[records_len].alive = 1;
    records_len++;
}

static long read_mem_total_kb() {
    FILE *f = fopen("/proc/meminfo", "r");
    if (!f) return 1;
    char line[256];
    long mt = 1;
    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, "MemTotal:", 9) == 0) {
            char *p = line + 9;
            while (*p && !isdigit((unsigned char)*p)) p++;
            mt = atol(p);
            break;
        }
    }
    fclose(f);
    return mt > 0 ? mt : 1;
}

static void read_loadavg(double *l1, double *l5, double *l15) {
    FILE *f = fopen("/proc/loadavg", "r");
    if (!f) { *l1 = *l5 = *l15 = 0.0; return; }
    fscanf(f, "%lf %lf %lf", l1, l5, l15);
    fclose(f);
}

static void read_uptime(double *uptime_s) {
    FILE *f = fopen("/proc/uptime", "r");
    if (!f) { *uptime_s = 0.0; return; }
    fscanf(f, "%lf", uptime_s);
    fclose(f);
}

static void read_cmdline(pid_t pid, char *out, size_t n) {
    char path[256];
    snprintf(path, sizeof(path), "/proc/%d/cmdline", pid);
    FILE *f = fopen(path, "r");
    if (!f) { out[0] = '\0'; return; }
    size_t r = fread(out, 1, n - 1, f);
    fclose(f);
    if (r == 0) { out[0] = '\0'; return; }
    for (size_t i = 0; i < r; ++i) if (out[i] == '\0') out[i] = ' ';
    out[r] = '\0';
    char *p = out;
    while (*p == ' ') p++;
    if (p != out) memmove(out, p, strlen(p) + 1);
}

static void read_status_uid(pid_t pid, uid_t *uid) {
    char path[256];
    snprintf(path, sizeof(path), "/proc/%d/status", pid);
    FILE *f = fopen(path, "r");
    if (!f) { *uid = 0; return; }
    char line[256];
    *uid = 0;
    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, "Uid:", 4) == 0) {
            char *p = line + 4;
            while (*p && !isdigit((unsigned char)*p)) p++;
            *uid = (uid_t)strtoul(p, NULL, 10);
            break;
        }
    }
    fclose(f);
}

static void uid_to_name(uid_t uid, char *out, size_t n) {
    struct passwd *pw = getpwuid(uid);
    if (pw && pw->pw_name) {
        strncpy(out, pw->pw_name, n - 1);
        out[n - 1] = '\0';
    } else {
        snprintf(out, n, "%u", (unsigned)uid);
    }
}

static int read_proc_stat(pid_t pid, char *comm_out, size_t comm_len, char *state, ull *utime, ull *stime, long *prio, long *nicev, long *num_threads, ull *starttime, long *vsize_kb, long *rss_kb) {
    char path[256];
    snprintf(path, sizeof(path), "/proc/%d/stat", pid);
    FILE *f = fopen(path, "r");
    if (!f) return 0;
    char buf[2048];
    if (!fgets(buf, sizeof(buf), f)) { fclose(f); return 0; }
    fclose(f);
    char *l = strchr(buf, '(');
    char *r = strrchr(buf, ')');
    if (!l || !r || r < l) return 0;
    size_t n = r - l - 1;
    if (n >= comm_len) n = comm_len - 1;
    memcpy(comm_out, l + 1, n);
    comm_out[n] = '\0';
    char *p = r + 2;
    int field = 3;
    char *tok = strtok(p, " ");
    ull ut=0, st=0, stt=0;
    long pr=0, ni=0, th=0;
    long vsize=0, rss_pages=0;
    char stc='?';
    long page_kb = sysconf(_SC_PAGESIZE) / 1024;
    while (tok) {
        if (field == 3) stc = tok[0];
        if (field == 14) ut = strtoull(tok, NULL, 10);
        if (field == 15) st = strtoull(tok, NULL, 10);
        if (field == 18) pr = strtol(tok, NULL, 10);
        if (field == 19) ni = strtol(tok, NULL, 10);
        if (field == 20) th = strtol(tok, NULL, 10);
        if (field == 22) stt = strtoull(tok, NULL, 10);
        if (field == 23) vsize = strtol(tok, NULL, 10);
        if (field == 24) { rss_pages = strtol(tok, NULL, 10); break; }
        tok = strtok(NULL, " ");
        field++;
    }
    *state = stc;
    *utime = ut;
    *stime = st;
    *prio = pr;
    *nicev = ni;
    *num_threads = th;
    *starttime = stt;
    *vsize_kb = (long)(vsize / 1024);
    *rss_kb = (long)(rss_pages * page_kb);
    return 1;
}

static int cmp_cpu_desc(const void *a, const void *b) {
    const proc_rec *pa = a;
    const proc_rec *pb = b;
    if (pa->cpu < pb->cpu) return 1;
    if (pa->cpu > pb->cpu) return -1;
    if (pa->rss_kb < pb->rss_kb) return 1;
    if (pa->rss_kb > pb->rss_kb) return -1;
    return 0;
}

static void fmt_time_ticks(ull ticks, char *out, size_t n) {
    long hz = sysconf(_SC_CLK_TCK);
    ull total_cs = (ticks * 100) / hz;
    ull h = total_cs / 360000;
    ull m = (total_cs % 360000) / 6000;
    ull s = (total_cs % 6000) / 100;
    ull cs = total_cs % 100;
    if (h > 0) snprintf(out, n, "%lluh%02llum%02llus", h, m, s);
    else snprintf(out, n, "%02llu:%02llu.%02llu", m, s, cs);
}

int main(void) {
    ull prev_total = read_total_cpu_time();
    long mem_total_kb = read_mem_total_kb();
    while (1) {
        DIR *d = opendir("/proc");
        if (!d) return 1;
        struct dirent *ent;
        for (size_t i = 0; i < records_len; ++i) { records[i].alive = 0; records[i].cur_proc_time = 0; }
        size_t tasks = 0, running = 0;
        while ((ent = readdir(d)) != NULL) {
            if (!is_numeric(ent->d_name)) continue;
            pid_t pid = (pid_t)atoi(ent->d_name);
            ensure_record_exists(pid);
            int idx = find_record(pid);
            if (idx < 0) continue;
            char comm[256] = {0};
            char stc = '?';
            ull ut=0, st=0, stt=0;
            long pr=0, ni=0, th=0;
            long vsize_kb=0, rss_kb=0;
            if (!read_proc_stat(pid, comm, sizeof(comm), &stc, &ut, &st, &pr, &ni, &th, &stt, &vsize_kb, &rss_kb)) {
                records[idx].alive = 0;
                continue;
            }
            records[idx].alive = 1;
            records[idx].state = stc;
            records[idx].prio = pr;
            records[idx].nicev = ni;
            records[idx].vsize_kb = vsize_kb;
            records[idx].rss_kb = rss_kb;
            read_status_uid(pid, &records[idx].uid);
            uid_to_name(records[idx].uid, records[idx].user, sizeof(records[idx].user));
            read_cmdline(pid, records[idx].cmd, sizeof(records[idx].cmd));
            if (records[idx].cmd[0] == '\0') {
                strncpy(records[idx].cmd, comm, sizeof(records[idx].cmd)-1);
            }
            records[idx].cur_proc_time = ut + st;
            tasks++;
            if (stc == 'R') running++;
        }
        closedir(d);

        ull cur_total = read_total_cpu_time();
        ull sys_delta = (cur_total > prev_total) ? (cur_total - prev_total) : 1;
        for (size_t i = 0; i < records_len; ++i) {
            if (!records[i].alive) continue;
            if (records[i].prev_proc_time == 0) {
                records[i].cpu = 0.0;
            } else {
                ull proc_delta = (records[i].cur_proc_time > records[i].prev_proc_time) ? (records[i].cur_proc_time - records[i].prev_proc_time) : 0;
                records[i].cpu = (double)proc_delta * 100.0 / (double)sys_delta;
            }
            records[i].pmem = mem_total_kb > 0 ? (100.0 * (double)records[i].rss_kb / (double)mem_total_kb) : 0.0;
        }
        for (size_t i = 0; i < records_len; ++i) if (records[i].alive) records[i].prev_proc_time = records[i].cur_proc_time;
        prev_total = cur_total;
        mem_total_kb = read_mem_total_kb();

        if (records_len > 0) qsort(records, records_len, sizeof(proc_rec), cmp_cpu_desc);

        double l1=0,l5=0,l15=0, up=0;
        read_loadavg(&l1,&l5,&l15);
        read_uptime(&up);

        printf("\033[H\033[J");
        time_t t = time(NULL);
        char *ts = ctime(&t);
        printf("SimpleMonitor %s", ts ? ts : "");
        printf("Tasks: %zu, running: %zu\n", tasks, running);
        printf("Load average: %.2f %.2f %.2f  | Uptime: %.0fs\n", l1, l5, l15, up);
        printf("%5s %-15s %3s %3s %1s %6s %6s %8s %8s %9s %s\n",
               "PID","USER","PR","NI","S","%CPU","%MEM","VIRT(KB)","RES(KB)","TIME+","COMMAND");

        size_t shown = 0;
        for (size_t i = 0; i < records_len && shown < MAX_ROWS; ++i) {
            if (!records[i].alive) continue;
            char timebuf[32]; fmt_time_ticks(records[i].cur_proc_time, timebuf, sizeof(timebuf));
            printf("%5d %-15s %3ld %3ld %1c %6.2f %6.2f %8ld %8ld %9s %.30s\n",
                   (int)records[i].pid,
                   records[i].user,
                   records[i].prio,
                   records[i].nicev,
                   records[i].state,
                   records[i].cpu,
                   records[i].pmem,
                   records[i].vsize_kb,
                   records[i].rss_kb,
                   timebuf,
                   records[i].cmd);
            shown++;
        }
        fflush(stdout);
        usleep(INTERVAL_USEC);
    }
    free(records);
    return 0;
}
