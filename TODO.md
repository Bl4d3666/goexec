# TODO

## Resolve Before Release

### TSCH

- [X] Clean up TSCH module
- [X] Session hijacking
- [X] Generate random name/path
- [X] Output
- [ ] Add more trigger types
- [ ] Add command to tsch - update task if it already exists. See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167 (`flags` argument)

### SCMR

- [X] Clean up SCMR module 
- [X] add dynamic string binding support
- [X] general clean up. Use TSCH & WMI as reference
- [X] Output
- [ ] Fix SCMR `change` method so that dependencies field isn't permanently overwritten

### DCOM

- [X] Add DCOM module
- [X] MMC20.Application method
- [X] Output
- [ ] ShellWindows & ShellBrowserWindow

### WMI

- [X] Add WMI module
- [X] Clean up WMI module
- [X] Output
- [ ] WMI `reg` subcommand - read & edit the registry
- [ ] File transfer functionality

### Other

- [X] Add proxy support - see https://github.com/oiweiwei/go-msrpc/issues/21
- [ ] Descriptions for all modules and methods
- [ ] Add SMB file transfer interface
- [ ] README

#### CLI

- [ ] `--full-help`/`-H`

## Resolve Eventually

- [ ] `--shell` option
- [ ] Add Go tests
- [ ] ability to specify multiple targets

### WinRM

- [ ] Add basic WinRM module - https://github.com/bryanmcnulty/winrm
    - [ ] File transfer functionality
    - [ ] Shell functionality