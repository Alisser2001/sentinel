# Sentinel - Linux System Monitor

A lightweight, real-time system resource monitor for Linux with an interactive TUI interface.

![Sentinel Screenshot](docs/screenshot.png)

## Features

- 📊 **Real-time monitoring** of CPU, memory, and process metrics
- 🎨 **Interactive TUI** with keyboard navigation (similar to htop)
- 🔍 **Process details** including PID, user, state, CPU%, MEM%, command
- ⚡ **Low overhead** - efficient parsing of `/proc` filesystem
- 🎯 **Configurable** refresh interval and sorting
- 🚀 **Fast** - written in Go with minimal dependencies

## Requirements

- Linux kernel 2.6+ (uses `/proc` filesystem)
- Go 1.23+ (for building from source)

## Installation

### From source

```bash
git clone https://github.com/yourusername/sentinel.git
cd sentinel
go build -o sentinel ./cmd
sudo mv sentinel /usr/local/bin/
```

### Pre-built binaries

Download from [Releases](https://github.com/yourusername/sentinel/releases)

## Usage

```bash
# Run with default settings (1s refresh)
sentinel

# Custom refresh interval
sentinel -interval 500ms

# Specify clock ticks (auto-detected by default)
sentinel -hz 250
```

### Keyboard shortcuts

- `↑/↓` or `j/k` - Navigate process list
- `s` - Toggle sort (CPU ↔ MEM)
- `q` or `Ctrl+C` - Quit

## Architecture

```
sentinel/
├── cmd/          # Entry point (main.go)
├── monitor/      # Core monitoring engine
│   ├── collector.go  # Process scanning & tracking
│   ├── engine.go     # Main loop & coordination
│   └── sorter.go     # Sorting algorithms
├── proc/         # /proc filesystem readers
│   ├── process.go    # Per-process metrics
│   ├── cpu.go        # System CPU stats
│   ├── mem.go        # Memory stats
│   └── hz.go         # CLK_TCK detection
├── model/        # Data structures
│   └── process.go    # ProcRec definition
└── ui/           # Terminal UI
    ├── tui.go        # Bubbletea model
    └── format.go     # Formatting utilities
```

## How it works

Sentinel reads Linux kernel metrics from the `/proc` pseudo-filesystem:

1. **Process discovery**: Scans `/proc/` for numeric directories (PIDs)
2. **Metrics collection**: Reads `/proc/<pid>/stat`, `/proc/<pid>/status`, `/proc/<pid>/cmdline`
3. **CPU calculation**: Computes %CPU using delta of process jiffies vs system jiffies
4. **Memory calculation**: Reads RSS (Resident Set Size) and compares to total memory
5. **Rendering**: Updates TUI using Bubbletea framework with 1-second interval

## Development

### Running tests

```bash
go test ./...
```

### Building with optimizations

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o sentinel ./cmd
```

### Adding new metrics

1. Add reader function in `proc/` package
2. Update `model.ProcRec` struct
3. Call reader in `monitor/collector.go`
4. Update `ui/tui.go` to display new column

## Troubleshooting

**"permission denied" errors**:
- Some `/proc` files require root access (e.g., `/proc/kcore`)
- Run with `sudo` if needed, though most metrics work for owned processes

**Incorrect CPU percentages**:
- Check HZ value with `getconf CLK_TCK`
- Override with `-hz <value>` flag

**Missing processes**:
- Processes may exit between scan and metric read
- This is expected; Sentinel marks them as `!Alive` and removes on next compact

## Performance

- Memory: ~5-10 MB resident
- CPU: <1% on idle system, 2-5% during updates
- Tested with 1000+ concurrent processes

## Roadmap

- [ ] Process tree view
- [ ] Kill/renice processes
- [ ] Network I/O per process
- [ ] Disk I/O metrics
- [ ] Configurable themes
- [ ] Export metrics (JSON/Prometheus)

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details

## Acknowledgments

- Inspired by `htop`, `top`, and `btop`
- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework
- Uses [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling

## Author

Your Name - [@yourhandle](https://github.com/yourusername)