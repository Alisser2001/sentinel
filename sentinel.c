#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <unistd.h>
#include <ctype.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <time.h>

#define INTERVAL_USEC 800000
#define TOP_N 12

typedef unsigned long long ull;

typedef struct {
    pid_t pid;
    ull prev_proc_time;
    double cpu;
    long rss_kb;
    char name[256];
    int alive;
} proc_rec;

static proc_rec *records = NULL;
static size_t records_len = 0;

static ull read_total_cpu_time() {
    FILE *f = fopen("/proc/stat", "r");
    if (!f) return 0;
    char line[512];
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
    records[records_len].prev_proc_time = 0;
    records[records_len].cpu = 0.0;
    records[records_len].rss_kb = 0;
    records[records_len].name[0] = '\0';
    records[records_len].alive = 1;
    records_len++;
}

static void remove_dead_records() {
    size_t write = 0;
    for (size_t i = 0; i < records_len; ++i) {
        if (records[i].alive) {
            if (write != i) records[write] = records[i];
            write++;
        }
    }
    records_len = write;
    records = realloc(records, records_len * sizeof(proc_rec));
}

static int read_proc_stat(pid_t pid, ull *proc_time, char *name_out, size_t name_len) {
    char path[256];
    snprintf(path, sizeof(path), "/proc/%d/stat", pid);
    FILE *f = fopen(path, "r");
    if (!f) return 0;
    char buf[1024];
    if (!fgets(buf, sizeof(buf), f)) { fclose(f); return 0; }
    fclose(f);
    char *start = strchr(buf, '(');
    char *end = strrchr(buf, ')');
    if (!start || !end || end < start) return 0;
    size_t n = end - start - 1;
    if (n >= name_len) n = name_len - 1;
    memcpy(name_out, start + 1, n);
    name_out[n] = '\0';
    char *p = end + 2;
    int field = 3;
    char *tok;
    ull utime = 0, stime = 0;
    tok = strtok(p, " ");
    while (tok) {
        if (field == 14) utime = strtoull(tok, NULL, 10);
        if (field == 15) { stime = strtoull(tok, NULL, 10); break; }
        tok = strtok(NULL, " ");
        field++;
    }
    *proc_time = utime + stime;
    return 1;
}

static long read_proc_rss_kb(pid_t pid) {
    char path[256];
    snprintf(path, sizeof(path), "/proc/%d/status", pid);
    FILE *f = fopen(path, "r");
    if (!f) return 0;
    char line[256];
    long rss = 0;
    while (fgets(line, sizeof(line), f)) {
        if (strncmp(line, "VmRSS:", 6) == 0) {
            char *p = line + 6;
            while (*p && !isdigit((unsigned char)*p)) p++;
            rss = atol(p);
            break;
        }
    }
    fclose(f);
    return rss;
}

static int cmp_proc(const void *a, const void *b) {
    const proc_rec *pa = a;
    const proc_rec *pb = b;
    if (pa->cpu < pb->cpu) return 1;
    if (pa->cpu > pb->cpu) return -1;
    return 0;
}

int main(void) {
    ull prev_total = read_total_cpu_time();
    while (1) {
        DIR *d = opendir("/proc");
        if (!d) return 1;
        struct dirent *ent;
        for (size_t i = 0; i < records_len; ++i) records[i].alive = 0;
        while ((ent = readdir(d)) != NULL) {
            if (!is_numeric(ent->d_name)) continue;
            pid_t pid = (pid_t)atoi(ent->d_name);
            ensure_record_exists(pid);
            int idx = find_record(pid);
            if (idx < 0) continue;
            ull cur_proc_time = 0;
            char name[256] = {0};
            if (!read_proc_stat(pid, &cur_proc_time, name, sizeof(name))) {
                records[idx].alive = 0;
                continue;
            }
            records[idx].alive = 1;
            records[idx].rss_kb = read_proc_rss_kb(pid);
            strncpy(records[idx].name, name, sizeof(records[idx].name)-1);
            if (records[idx].prev_proc_time == 0) {
                records[idx].prev_proc_time = cur_proc_time;
                records[idx].cpu = 0.0;
            } else {
                ull cur_total = read_total_cpu_time();
                ull proc_delta = cur_proc_time - records[idx].prev_proc_time;
                ull sys_delta = (cur_total > prev_total) ? (cur_total - prev_total) : 1;
                records[idx].cpu = (double)proc_delta * 100.0 / (double)sys_delta;
                records[idx].prev_proc_time = cur_proc_time;
                prev_total = cur_total;
            }
        }
        closedir(d);
        remove_dead_records();
        if (records_len > 0) qsort(records, records_len, sizeof(proc_rec), cmp_proc);
        printf("\033[H\033[J");
        time_t t = time(NULL);
        printf("SimpleMonitor - %s", ctime(&t));
        printf("%5s %6s %8s %s\n", "PID", "%CPU", "RSS(KB)", "COMMAND");
        size_t toshow = records_len < TOP_N ? records_len : TOP_N;
        for (size_t i = 0; i < toshow; ++i) {
            printf("%5d %6.2f %8ld %s\n", (int)records[i].pid, records[i].cpu, records[i].rss_kb, records[i].name);
        }
        fflush(stdout);
        usleep(INTERVAL_USEC);
    }
    free(records);
    return 0;
}
