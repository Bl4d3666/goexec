# Testing

## Known Issues

| Issue                                           | Fixed | Fix                                                                                  |
|:------------------------------------------------|:------|:-------------------------------------------------------------------------------------|
| NTLMv2 authentication broken when using NT hash | yes   | https://github.com/oiweiwei/go-msrpc/commit/e65ccab483f45ebf545fd1122cb405931cc3c886 |
| Kerberos authentication broken for DCOM module  | no    |                                                                                      |

## Windows Server 2025

### DCOM

- [X] `goexec dcom mmc $target -u "$auth_user" -H "$auth_nt" -c "$cmd" --debug --no-epm`

## Windows 11 Pro