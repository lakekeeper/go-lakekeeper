// Command licenseexpiry reads a Lakekeeper+ ServerInfo document (JSON, as
// `lkctl server info --output json` prints it) on stdin and decides whether
// the CI test license is close to expiry. When fewer than 30 days remain, it
// prints a GitHub issue body to stdout; otherwise it prints nothing. It never
// echoes its input.
//
// The body holds only the expiry date, the days left, the license type, the
// customer name and the renewal note: the issue is public, so it must never
// carry the license key, the license ID or other ServerInfo fields.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

// warnDays is the number of days before expiry at which a notice is raised.
const warnDays = 30

// serverInfo is the subset of ServerInfo this command reads. Decoding only
// these fields keeps the rest of the document out of memory and output.
type serverInfo struct {
	LicenseStatus *struct {
		LicenseType string     `json:"license-type"`
		Customer    *string    `json:"customer"`
		Expiration  *time.Time `json:"expiration"`
	} `json:"license-status"`
}

// Notice describes a license that expires within warnDays.
type Notice struct {
	Expiration  time.Time
	DaysLeft    int
	LicenseType string
	Customer    string
}

// notice returns a Notice when the license in the ServerInfo read from r
// expires in fewer than warnDays days from now, and nil when it does not or
// has no expiration.
func notice(r io.Reader, now time.Time) (*Notice, error) {
	var info serverInfo
	if err := json.NewDecoder(r).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode server info: %w", err)
	}
	ls := info.LicenseStatus
	if ls == nil || ls.Expiration == nil {
		return nil, nil
	}
	daysLeft := int(math.Floor(ls.Expiration.Sub(now).Hours() / 24))
	if daysLeft >= warnDays {
		return nil, nil
	}
	n := &Notice{
		Expiration:  ls.Expiration.UTC(),
		DaysLeft:    daysLeft,
		LicenseType: ls.LicenseType,
	}
	if ls.Customer != nil {
		n.Customer = *ls.Customer
	}
	return n, nil
}

// Body is the GitHub issue body for the notice.
func (n *Notice) Body() string {
	var b strings.Builder
	if n.DaysLeft < 0 {
		fmt.Fprintf(&b, "The Lakekeeper+ test license used by CI expired %d days ago.\n\n", -n.DaysLeft)
	} else {
		fmt.Fprintf(&b, "The Lakekeeper+ test license used by CI expires in %d days.\n\n", n.DaysLeft)
	}
	fmt.Fprintf(&b, "- Expiry date: %s\n", n.Expiration.Format(time.DateOnly))
	fmt.Fprintf(&b, "- Days left: %d\n", n.DaysLeft)
	fmt.Fprintf(&b, "- License type: %s\n", n.LicenseType)
	customer := n.Customer
	if customer == "" {
		customer = "(not set)"
	}
	fmt.Fprintf(&b, "- Customer: %s\n\n", customer)
	b.WriteString("A maintainer must renew the license and update the `LAKEKEEPER_PLUS_LICENSE_KEY` " +
		"secret of the `lakekeeper-plus` environment.\n")
	return b.String()
}

func main() {
	n, err := notice(os.Stdin, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "licenseexpiry:", err)
		os.Exit(1)
	}
	if n == nil {
		return
	}
	fmt.Print(n.Body())
}
