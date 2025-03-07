# TODO

## Resolve Before Release

### Higher Priority
- [ ] Add WMI module
- [ ] Add psexec module (RemComSvc)
- [ ] Testing on different Windows versions

### Other
- [ ] Fix SCMR `change` method so that dependencies field isn't permanently overwritten
- [ ] Add `delete` command to all modules that may involve cleanup - use `tsch delete` for reference

## Resolve Eventually

### Higher Priority
- [ ] Add dcom module
- [ ] Add command to tsch - update task if it already exists. See https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167 (`flags` argument)

### Lower Priority
- [ ] `--ctf` option - allow unsafe OPSEC (i.e. fetching execution output via file write/read)
- [ ] ability to specify multiple targets