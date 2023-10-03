# Shortcut signing server

A simple server for signing iOS/macOS shortcuts

## How to use
- Clone repo and run: `go run . serve <address:port>`  
  - or run `go install .` then run `shortcut-signing-server <address:post>`
- Or install directly via go, run: `go install github.com/scaxyz/shortcut-signing-server` then `shortcut-signing-server <address:post>`

## Usage
```text
NAME:
   shortcut-signing-server - A simple server for signing iOS/macOS shortcuts over http

USAGE:
   shortcut-signing-server [global options] command [command options] [arguments...]

COMMANDS:
   serve    Start the server
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value  Path to the yaml config file
   --help, -h      show help

   LOGGING

   --debug             Set log level to debug (default: false)
   --log-file value    Log file path
   --log-format value  Available log formats: text, json (default: "text")
   --log-level value   Available log levels: debug, info, warn, error, fatal, panic, trace (default: "info")
   --quiet, -q         Disable logging to stdout (default: false)
```
### serve
```text
NAME:
   shortcut-signing-server serve - Start the server

USAGE:
   shortcut-signing-server serve [command options] [arguments...]

OPTIONS:
   --config value  Path to the yaml config file

   LOGGING

   --debug             Set log level to debug (default: false)
   --log-file value    Log file path
   --log-format value  Available log formats: text, json (default: "text")
   --log-level value   Available log levels: debug, info, warn, error, fatal, panic, trace (default: "info")
   --quiet, -q         Disable logging to stdout (default: false)

   SERVER

   --max-concurrent-jobs value   Maximum number of concurrent signing jobs (default: 0)
   --real-error-responses, --re  Return real errors instead of only HTTP codes names in the response (default: false)
   --templates value             Folder containing custom html templates (currently only: 'form.html' is used)
   --tls                         Enable TLS (default: false)
   --tls-cert value              Path to the tls cert file
   --tls-key value               Path to the tls cert key
```
## Config file
A yaml file which uses the name of the cli flags as keys
e.g.
```yaml
log-level: debug
log-format: text
log-file: ./testdata/test.log
tls: false
real-error-responses: true
```

### API
View [editor.swagger.io](https://editor.swagger.io/?url=https://raw.githubusercontent.com/scaxyz/shortcut-signing-server/main/openapi.yml)

## Docker
View [docker-osx.md](./docker-osx.md) for an guide how to set up docker with docker-osx to run this program on most non mac machines

## Errors
### `signal: abort trap`
Make sure are logged in to your macOS via the UI/VNC
