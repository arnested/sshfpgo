package dnsimple

import (
	"context"
	"log"
	"regexp"
	"strconv"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"

	"github.com/ShowMax/go-fqdn"
	"github.com/arnested/sshfpgo/providers"
	"github.com/arnested/sshfpgo/sshkeygen"
	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/urfave/cli"
)

func init() {
	providers.Register(dnsimpleCommand)
}

func dnsimpleCommand() cli.Command {
	defaultHostname := fqdn.Get()
	apex, err := publicsuffix.EffectiveTLDPlusOne(defaultHostname)
	if err != nil {
		log.Fatal(err)
	}
	return cli.Command{
		Name:  "dnsimple",
		Usage: "Update SSHFP DNS records for DNSimple provider",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "zone",
				Usage: "DNSimple `ZONE`",
				Value: apex,
			},
			cli.StringFlag{
				Name:   "token",
				Usage:  "DNSimple Oauth `TOKEN`",
				EnvVar: "DNSIMPLE_TOKEN",
			},
			cli.BoolFlag{
				Name:  "sandbox",
				Usage: "Run against DNSimples sandbox environment",
			},
		},
		Action: action,
	}
}

func action(c *cli.Context) error {
	if c.String("token") == "" {
		_ = cli.ShowCommandHelp(c, "dnsimple")
		return cli.NewExitError("You need to provide a DNSimple token", 0)
	}

	verbose := c.GlobalBool("verbose")
	dryRun := c.GlobalBool("dry-run")

	recordMap := sshkeygen.SshfpRecords

	oauthToken := c.String("token")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: oauthToken})
	tc := oauth2.NewClient(context.Background(), ts)

	// new client
	client := dnsimple.NewClient(tc)

	if c.Bool("sandbox") {
		client.BaseURL = "https://api.sandbox.dnsimple.com"
	} else {
		client.BaseURL = "https://api.dnsimple.com"
	}

	account, err := client.Accounts.ListAccounts(&dnsimple.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Define an account id
	accountID := strconv.FormatInt(account.Data[0].ID, 10)

	re := regexp.MustCompile("\\.?" + regexp.QuoteMeta(c.String("zone")) + "$")
	recordName := re.ReplaceAllString(c.GlobalString("hostname"), "")

	zoneResponse, err := client.Zones.ListRecords(accountID, c.String("zone"), &dnsimple.ZoneRecordListOptions{Name: recordName, Type: "SSHFP", ListOptions: dnsimple.ListOptions{}})

	if err != nil {
		log.Fatal(err)
	}

	for _, record := range zoneResponse.Data {
		// Filter by record type SSHFP. ListRecords above
		// should have done that already but didn't.
		if record.Type != "SSHFP" {
			continue
		}

		_, exists := recordMap[record.Content]

		delete(recordMap, record.Content)

		if exists {
			if verbose {
				log.Printf("Skipping up to date SSHFP record: '%v'...", record.Content)
			}

			continue
		}

		if verbose {
			log.Printf("Deletes incorrect SSHFP record: '%v'...", record.Content)
		}

		if dryRun {
			continue
		}

		_, err := client.Zones.DeleteRecord(accountID, record.ZoneID, record.ID)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, record := range recordMap {
		if verbose {
			log.Printf("Creates new SSHFP record: '%v %v %v'...", record.Algorithm, record.FingerprintType, record.Fingerprint)
		}

		if dryRun {
			continue
		}

		zoneRecord := dnsimple.ZoneRecord{
			Name:    recordName,
			Type:    "SSHFP",
			Content: record.Algorithm + " " + record.FingerprintType + " " + record.Fingerprint,
			Regions: nil,
		}

		_, err := client.Zones.CreateRecord(accountID, c.String("zone"), zoneRecord)

		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
