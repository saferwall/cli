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
  download    Download a sample (and its artifacts)
  souk        Populate malware-souk database
  version     Version number
```

### Scan

Upload and scan files. Supports scanning a single file or an entire directory.

```sh
saferwall-cli scan -p /path/to/sample
```

### Rescan

Rescan an existing file by its SHA256 hash, or rescan a batch of hashes from a text file.

```sh
saferwall-cli rescan <sha256>
```

### Download

Download a sample by its SHA256 hash, or provide a text file with one hash per line to download in batch.

```sh
# Single sample
saferwall-cli download <sha256>

# Batch from a text file
saferwall-cli download hashes.txt
```
