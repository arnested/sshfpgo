package sshkeygen

import (
	"log"
	"os/exec"
	"strings"
)

// SshfpRecord is an SSHFP DNS Record.
type SshfpRecord struct {
	Name            string
	Algorithm       string
	FingerprintType string
	Fingerprint     string
	TTL             string
}

// RecordMap is a map of SSHFP Records indexed by algorithm and
// fingerprinttype.
type RecordMap map[string]SshfpRecord

// SshfpRecords are the collected records.
var SshfpRecords RecordMap //nolint:gochecknoglobals // needs refactoring.

// Collect ssh host key fingerprints.
func Collect(hostname string) {
	output, err := exec.Command("ssh-keygen", "-r", hostname).Output()
	if err != nil {
		log.Fatal(err)
	}

	keys := strings.Split(strings.TrimSpace(string(output)), "\n")
	SshfpRecords = RecordMap{}

	for _, key := range keys {
		var sshfpRecord SshfpRecord

		//nolint:gomnd
		s := strings.SplitN(key, " ", 6)
		//nolint:gomnd
		if len(s) != 6 {
			log.Printf("unexpected number of fields in ssh-keygen output: %q", key)

			continue
		}

		sshfpRecord.Name, sshfpRecord.Algorithm, sshfpRecord.FingerprintType, sshfpRecord.Fingerprint = s[0], s[3], s[4], s[5]
		SshfpRecords[sshfpRecord.Algorithm+" "+sshfpRecord.FingerprintType+" "+sshfpRecord.Fingerprint] = sshfpRecord
	}
}
