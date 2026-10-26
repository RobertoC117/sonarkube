package probe

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTLSCertificateIsValid(t *testing.T) {

	testCases := []struct {
		name          string
		cert          x509.Certificate
		currentDate   time.Time
		isValidWanted bool
	}{
		{
			name: "Certificate is valid",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Now(),
			isValidWanted: true,
		},
		{
			name: "Certificate is expired",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(-time.Hour * time.Duration(24)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Now(),
			isValidWanted: false,
		},
		{
			name: "Certificate is not valid yet",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(time.Hour * time.Duration(24)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Now(),
			isValidWanted: false,
		},
		{
			name: "Certificate with notBefore as Zero value",
			cert: x509.Certificate{
				Issuer:   pkix.Name{CommonName: "mTLS"},
				NotAfter: time.Now().Add(time.Hour * time.Duration(48)),
				DNSNames: []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Now(),
			isValidWanted: true,
		},
		{
			name: "Certificate with border notBefore",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC),
			isValidWanted: true,
		},
		{
			name: "Certificate with border notAfter",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:   time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC),
			isValidWanted: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			certDetails := buildCertDetails(tls.VersionTLS12, &tt.cert, tt.currentDate)

			if certDetails.IsValid != tt.isValidWanted {
				t.Errorf("Certificate valid value expected %v, got %v", tt.isValidWanted, certDetails.IsValid)
			}
		})
	}
}

func TestTLSVersion(t *testing.T) {

	testCases := []struct {
		name          string
		cert          x509.Certificate
		version       uint16
		versionWanted string
	}{
		{

			name: "TLS version 1.0",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			version:       tls.VersionTLS10,
			versionWanted: "TLS 1.0",
		},
		{
			name: "TLS version 1.1",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			version:       tls.VersionTLS11,
			versionWanted: "TLS 1.1",
		},
		{
			name: "TLS version 1.2",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			version:       tls.VersionTLS12,
			versionWanted: "TLS 1.2",
		},
		{
			name: "TLS version 1.3",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			version:       tls.VersionTLS13,
			versionWanted: "TLS 1.3",
		},
		{
			name: "TLS version unknown",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(time.Hour * time.Duration(48)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			version:       0x0001,
			versionWanted: "unknown (0x0001)",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			currentDate := time.Now()
			certDetails := buildCertDetails(tt.version, &tt.cert, currentDate)
			if certDetails.Version != tt.versionWanted {
				t.Errorf("Version expected: %v, got: %v", tt.versionWanted, certDetails.Version)
			}
		})
	}
}

func TestDaysToExpiry(t *testing.T) {

	testCases := []struct {
		name               string
		cert               x509.Certificate
		currentDate        time.Time
		daysToExpiryWanted int
	}{
		{
			name: "Certificate is valid",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Date(2026, time.October, 27, 11, 0, 0, 0, time.UTC),
				NotBefore: time.Date(2025, time.October, 27, 11, 0, 0, 0, time.UTC),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:        time.Date(2026, time.October, 25, 11, 0, 0, 0, time.UTC),
			daysToExpiryWanted: 2,
		},
		{
			// 2 days and 23 hours to expiry: floors to 2 (round/ceil would give 3).
			name: "Certificate is valid (fraction floors down)",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Date(2026, time.October, 28, 10, 0, 0, 0, time.UTC),
				NotBefore: time.Date(2025, time.October, 27, 11, 0, 0, 0, time.UTC),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:        time.Date(2026, time.October, 25, 11, 0, 0, 0, time.UTC),
			daysToExpiryWanted: 2,
		},
		{
			name: "Certificate is expired",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Now().Add(-time.Hour * time.Duration(24)),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:        time.Now(),
			daysToExpiryWanted: 0,
		},
		{
			name: "Certificate is not valid yet",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Date(2026, time.October, 27, 11, 0, 0, 0, time.UTC),
				NotBefore: time.Date(2025, time.October, 27, 11, 0, 0, 0, time.UTC),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:        time.Date(2025, time.October, 25, 11, 0, 0, 0, time.UTC),
			daysToExpiryWanted: 367,
		},
		{
			name: "Certificate with border notAfter",
			cert: x509.Certificate{
				Issuer:    pkix.Name{CommonName: "mTLS"},
				NotAfter:  time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC),
				NotBefore: time.Now().Add(-time.Hour * time.Duration(48)),
				DNSNames:  []string{"www.rincondelvago.com"},
			},
			currentDate:        time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC),
			daysToExpiryWanted: 0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			certDetails := buildCertDetails(tls.VersionTLS12, &tt.cert, tt.currentDate)

			if certDetails.DaysToExpiry != tt.daysToExpiryWanted {
				t.Errorf("Certificate days to expiry expected %v, got %v", tt.daysToExpiryWanted, certDetails.DaysToExpiry)
			}
		})
	}
}

func TestCertFields(t *testing.T) {

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name         string
		cert         x509.Certificate
		issuerWanted string
		namesWanted  []string
	}{
		{
			name: "Single name",
			cert: x509.Certificate{
				Issuer:   pkix.Name{CommonName: "Let's Encrypt"},
				NotAfter: now.Add(time.Hour * time.Duration(24)),
				DNSNames: []string{"example.com"},
			},
			issuerWanted: "Let's Encrypt",
			namesWanted:  []string{"example.com"},
		},
		{
			name: "Multiple SANs with wildcard",
			cert: x509.Certificate{
				Issuer:   pkix.Name{CommonName: "R3"},
				NotAfter: now.Add(time.Hour * time.Duration(24)),
				DNSNames: []string{"example.com", "www.example.com", "*.example.com"},
			},
			issuerWanted: "R3",
			namesWanted:  []string{"example.com", "www.example.com", "*.example.com"},
		},
		{
			name: "No names",
			cert: x509.Certificate{
				Issuer:   pkix.Name{CommonName: "Internal CA"},
				NotAfter: now.Add(time.Hour * time.Duration(24)),
				DNSNames: nil,
			},
			issuerWanted: "Internal CA",
			namesWanted:  nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			certDetails := buildCertDetails(tls.VersionTLS12, &tt.cert, now)

			if certDetails.Issuer != tt.issuerWanted {
				t.Errorf("Issuer expected %q, got %q", tt.issuerWanted, certDetails.Issuer)
			}
			if !reflect.DeepEqual(certDetails.Names, tt.namesWanted) {
				t.Errorf("Names expected %v, got %v", tt.namesWanted, certDetails.Names)
			}
		})
	}
}

func TestRenderTable(t *testing.T) {

	details := TLSCertificateDetails{
		Issuer:         "R3",
		Version:        "TLS 1.3",
		ExpirationDate: time.Date(2026, time.December, 21, 18, 38, 32, 0, time.UTC),
		IsValid:        true,
		Names:          []string{"example.com", "www.example.com"},
		DaysToExpiry:   56,
	}

	out, err := details.RenderTable()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wants := []string{
		"Issuer: R3",
		"TLS version: TLS 1.3",
		"Valid: true",
		"Days to expiry: 56",
		"example.com, www.example.com",
	}

	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("RenderTable output missing %q\ngot:\n%s", w, out)
		}
	}
}
