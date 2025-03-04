package dcerpc

import "github.com/oiweiwei/go-msrpc/smb2"

const (
	SmbDefaultPort = 445
)

type SmbConfig struct {
	Port         uint16
	FullSecurity bool
	ForceDialect smb2.Dialect
}
