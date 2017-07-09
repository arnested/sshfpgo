package providers

import (
	"github.com/urfave/cli"
)

// ProviderFunc is function returning a provider CLI command.
type ProviderFunc func() cli.Command

// Providers is a map of providers.
var Providers []ProviderFunc

// Register a provider.
func Register(provider ProviderFunc) {
	Providers = append(Providers, provider)
}
