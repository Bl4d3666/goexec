# TODO

## Resolve Before Release

### TSCH

- [X] Clean up TSCH module
- [ ] Add command to tsch - update task if it already exists. See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167 (`flags` argument)
- [ ] Add more trigger types

### SCMR

- [X] Clean up SCMR module 
- [X] add dynamic string binding support
- [X] general clean up. Use TSCH & WMI as reference
- [ ] Fix SCMR `change` method so that dependencies field isn't permanently overwritten

### DCOM

- [X] Add DCOM module
- [X] MMC20.Application method

### WMI

- [X] Add WMI module
- [X] Clean up WMI module
- [ ] WMI `reg` subcommand - read & edit the registry
- [ ] File transfer functionality

### Other

- [X] Add proxy support - see https://github.com/oiweiwei/go-msrpc/issues/21
- [ ] `--unsafe` option - allow unsafe OPSEC (i.e. fetching execution output via file write/read)
- [ ] Descriptions for all modules and methods
- [ ] Add SMB file transfer interface
- [ ] README

## Resolve Eventually

- [ ] Add Go tests
- [ ] ability to specify multiple targets
- [ ] Standardize modules to interface for future use

### WinRM

- [ ] Add basic WinRM module - https://github.com/bryanmcnulty/winrm
    - [ ] File transfer functionality
    - [ ] Shell functionality