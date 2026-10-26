package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"
)

type TLSCertificateDetails struct {
	Issuer         string    `json:"issuer"`
	Version        string    `json:"version"`
	ExpirationDate time.Time `json:"expiration_date"`
	IsValid        bool      `json:"valid"`
	Names          []string  `json:"names"`
	DaysToExpiry   int       `json:"days_to_expiry"`
}

func (d TLSCertificateDetails) RenderTable() (string, error) {
	s := fmt.Sprintf("Issuer: %s\n", d.Issuer)
	s += fmt.Sprintf("TLS version: %s\n", d.Version)
	s += fmt.Sprintf("Valid: %t\n", d.IsValid)
	s += fmt.Sprintf("Expires: %s\n", d.ExpirationDate.Format(time.RFC3339))
	s += fmt.Sprintf("Days to expiry: %d\n", d.DaysToExpiry)
	s += fmt.Sprintf("Names: %s\n", strings.Join(d.Names, ", "))
	return s, nil
}

func isValid(currentDate time.Time, notAfter time.Time, notBefore time.Time) bool {

	isNotBeforeDateReached := true

	if !notBefore.IsZero() {
		isNotBeforeDateReached = !notBefore.After(currentDate)
	}

	return isNotBeforeDateReached && !notAfter.Before(currentDate)
}

func calculateDaysToExpiry(currentDate time.Time, notAfter time.Time) int {

	diff := notAfter.Sub(currentDate)

	days := int(diff.Hours() / 24)

	if days < 0 {
		return 0
	}

	return days
}

// tlsVersionName maps a negotiated TLS version (the uint16 from the connection
// state) to a human-readable label. Unknown versions are reported with their
// raw hex value instead of being hidden.
func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

func buildCertDetails(version uint16, cert *x509.Certificate, now time.Time) TLSCertificateDetails {
	return TLSCertificateDetails{
		Issuer:         cert.Issuer.CommonName,
		Version:        tlsVersionName(version),
		ExpirationDate: cert.NotAfter,
		IsValid:        isValid(now, cert.NotAfter, cert.NotBefore),
		Names:          cert.DNSNames,
		DaysToExpiry:   calculateDaysToExpiry(now, cert.NotAfter),
	}
}

func TLSVerification(ctx context.Context, target string, insecure bool, now time.Time) (TLSCertificateDetails, error) {

	dialer := tls.Dialer{
		Config: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	conn, err := dialer.DialContext(ctx, "tcp", target)

	if err != nil {
		return TLSCertificateDetails{}, fmt.Errorf("could not establish TLS connection to %q: %w", target, err)
	}

	defer func() { _ = conn.Close() }()

	connState := conn.(*tls.Conn).ConnectionState()

	if len(connState.PeerCertificates) == 0 {
		return TLSCertificateDetails{}, fmt.Errorf("no certificates presented by %q", target)
	}

	return buildCertDetails(connState.Version, connState.PeerCertificates[0], now), nil
}
