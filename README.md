# saferwall-cli

A CLI tool to interact with Saferwall.com to download samples, scan or re-scan new samples.

## Install

You can either download pre-built binaries or build the tool yourself.

```sh
go install github.com/saferwall/saferwall-cli
```


## Usage

To use the CLI tool you need a [Saferwall](https://saferwall.com) account in order to authenticate yourself.

Use the `config.example.toml` as a reference to reference your credendials. The file should be located in:
``~/.config/saferwall/config.toml`:

```toml
[credentials]
# The URL used to interact with saferwall APIs.
url = "https://api.saferwall.com"
# The user name you choose when you sign-up for saferwall.com.
username = "YourUsername"
# The password you choose when you sign-up for saferwall.com.
password = "YourPassword"
```

The CLI app can also be used to interfact with a self-hosted deployment.

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
  completion  Generate the autocompletion script for the specified shell
  delete      Delete a sample(s) given its SHA256 hash.
  download    Download a sample(s) or a behavior report
  help        Help about any command
  list        List users or files.
  rescan      Rescan an exiting file using its hash
  scan        Submit a scan request of a file using its hash
  souk        Populate malware-souk database.
  upload      Upload samples directly to object storage.
  version     Version number

Flags:
  -h, --help   help for saferwall-cli
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
