# saferwall-cli

A CLI tool to interact with Saferwall.com to download samples, scan or re-scan new samples.

## Install

You can either download pre-built binaries or build the tool yourself.

```sh
go install github.com/saferwall/saferwall-cli
```

## Usage

To use the CLI tool you need a [Saferwall](https://saferwall.com) account in order to authenticate yourself.

The CLI tool reads your username and password from a local `.env` file or from your OS environment variable.

```sh
SAFERWALL_AUTH_USERNAME=username
SAFERWALL_AUTH_PASSWORD=password
```


```sh
saferwall-cli - Saferwall command line tool

	███████╗ █████╗ ███████╗███████╗██████╗ ██╗    ██╗ █████╗ ██╗     ██╗          ██████╗██╗     ██╗
	██╔════╝██╔══██╗██╔════╝██╔════╝██╔══██╗██║    ██║██╔══██╗██║     ██║         ██╔════╝██║     ██║
	███████╗███████║█████╗  █████╗  ██████╔╝██║ █╗ ██║███████║██║     ██║         ██║     ██║     ██║
	╚════██║██╔══██║██╔══╝  ██╔══╝  ██╔══██╗██║███╗██║██╔══██║██║     ██║         ██║     ██║     ██║
	███████║██║  ██║██║     ███████╗██║  ██║╚███╔███╔╝██║  ██║███████╗███████╗    ╚██████╗███████╗██║
	╚══════╝╚═╝  ╚═╝╚═╝     ╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝     ╚═════╝╚══════╝╚═╝


saferwall-cli allows you to interact with the saferwall API. You can
upload, scan samples from your drive, or download samples.
For more details see the github repo at https://github.com/saferwall

Usage:
  saferwall-cli [flags]
  saferwall-cli [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  download    Download a sample given its SHA256 hash.
  gen         Generate malware souk markdown for the entire corpus
  help        Help about any command
  scan        Submit a scan request of a file using its hash
  version     Version number

Flags:
  -h, --help   help for saferwall-cli

Use "saferwall-cli [command] --help" for more information about a command.
```


### Download

You can download files using their SHA256 hash and specify an output folder, you can also download a batch of samples by copying their SHA256 hash to the clipboard.

```sh
cli download --hash 0001cb47c8277e44a09543291d95559886b9c2da195bd78fdf108775ac91ac53
```

### Scan

You can scan or rescan files using the scan command.

```sh
cli scan -p /samples/0001cb47c8277e44a09543291d95559886b9c2da195bd78fdf108775ac91ac53
```
