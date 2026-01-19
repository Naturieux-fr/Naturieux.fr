# Security Report

> Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Summary

| Category | Issues | Status |
|----------|--------|--------|
| SAST (gosec) | 0 | ✅ |
| High Severity | 0 | ✅ |
| Medium Severity | 0 | ✅ |
| Vulnerabilities | 12 | ❌ |
| Secrets Detected | 0 | ✅ |

## SAST Results (gosec)

```
Results:


Summary:
  Gosec  : dev
  Files  : 14
  Lines  : 2484
  Nosec  : 0
  Issues : 0
```

## Dependency Vulnerabilities

```
=== Symbol Results ===

Vulnerability #1: GO-2025-4175
    Improper application of excluded DNS name constraints when verifying
    wildcard names in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4175
  Standard library
    Found in: crypto/x509@go1.22.12
    Fixed in: crypto/x509@go1.24.11
    Example traces found:
      #1: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #2: GO-2025-4155
    Excessive resource consumption when printing error string for host
    certificate validation in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4155
  Standard library
    Found in: crypto/x509@go1.22.12
    Fixed in: crypto/x509@go1.24.11
    Example traces found:
      #1: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify
      #2: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls x509.Certificate.VerifyHostname

Vulnerability #3: GO-2025-4013
    Panic when validating certificates with DSA public keys in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4013
  Standard library
    Found in: crypto/x509@go1.22.12
    Fixed in: crypto/x509@go1.24.8
    Example traces found:
      #1: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #4: GO-2025-4012
    Lack of limit when parsing cookies can cause memory exhaustion in net/http
  More info: https://pkg.go.dev/vuln/GO-2025-4012
  Standard library
    Found in: net/http@go1.22.12
    Fixed in: net/http@go1.24.8
    Example traces found:
      #1: internal/adapters/inaturalist/client.go:150:30: inaturalist.Client.doRequest calls http.Client.Do

Vulnerability #5: GO-2025-4011
    Parsing DER payload can cause memory exhaustion in encoding/asn1
  More info: https://pkg.go.dev/vuln/GO-2025-4011
  Standard library
    Found in: encoding/asn1@go1.22.12
    Fixed in: encoding/asn1@go1.24.8
    Example traces found:
      #1: cmd/server/main.go:91:15: server.main calls signal.Notify, which eventually calls asn1.Unmarshal

Vulnerability #6: GO-2025-4010
    Insufficient validation of bracketed IPv6 hostnames in net/url
  More info: https://pkg.go.dev/vuln/GO-2025-4010
  Standard library
    Found in: net/url@go1.22.12
    Fixed in: net/url@go1.24.8
    Example traces found:
      #1: internal/adapters/inaturalist/client.go:142:40: inaturalist.Client.doRequest calls http.NewRequestWithContext, which calls url.Parse
      #2: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls url.ParseRequestURI
      #3: internal/adapters/inaturalist/client.go:150:30: inaturalist.Client.doRequest calls http.Client.Do, which eventually calls url.URL.Parse

Vulnerability #7: GO-2025-4009
    Quadratic complexity when parsing some invalid inputs in encoding/pem
  More info: https://pkg.go.dev/vuln/GO-2025-4009
  Standard library
    Found in: encoding/pem@go1.22.12
    Fixed in: encoding/pem@go1.24.8
    Example traces found:
      #1: cmd/server/main.go:91:15: server.main calls signal.Notify, which eventually calls pem.Decode

Vulnerability #8: GO-2025-4008
    ALPN negotiation error contains attacker controlled information in
    crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2025-4008
  Standard library
    Found in: crypto/tls@go1.22.12
    Fixed in: crypto/tls@go1.24.8
    Example traces found:
      #1: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls tls.Conn.HandshakeContext
      #2: internal/adapters/inaturalist/client.go:338:45: inaturalist.Client.Search calls json.Decoder.Decode, which eventually calls tls.Conn.Read
      #3: internal/application/quiz/service.go:212:14: quiz.Service.SubmitAnswer calls fmt.Printf, which eventually calls tls.Conn.Write
      #4: internal/adapters/inaturalist/client.go:150:30: inaturalist.Client.doRequest calls http.Client.Do, which eventually calls tls.Dialer.DialContext

Vulnerability #9: GO-2025-4007
    Quadratic complexity when checking name constraints in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4007
  Standard library
    Found in: crypto/x509@go1.22.12
    Fixed in: crypto/x509@go1.24.9
    Example traces found:
      #1: cmd/server/main.go:91:15: server.main calls signal.Notify, which eventually calls x509.CertPool.AppendCertsFromPEM
      #2: cmd/server/main.go:84:34: server.main calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify
      #3: cmd/server/main.go:91:15: server.main calls signal.Notify, which eventually calls x509.ParseCertificate

Vulnerability #10: GO-2025-3751
    Sensitive headers not cleared on cross-origin redirect in net/http
  More info: https://pkg.go.dev/vuln/GO-2025-3751
  Standard library
    Found in: net/http@go1.22.12
    Fixed in: net/http@go1.23.10
```

## License Compliance

```
E0119 21:06:18.489638    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/gamification" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.499369    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/species: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/species" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.509899    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.526408    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/ports: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/ports" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.538138    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.550472    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.563897    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/inaturalist: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/inaturalist" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 21:06:18.578279    4574 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/server: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/cmd/server
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/adapters/inaturalist
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/domain/species
Unknown license type  found for library github.com/Naturieux-fr/Naturieux.fr/internal/ports
```

---
*Generated by Security Analysis workflow*
