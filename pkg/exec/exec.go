package exec

import (
	"context"
	"github.com/bryanmcnulty/adauth"
)

type ExecutionConfig struct {
	ExecutableName string // ExecutableName represents the name of the executable; i.e. "notepad.exe", "calc"
	ExecutablePath string // ExecutablePath represents the full path to the executable; i.e. `C:\Windows\explorer.exe`
	ExecutableArgs string // ExecutableArgs represents the arguments to be passed to the executable during execution; i.e. "/C whoami"

	ExecutionMethod       string // ExecutionMethod represents the specific execution strategy used by the module.
	ExecutionMethodConfig interface{}
	ExecutionOutput       string      // not implemented
	ExecutionOutputConfig interface{} // not implemented
}

type ShellConfig struct {
	ShellName string // ShellName specifies the name of the shell executable; i.e. "cmd.exe", "powershell"
	ShellPath string // ShellPath is the full Windows path to the shell executable; i.e. `C:\Windows\System32\cmd.exe`
}

type Executor interface {
	// Exec performs a single execution task without the need to call Init.
	Exec(ctx context.Context, creds *adauth.Credential, target *adauth.Target, config *ExecutionConfig)

	// Init assigns the provided TODO
	//Init(ctx context.Context, creds *adauth.Credential, target *adauth.Target)
	//Shell(ctx context.Context, input chan *ExecutionConfig, output chan []byte)
}

func (cfg *ExecutionConfig) GetRawCommand() string {
	if cfg.ExecutableArgs != "" {
		return cfg.ExecutablePath + " " + cfg.ExecutableArgs
	}
	return cfg.ExecutablePath
}
