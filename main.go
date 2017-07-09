package main

import (
	"log"
	"os"

	"github.com/ShowMax/go-fqdn"
	"github.com/arnested/sshfpgo/providers"
	// Blank import to let dnsimple register it self.
	_ "github.com/arnested/sshfpgo/providers/dnsimple"
	"github.com/arnested/sshfpgo/sshkeygen"

	"github.com/urfave/cli"
)

// Version string to be set at compile time via command line (-ldflags "-X main.VersionString=1.2.3")
var (
	VersionString string
)

var cliCommands []cli.Command

func main() {
	defaultHostname := fqdn.Get()

	app := cli.NewApp()
	app.Name = "sshfpgo"
	app.Usage = "Update SSHFP DNS records"
	app.EnableBashCompletion = true
	app.Authors = []cli.Author{
		{
			Name:  "Arne Jørgensen",
			Email: "arne@arnested.dk",
		},
	}
	app.Version = VersionString

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Verbose output",
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "Do no updates",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "The `HOSTNAME` to update records for",
			Value: defaultHostname,
		},
	}

	app.Before = func(c *cli.Context) error {
		sshkeygen.Collect(c.GlobalString("hostname"))
		if c.Bool("verbose") && len(sshkeygen.SshfpRecords) <= 0 {
			log.Printf("No SSH host keys found.\n")
		}
		return nil
	}

	for _, provider := range providers.Providers {
		app.Commands = append(app.Commands, provider())
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
