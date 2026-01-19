# Security Report

> Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

## Summary

| Category | Issues | Status |
|----------|--------|--------|
| SAST (gosec) | 0 | ✅ |
| High Severity | 0 | ✅ |
| Medium Severity | 0 | ✅ |
| Vulnerabilities | 0 | ✅ |
| Secrets Detected | 0 | ✅ |

## SAST Results (gosec)

```
Results:


Summary:
  Gosec  : dev
  Files  : 14
  Lines  : 2386
  Nosec  : 0
  Issues : 0
```

## Dependency Vulnerabilities

```
govulncheck: loading packages: 
There are errors with the provided package patterns:

/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/gamification/achievements.go:1:1: package requires newer Go version go1.24
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/species/species.go:2:1: package requires newer Go version go1.24
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/slices/iter.go:50:17: cannot range over seq (variable of type iter.Seq[E])
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/quiz/question.go:1:1: package requires newer Go version go1.24
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/ports/player_repository.go:1:1: package requires newer Go version go1.24
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/quiz/factory.go:2:1: package requires newer Go version go1.24
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/maps/iter.go:51:20: cannot range over seq (variable of type iter.Seq2[K, V])
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/crypto/x509/verify.go:1490:20: cannot range over pg.parents() (value of type iter.Seq[*policyGraphNode])
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http/handlers.go:2:1: package requires newer Go version go1.24
/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/inaturalist/client.go:2:1: package requires newer Go version go1.24
/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server/main.go:2:1: package requires newer Go version go1.24

For details on package patterns, see https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns.
```

## License Compliance

```
E0119 20:07:19.503827    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/domain/gamification: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/gamification" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.503867    4550 library.go:117] Package errors does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
W0119 20:07:19.503878    4550 library.go:101] "math" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/math/dim_amd64.s
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/math/exp_amd64.s
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/math/floor_amd64.s
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/math/hypot_amd64.s
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/math/log_amd64.s
E0119 20:07:19.503886    4550 library.go:117] Package math does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.503893    4550 library.go:117] Package time does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.516027    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/domain/species: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/species" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.529052    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/domain/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/domain/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.534217    4550 library.go:117] Package bytes does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
W0119 20:07:19.534236    4550 library.go:101] "crypto/md5" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/crypto/md5/md5block_amd64.s
E0119 20:07:19.534245    4550 library.go:117] Package crypto/md5 does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534249    4550 library.go:117] Package crypto/rand does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
W0119 20:07:19.534257    4550 library.go:101] "crypto/sha1" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/crypto/sha1/sha1block_amd64.s
E0119 20:07:19.534261    4550 library.go:117] Package crypto/sha1 does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534269    4550 library.go:117] Package database/sql/driver does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534273    4550 library.go:117] Package encoding/binary does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534279    4550 library.go:117] Package encoding/hex does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534284    4550 library.go:117] Package encoding/json does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534288    4550 library.go:117] Package fmt does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534292    4550 library.go:117] Package hash does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534296    4550 library.go:117] Package io does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534314    4550 library.go:117] Package net does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534318    4550 library.go:117] Package os does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534323    4550 library.go:117] Package strings does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.534327    4550 library.go:117] Package sync does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.556533    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/ports: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/ports" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.556558    4550 library.go:117] Package context does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.579864    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/application/quiz: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/application/quiz" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.579883    4550 library.go:117] Package math/rand does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.604974    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/adapters/http: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/http" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.604994    4550 library.go:117] Package net/http does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.631944    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/internal/adapters/inaturalist: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/internal/adapters/inaturalist" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.631966    4550 library.go:117] Package net/url does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.631970    4550 library.go:117] Package strconv does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
E0119 20:07:19.659577    4550 library.go:122] Failed to find license for github.com/fieve/naturieux/cmd/server: cannot find a known open source license for "/home/runner/work/Naturieux.fr/Naturieux.fr/cmd/server" whose name matches regexp ^(?i)((UN)?LICEN(S|C)E|COPYING|README|NOTICE).*$ and locates up until "/home/runner/work/Naturieux.fr/Naturieux.fr"
E0119 20:07:19.659606    4550 library.go:117] Package log does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
W0119 20:07:19.659617    4550 library.go:101] "os/signal" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/os/signal/sig.s
E0119 20:07:19.659624    4550 library.go:117] Package os/signal does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
W0119 20:07:19.659631    4550 library.go:101] "syscall" contains non-Go code that can't be inspected for further dependencies:
/home/runner/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.24.5.linux-amd64/src/syscall/asm_linux_amd64.s
E0119 20:07:19.659642    4550 library.go:117] Package syscall does not have module info. Non go modules projects are no longer supported. For feedback, refer to https://github.com/google/go-licenses/issues/128.
F0119 20:07:19.659653    4550 main.go:77] some errors occurred when loading direct and transitive dependency packages
```

---
*Generated by Security Analysis workflow*
