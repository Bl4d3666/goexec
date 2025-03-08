# TODO

## Resolve Before Release

### Higher Priority
- [X] Add WMI module
- [X] Clean up TSCH module

- [ ] Clean up SCMR module 
  - [ ] add dynamic string binding support
  - [ ] general clean up. Use TSCH & WMI as reference

- [ ] Add DCOM module
  - [ ] MMC20.Application method

- [ ] Add psexec module (RemComSvc)
  - [ ] Add support for dynamic service executable (of course)

### Other
- [ ] Fix SCMR `change` method so that dependencies field isn't permanently overwritten
- [ ] Add `delete` command to all modules that may involve cleanup - use `tsch delete` for reference
- [ ] Standardize modules to interface for future use
- [ ] Add command to tsch - update task if it already exists. See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167 (`flags` argument)

### Testing
- [ ] Testing against different Windows machines & versions
- [ ] Testing from Windows (compile to PE)

## Resolve Eventually

### Higher Priority
- [ ] Add dcom module

### Lower Priority
- [ ] `--ctf` option - allow unsafe OPSEC (i.e. fetching execution output via file write/read)
- [ ] ability to specify multiple targets