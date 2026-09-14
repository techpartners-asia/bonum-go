package bonum

import "github.com/techpartners-asia/bonum-go/internal/gateway/application"

// SandboxService groups helpers Bonum only permits outside production. Keeping them off the
// aggregate services stops them leaking into production code paths.
type SandboxService = application.Sandbox
