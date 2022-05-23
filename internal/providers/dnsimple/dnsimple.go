package dnsimple

import (
	"context"
	"log"
	"regexp"
	"strconv"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/oauth2"

	"github.com/Showmax/go-fqdn"
	"github.com/arnested/sshfpgo/internal/providers"
	"github.com/arnested/sshfpgo/internal/sshkeygen"
	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/urfave/cli/v2"
)

//nolint:gochecknoinits // needs refactoring.
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
			&cli.StringFlag{
				Name:  "zone",
				Usage: "DNSimple `ZONE`",
				Value: apex,
			},
			&cli.StringFlag{
				Name:    "token",
				Usage:   "DNSimple Oauth `TOKEN`",
				EnvVars: []string{"DNSIMPLE_TOKEN"},
			},
			&cli.BoolFlag{
				Name:  "sandbox",
				Usage: "Run against DNSimples sandbox environment",
			},
		},
		Action: action,
	}
}

//nolint:funlen,cyclop // needs refactoring.
func action(cliCtx *cli.Context) error {
	if cliCtx.String("token") == "" {
		_ = cli.ShowCommandHelp(cliCtx, "dnsimple")

		return cli.Exit("You need to provide a DNSimple token", 0)
	}

	verbose := cliCtx.Bool("verbose")
	dryRun := cliCtx.Bool("dry-run")

	recordMap := sshkeygen.SshfpRecords

	ctx := context.Background()

	oauthToken := cliCtx.String("token")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: oauthToken})
	tc := oauth2.NewClient(ctx, ts)

	// new client
	client := dnsimple.NewClient(tc)

	if cliCtx.Bool("sandbox") {
		client.BaseURL = "https://api.sandbox.dnsimple.com"
	} else {
		client.BaseURL = "https://api.dnsimple.com"
	}

	account, err := client.Accounts.ListAccounts(ctx, &dnsimple.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Define an account id
	//nolint:gomnd
	accountID := strconv.FormatInt(account.Data[0].ID, 10)

	re := regexp.MustCompile("\\.?" + regexp.QuoteMeta(cliCtx.String("zone")) + "$")
	recordName := re.ReplaceAllString(cliCtx.String("hostname"), "")

	recordType := "SSHFP"

	zoneResponse, err := client.Zones.ListRecords(
		ctx,
		accountID,
		cliCtx.String("zone"),
		&dnsimple.ZoneRecordListOptions{Name: &recordName, Type: &recordType, ListOptions: dnsimple.ListOptions{}},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, record := range zoneResponse.Data {
		// Filter by record type SSHFP. ListRecords above
		// should have done that already but didn't.
		if record.Type != recordType {
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

		_, err := client.Zones.DeleteRecord(ctx, accountID, record.ZoneID, record.ID)
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

		zoneRecord := dnsimple.ZoneRecordAttributes{
			Name:    &recordName,
			Type:    recordType,
			Content: record.Algorithm + " " + record.FingerprintType + " " + record.Fingerprint,
			Regions: nil,
		}

		_, err := client.Zones.CreateRecord(ctx, accountID, cliCtx.String("zone"), zoneRecord)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
