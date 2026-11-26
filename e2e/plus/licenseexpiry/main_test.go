package main

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2028, 12, 1, 12, 0, 0, 0, time.UTC)

func serverInfoJSON(licenseStatus string) string {
	return `{"server-id":"s","version":"0.15.1","license-status":` + licenseStatus + `}`
}

func TestNotice_FewerThan30DaysLeft_ReturnsNotice(t *testing.T) {
	in := serverInfoJSON(`{"license-type":"Vakamo-Enterprise","valid":true,
		"expiration":"2028-12-20T23:59:59Z","customer":"Example Customer",
		"license-id":"lic-secret-id","issuer":"Vakamo","audience":"aud-x"}`)

	n, err := notice(strings.NewReader(in), now)
	if err != nil {
		t.Fatalf("notice: %v", err)
	}
	if n == nil {
		t.Fatal("expected a notice with 19 days left")
	}
	if n.DaysLeft != 19 {
		t.Errorf("days left: got %d, want 19", n.DaysLeft)
	}
	body := n.Body()
	for _, want := range []string{"2028-12-20", "19", "Vakamo-Enterprise", "Example Customer", "renew", "LAKEKEEPER_PLUS_LICENSE_KEY"} {
		if !strings.Contains(body, want) {
			t.Errorf("body misses %q:\n%s", want, body)
		}
	}
	for _, leak := range []string{"lic-secret-id", "aud-x", "server-id", "linear"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(leak)) {
			t.Errorf("body leaks %q:\n%s", leak, body)
		}
	}
}

func TestNotice_MoreThan30DaysLeft_ReturnsNil(t *testing.T) {
	in := serverInfoJSON(`{"license-type":"Vakamo-Enterprise","valid":true,"expiration":"2029-01-30T23:59:59Z"}`)

	n, err := notice(strings.NewReader(in), now)
	if err != nil {
		t.Fatalf("notice: %v", err)
	}
	if n != nil {
		t.Errorf("expected no notice with 60 days left, got %+v", n)
	}
}

func TestNotice_Exactly30DaysLeft_ReturnsNil(t *testing.T) {
	in := serverInfoJSON(`{"license-type":"t","valid":true,"expiration":"2028-12-31T12:00:00Z"}`)

	n, err := notice(strings.NewReader(in), now)
	if err != nil {
		t.Fatalf("notice: %v", err)
	}
	if n != nil {
		t.Errorf("expected no notice with exactly 30 days left, got %+v", n)
	}
}

func TestNotice_AlreadyExpired_ReturnsNotice(t *testing.T) {
	in := serverInfoJSON(`{"license-type":"t","valid":false,"expiration":"2028-11-28T12:00:00Z"}`)

	n, err := notice(strings.NewReader(in), now)
	if err != nil {
		t.Fatalf("notice: %v", err)
	}
	if n == nil {
		t.Fatal("expected a notice for an expired license")
	}
	if n.DaysLeft != -3 {
		t.Errorf("days left: got %d, want -3", n.DaysLeft)
	}
	if !strings.Contains(n.Body(), "expired") {
		t.Errorf("body should say the license expired:\n%s", n.Body())
	}
}

func TestNotice_NoExpiration_ReturnsNil(t *testing.T) {
	for name, status := range map[string]string{
		"null expiration":    `{"license-type":"t","valid":true,"expiration":null}`,
		"missing expiration": `{"license-type":"t","valid":true}`,
		"no license status":  `null`,
	} {
		t.Run(name, func(t *testing.T) {
			n, err := notice(strings.NewReader(serverInfoJSON(status)), now)
			if err != nil {
				t.Fatalf("notice: %v", err)
			}
			if n != nil {
				t.Errorf("expected no notice, got %+v", n)
			}
		})
	}
}

func TestNotice_MalformedInput_ReturnsError(t *testing.T) {
	if _, err := notice(strings.NewReader("not json"), now); err == nil {
		t.Error("expected an error for malformed input")
	}
}
