# saferwall-cli

A CLI tool to interact with Saferwall.com to scan, rescan and download malware samples.

## Install

You can either download pre-built binaries or build the tool yourself.

```sh
go install github.com/saferwall/cli@latest
```

## Getting Started

To use the CLI you need a [Saferwall](https://saferwall.com) account. Run the `init` command to interactively set up your credentials:

```sh
saferwall-cli init
```

This launches an interactive prompt that asks for:
- **URL** — the Saferwall API endpoint (defaults to `https://api.saferwall.com`)
- **Username** — your Saferwall account username
- **Password** — your Saferwall account password

The credentials are saved to `~/.config/saferwall/config.toml`. To reconfigure, delete that file and run `init` again.

The CLI can also be used with a self-hosted Saferwall deployment by providing your own API URL during init.

## Usage

```
Available Commands:
  init        Configure saferwall CLI credentials
  scan        Upload and scan files
  rescan      Rescan an existing file using its hash
  view        View scan results for a file by its SHA256 hash
  download    Download a sample (and its artifacts)
  souk        Populate malware-souk database
  version     Version number
```

### Scan

Upload and scan files. Supports scanning a single file or an entire directory.

```sh
# Scan a single file
saferwall-cli scan /path/to/sample

# Scan an entire directory
saferwall-cli scan /path/to/directory

# Scan with parallel uploads
saferwall-cli scan -p 4 /path/to/directory

# Force rescan if the file already exists
saferwall-cli scan -f /path/to/sample

# Enable detonation with custom timeout and OS
saferwall-cli scan -d -t 30 -o win-7 /path/to/sample
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--force` | `-f` | `false` | Force rescan if the file already exists |
| `--parallel` | `-p` | `1` | Number of files to scan in parallel |
| `--enableDetonation` | `-d` | `false` | Enable detonation (dynamic analysis) |
| `--timeout` | `-t` | `15` | Detonation duration in seconds |
| `--os` | `-o` | `win-10` | Preferred OS for detonation (`win-7` or `win-10`) |

### Rescan

Rescan an existing file by its SHA256 hash, or rescan a batch of hashes from a text file.

```sh
saferwall-cli rescan <sha256>
```

### View

View scan results for a file by its SHA256 hash. Displays file identification (hashes, size), properties (format, packer, timestamps), classification verdict, and antivirus detection results. For archive files, it shows a summary table of all contained files.

```sh
saferwall-cli view <sha256>
```

### Download

Download a sample by its SHA256 hash, or provide a text file with one hash per line to download in batch.

```sh
# Single sample
saferwall-cli download <sha256>

# Batch from a text file
saferwall-cli download hashes.txt

# Extract from zip (password: infected) instead of keeping the .zip
saferwall-cli download -x <sha256>
```
