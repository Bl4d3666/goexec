# TODO

## TSCH

- [X] Clean up TSCH module
- [X] Session hijacking
- [X] Generate random name/path
- [X] Output
- [X] Add `tsch change`
- [ ] Serialize XML with default indent level

## SCMR

- [X] Clean up SCMR module 
- [X] add dynamic string binding support
- [X] general cleanup. Use TSCH & WMI as reference
- [ ] Output

## DCOM

- [X] Add DCOM module
- [X] MMC20.Application method
- [X] Output

## WMI

- [X] Add WMI module
- [X] Clean up WMI module
- [X] Output
- [ ] WMI `reg` subcommand - read & edit the registry
- [ ] File transfer functionality

## Other

- [X] Add proxy support - see https://github.com/oiweiwei/go-msrpc/issues/21
- [ ] Descriptions for all modules and methods
- [ ] Add SMB file transfer interface
- [ ] README

## Bug Fixes

- [ ] Fix proxy - EPM, SMB don't use the proxy dialer
- [ ] Fix SMB transport for SCMR module - `rpc_s_cannot_support: The requested operation is not supported.`
- [ ] Fix SCMR `change` method so that dependencies field isn't permanently overwritten

## Lower Priority

- [ ] `--shell` option
- [ ] Add Go tests
- [ ] ability to specify multiple targets

### TSCH

- [ ] Add more trigger types

### SCMR

- [ ] `psexec` with PsExeSVC.exe AND NOT Impacket's RemCom build - https://sensepost.com/blog/2025/psexecing-the-right-way-and-why-zero-trust-is-mandatory/

### DCOM

- [ ] ShellWindows & ShellBrowserWindow

### WinRM

- [ ] Add basic WinRM module - https://github.com/bryanmcnulty/winrm
    - [ ] File transfer functionality
    - [ ] Shell functionality