# Security Report

> Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Summary

| Category | Issues | Status |
|----------|--------|--------|
| SAST (gosec) | 15 | ❌ |
| High Severity | 4 | ❌ |
| Medium Severity | 4 | ✅ |
| Vulnerabilities | 0 | ✅ |
| Secrets Detected | 0 | ✅ |

## SAST Results (gosec)

```
Results:


[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http/locate.go:30] - G404 (CWE-338): Use of weak random number generator (math/rand or math/rand/v2 instead of crypto/rand) (Confidence: MEDIUM, Severity: HIGH)
    29: 		}
  > 30: 		return rand.Intn(n)
    31: 	}}

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http/handlers.go:717] - G704 (CWE-918): SSRF via taint analysis (Confidence: HIGH, Severity: HIGH)
    716: 	req.Header.Set("User-Agent", "Naturieux/1.0 (https://naturieux.fr)")
  > 717: 	resp, err := http.DefaultClient.Do(req)
    718: 	if err != nil {

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http/handlers.go:711] - G704 (CWE-918): SSRF via taint analysis (Confidence: HIGH, Severity: HIGH)
    710: 	// Remote (e.g. iNaturalist / S3): proxy the bytes server-side.
  > 711: 	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, mediaURL, nil)
    712: 	if err != nil {

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/storage/local.go:24] - G703 (CWE-22): Path traversal via taint analysis (Confidence: HIGH, Severity: HIGH)
    23: func NewLocal(dir string) (*Local, error) {
  > 24: 	if err := os.MkdirAll(dir, 0o755); err != nil {
    25: 		return nil, fmt.Errorf("creating media dir: %w", err)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/taxref/photocsv.go:21] - G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
    20: func ParsePhotoCSV(path string) ([]PhotoCSVRow, error) {
  > 21: 	f, err := os.Open(path)
    22: 	if err != nil {

Autofix: Consider using os.Root to scope file access under a fixed root (Go >=1.24). Prefer root.Open/root.Stat over os.Open/os.Stat to prevent directory traversal.

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/storage/local.go:47] - G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
    46: 
  > 47: 	f, err := os.Create(path)
    48: 	if err != nil {

Autofix: Consider using os.Root to scope file access under a fixed root (Go >=1.24). Prefer root.Open/root.Stat over os.Open/os.Stat to prevent directory traversal.

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importphotos/main.go:90] - G304 (CWE-22): Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
    89: 		path := filepath.Join(*dir, row.Photo)
  > 90: 		raw, err := os.ReadFile(path)
    91: 		if err != nil {

Autofix: Consider using os.Root to scope file access under a fixed root (Go >=1.24). Prefer root.Open/root.Stat over os.Open/os.Stat to prevent directory traversal.

[/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/storage/local.go:24] - G301 (CWE-276): Expect directory permissions to be 0750 or less (Confidence: HIGH, Severity: MEDIUM)
    23: func NewLocal(dir string) (*Local, error) {
  > 24: 	if err := os.MkdirAll(dir, 0o755); err != nil {
    25: 		return nil, fmt.Errorf("creating media dir: %w", err)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:404] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    403: 	}
  > 404: 	log.Printf("Admin account ready: %s", user)
    405: }

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:341] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    340: 		} else {
  > 341: 			log.Printf("TAXREF loaded: %d species (version %q)", count, repo.Version(ctx))
    342: 		}

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:247] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    246: 		log.Printf("Health check: http://localhost:%s/health", port)
  > 247: 		log.Printf("API: http://localhost:%s/api/v1/", port)
    248: 		if err := server.ListenAndServe(); err != nil GOSEC_REPORT_PLACEHOLDERGOSEC_REPORT_PLACEHOLDER err != http.ErrServerClosed {

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:246] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    245: 		log.Printf("Frontend: http://localhost:%s/", port)
  > 246: 		log.Printf("Health check: http://localhost:%s/health", port)
    247: 		log.Printf("API: http://localhost:%s/api/v1/", port)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:245] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    244: 		log.Printf("Starting Naturieux server on port %s", port)
  > 245: 		log.Printf("Frontend: http://localhost:%s/", port)
    246: 		log.Printf("Health check: http://localhost:%s/health", port)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:244] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    243: 	go func() {
  > 244: 		log.Printf("Starting Naturieux server on port %s", port)
    245: 		log.Printf("Frontend: http://localhost:%s/", port)

Autofix: 
```

## Dependency Vulnerabilities

```
No vulnerabilities found.
```

## License Compliance

```
E0611 19:52:18.888798    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/gamification" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:18.899851    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/species: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/species" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:18.925157    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:18.942205    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/ports: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/ports" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:18.955250    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/sqlite" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
W0611 19:52:18.960216    2311 library.go:101] "golang.org/x/sys/unix" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/sys@v0.46.0/unix/asm_linux_amd64.s
W0611 19:52:18.985992    2311 library.go:101] "modernc.org/libc" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/modernc.org/libc@v1.72.3/abi0_linux_amd64.s
/home/runner/go/pkg/mod/modernc.org/libc@v1.72.3/tls_linux_amd64.s
E0611 19:52:19.057644    2311 library.go:122] Failed to find license for modernc.org/mathutil: cannot find a known open source license for "/home/runner/go/pkg/mod/modernc.org/mathutil@v1.7.1" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/go/pkg/mod/modernc.org/mathutil@v1.7.1"
E0611 19:52:19.126803    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/taxref" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.143469    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importoccurrences: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importoccurrences" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.161037    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/storage" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
W0611 19:52:19.190810    2311 library.go:101] "github.com/cespare/xxhash/v2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/cespare/xxhash/v2@v2.3.0/xxhash_amd64.s
W0611 19:52:19.194801    2311 library.go:101] "github.com/klauspost/compress/s2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/compress@v1.18.6/s2/decode_amd64.s
/home/runner/go/pkg/mod/github.com/klauspost/compress@v1.18.6/s2/encodeblock_amd64.s
W0611 19:52:19.311166    2311 library.go:101] "github.com/klauspost/crc32" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/crc32@v1.3.0/crc32_amd64.s
W0611 19:52:19.316436    2311 library.go:101] "golang.org/x/sys/cpu" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/sys@v0.46.0/cpu/cpu_gc_x86.s
W0611 19:52:19.321117    2311 library.go:101] "github.com/minio/crc64nvme" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/minio/crc64nvme@v1.1.1/crc64_amd64.s
W0611 19:52:19.349603    2311 library.go:101] "github.com/klauspost/cpuid/v2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/cpuid/v2@v2.2.11/cpuid_amd64.s
W0611 19:52:19.353019    2311 library.go:101] "github.com/minio/md5-simd" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/block16_amd64.s
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/block8_amd64.s
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/md5block_amd64.s
W0611 19:52:19.551459    2311 library.go:101] "golang.org/x/crypto/argon2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/argon2/blamka_amd64.s
W0611 19:52:19.556331    2311 library.go:101] "golang.org/x/crypto/blake2b" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/blake2b/blake2bAVX2_amd64.s
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/blake2b/blake2b_amd64.s
W0611 19:52:19.831627    2311 library.go:101] "github.com/zeebo/xxh3" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_avx512_amd64.s
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_avx_amd64.s
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_sse_amd64.s
E0611 19:52:19.890740    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/media: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/media" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.916142    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importphotos: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importphotos" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.942108    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importtaxref: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importtaxref" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.969173    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/cache: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/cache" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:19.997444    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/auth: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/auth" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:20.060424    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/account: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/account" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:20.090123    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/challenge" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:20.120691    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:20.152112    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/room: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/room" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0611 19:52:20.184616    2311 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
```

---
*Generated by Security Analysis workflow*
