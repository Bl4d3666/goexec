# Goexec - Remote Execution Multitool

![goexec](https://github.com/user-attachments/assets/4adc2087-3edf-4221-a310-824f9a185146)

Goexec is a new take on some of the methods used to gain remote execution on Windows devices. Goexec implements a number of largely unrealized execution methods and provides significant OPSEC improvements overall.

The original post about Goexec v0.1.0 can be found [here](https://www.falconops.com/blog/introducing-goexec)

## Usage

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

## Acknowledgements

- [@oiweiwei](https://github.com/oiweiwei) for the wonderful [go-msrpc](https://github.com/oiweiwei/go-msrpc) module
- [@RedTeamPentesting](https://github.com/RedTeamPentesting) and [Erik Geiser](https://github.com/rtpt-erikgeiser) for the [adauth](https://github.com/RedTeamPentesting/adauth) module
- The developers and contributors of [Impacket](https://github.com/fortra/impacket) for the inspiration and technical reference
