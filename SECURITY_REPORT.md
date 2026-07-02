# Security Report

> Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Summary

| Category | Issues | Status |
|----------|--------|--------|
| SAST (gosec) | 10 | ❌ |
| High Severity | 1 | ❌ |
| Medium Severity | 0 | ✅ |
| Vulnerabilities | 0 | ✅ |
| Secrets Detected | 0 | ✅ |

## SAST Results (gosec)

```
Results:


[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/bootstrap.go:93] - G704 (CWE-918): SSRF via taint analysis (Confidence: HIGH, Severity: HIGH)
    92: 	}
  > 93: 	resp, err := client.Do(req)
    94: 	if err != nil {

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:408] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    407: 	}
  > 408: 	log.Printf("Admin account ready: %s", user)
    409: }

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:345] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    344: 		} else {
  > 345: 			log.Printf("TAXREF loaded: %d species (version %q)", count, repo.Version(ctx))
    346: 		}

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:251] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    250: 		log.Printf("Health check: http://localhost:%s/health", port)
  > 251: 		log.Printf("API: http://localhost:%s/api/v1/", port)
    252: 		if err := server.ListenAndServe(); err != nil GOSEC_REPORT_PLACEHOLDERGOSEC_REPORT_PLACEHOLDER err != http.ErrServerClosed {

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:250] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    249: 		log.Printf("Frontend: http://localhost:%s/", port)
  > 250: 		log.Printf("Health check: http://localhost:%s/health", port)
    251: 		log.Printf("API: http://localhost:%s/api/v1/", port)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:249] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    248: 		log.Printf("Starting Naturieux server on port %s", port)
  > 249: 		log.Printf("Frontend: http://localhost:%s/", port)
    250: 		log.Printf("Health check: http://localhost:%s/health", port)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:248] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    247: 	go func() {
  > 248: 		log.Printf("Starting Naturieux server on port %s", port)
    249: 		log.Printf("Frontend: http://localhost:%s/", port)

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:64] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    63: 	defer func() { _ = db.Close() }()
  > 64: 	log.Printf("Database: %s", dbPath)
    65: 

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/bootstrap.go:54] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    53: 				n, _ := repo.CountSpecies(ctx)
  > 54: 				log.Printf("Bootstrap: TAXREF imported (%d species)", n)
    55: 			}

Autofix: 

[/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/bootstrap.go:43] - G706 (CWE-117): Log injection via taint analysis (Confidence: HIGH, Severity: LOW)
    42: 		if n, _ := repo.CountSpecies(ctx); n > 0 {
  > 43: 			log.Printf("Bootstrap: TAXREF already present (%d species), skipping download", n)
    44: 		} else {

Autofix: 

Summary:
  Gosec  : dev
  Files  : 54
  Lines  : 11126
  Nosec  : 10
  Issues : 10
```

## Dependency Vulnerabilities

```
No vulnerabilities found.
```

## License Compliance

```
E0612 22:37:54.912898    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/gamification" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:54.921201    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/species: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/species" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:54.940169    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/domain/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:54.952781    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/ports: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/ports" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:54.962318    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/sqlite" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
W0612 22:37:54.965808    2151 library.go:101] "golang.org/x/sys/unix" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/sys@v0.46.0/unix/asm_linux_amd64.s
W0612 22:37:54.984518    2151 library.go:101] "modernc.org/libc" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/modernc.org/libc@v1.72.3/abi0_linux_amd64.s
/home/runner/go/pkg/mod/modernc.org/libc@v1.72.3/tls_linux_amd64.s
E0612 22:37:55.040091    2151 library.go:122] Failed to find license for modernc.org/mathutil: cannot find a known open source license for "/home/runner/go/pkg/mod/modernc.org/mathutil@v1.7.1" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/go/pkg/mod/modernc.org/mathutil@v1.7.1"
E0612 22:37:55.095268    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/taxref: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/taxref" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.107717    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importoccurrences: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importoccurrences" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.120517    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/storage: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/storage" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
W0612 22:37:55.142487    2151 library.go:101] "github.com/cespare/xxhash/v2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/cespare/xxhash/v2@v2.3.0/xxhash_amd64.s
W0612 22:37:55.145572    2151 library.go:101] "github.com/klauspost/compress/s2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/compress@v1.18.6/s2/decode_amd64.s
/home/runner/go/pkg/mod/github.com/klauspost/compress@v1.18.6/s2/encodeblock_amd64.s
W0612 22:37:55.234665    2151 library.go:101] "github.com/klauspost/crc32" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/crc32@v1.3.0/crc32_amd64.s
W0612 22:37:55.238820    2151 library.go:101] "golang.org/x/sys/cpu" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/sys@v0.46.0/cpu/cpu_gc_x86.s
W0612 22:37:55.242378    2151 library.go:101] "github.com/minio/crc64nvme" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/minio/crc64nvme@v1.1.1/crc64_amd64.s
W0612 22:37:55.263801    2151 library.go:101] "github.com/klauspost/cpuid/v2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/klauspost/cpuid/v2@v2.2.11/cpuid_amd64.s
W0612 22:37:55.266622    2151 library.go:101] "github.com/minio/md5-simd" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/block16_amd64.s
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/block8_amd64.s
/home/runner/go/pkg/mod/github.com/minio/md5-simd@v1.1.2/md5block_amd64.s
W0612 22:37:55.424114    2151 library.go:101] "golang.org/x/crypto/argon2" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/argon2/blamka_amd64.s
W0612 22:37:55.428456    2151 library.go:101] "golang.org/x/crypto/blake2b" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/blake2b/blake2bAVX2_amd64.s
/home/runner/go/pkg/mod/golang.org/x/crypto@v0.53.0/blake2b/blake2b_amd64.s
W0612 22:37:55.703792    2151 library.go:101] "github.com/zeebo/xxh3" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_avx512_amd64.s
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_avx_amd64.s
/home/runner/go/pkg/mod/github.com/zeebo/xxh3@v1.1.0/accum_vector_sse_amd64.s
E0612 22:37:55.748383    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/media: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/media" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.767104    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importphotos: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importphotos" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.788368    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/cmd/importtaxref: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/importtaxref" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.808379    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/cache: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/cache" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.830001    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/auth: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/auth" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.877418    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/account: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/account" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.902100    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/challenge: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/challenge" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.924597    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.947678    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/application/room: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/room" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0612 22:37:55.971606    2151 library.go:122] Failed to find license for github.com/Naturieux-fr/Naturieux.fr/internal/adapters/http: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
```

---
*Generated by Security Analysis workflow*
