# Security Report

> Generated: Initial setup

## Summary

| Category | Issues | Status |
|----------|--------|--------|
| SAST (gosec) | 0 | ✅ |
| High Severity | 0 | ✅ |
| Medium Severity | 0 | ✅ |
| Vulnerabilities | 0 | ✅ |

## Security Tools

### SAST - Static Application Security Testing
- **gosec**: Go security checker - detects common vulnerabilities
- **CodeQL**: GitHub's semantic code analysis

### SCA - Software Composition Analysis
- **govulncheck**: Official Go vulnerability database checker
- **nancy**: Sonatype dependency vulnerability scanner

### Secret Detection
- **gitleaks**: Scan for hardcoded secrets
- **TruffleHog**: Deep secret scanning

### License Compliance
- **go-licenses**: Check dependency licenses

## Security Rules Checked

### Critical (CWE)
| Rule | Description |
|------|-------------|
| G101 | Hardcoded credentials |
| G102 | Bind to all interfaces |
| G103 | Unsafe block usage |
| G104 | Unhandled errors |
| G106 | SSH InsecureIgnoreHostKey |
| G107 | URL provided to HTTP request |
| G108 | Profiling endpoint exposed |
| G109 | Integer overflow in strconv |
| G110 | Decompression bomb |

### High
| Rule | Description |
|------|-------------|
| G201 | SQL query construction |
| G202 | SQL string formatting |
| G203 | Unescaped HTML templates |
| G204 | Command injection |

### Medium
| Rule | Description |
|------|-------------|
| G301 | Poor file permissions on mkdir |
| G302 | Poor file permissions on create |
| G303 | File creation in shared tmp |
| G304 | File path provided as taint input |
| G305 | File traversal when extracting |
| G306 | Poor file permissions on write |
| G307 | Defer unsafe method call |

### Low
| Rule | Description |
|------|-------------|
| G401 | Use of weak crypto (MD5/SHA1) |
| G402 | TLS MinVersion too low |
| G403 | RSA keys < 2048 bits |
| G404 | Weak random number generator |
| G501 | Blacklisted import md5 |
| G502 | Blacklisted import des |
| G503 | Blacklisted import rc4 |

## OWASP Top 10 Coverage

| # | Risk | Covered By |
|---|------|------------|
| A01 | Broken Access Control | Manual review |
| A02 | Cryptographic Failures | G401-G404, G501-G503 |
| A03 | Injection | G201-G204 |
| A04 | Insecure Design | CodeQL |
| A05 | Security Misconfiguration | G102, G108, G301-G307 |
| A06 | Vulnerable Components | govulncheck, nancy |
| A07 | Auth Failures | Manual review |
| A08 | Data Integrity Failures | CodeQL |
| A09 | Logging Failures | Manual review |
| A10 | SSRF | G107 |

---
*This report is auto-updated by the Security Analysis workflow*
