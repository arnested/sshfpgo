package main

import (
	"log"
	"os"

	"github.com/Showmax/go-fqdn"
	"github.com/arnested/sshfpgo/internal/providers"

	// Blank import to let dnsimple register it self.
	_ "github.com/arnested/sshfpgo/internal/providers/dnsimple"
	"github.com/arnested/sshfpgo/internal/sshkeygen"

	"github.com/urfave/cli/v2"
)

// Version string to be set at compile time via command line (-ldflags "-X main.version=1.2.3").
var (
	version string
)

func main() {
	defaultHostname := fqdn.Get()

	app := cli.NewApp()
	app.Name = "sshfpgo"
	app.Usage = "Update SSHFP DNS records"
	app.EnableBashCompletion = true
	app.Authors = []*cli.Author{
		{
			Name:  "Arne Jørgensen",
			Email: "arne@arnested.dk",
		},
	}
	app.Version = version

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Verbose output",
		},
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Do no updates",
		},
		&cli.StringFlag{
			Name:  "hostname",
			Usage: "The `HOSTNAME` to update records for",
			Value: defaultHostname,
		},
	}

	app.Before = func(c *cli.Context) error {
		sshkeygen.Collect(c.String("hostname"))

		if c.Bool("verbose") && len(sshkeygen.SshfpRecords) == 0 {
			log.Printf("No SSH host keys found.\n")
		}

		return nil
	}

	for _, provider := range providers.Providers {
		provider := provider()
		app.Commands = append(app.Commands, &provider)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
