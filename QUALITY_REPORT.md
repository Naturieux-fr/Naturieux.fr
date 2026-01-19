# Quality Report

> Last updated: Initial setup

## Quality Gate: ✅ PASSED

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Test Coverage | 72.1% | ≥ 70% | ✅ |
| Critical Issues | 0 | ≤ 0 | ✅ |
| Security Issues | 0 | = 0 | ✅ |
| Lint Issues | TBD | - | ℹ️ |
| High Complexity | TBD | ≤ 10 | ℹ️ |
| Code Duplicates | TBD | ≤ 5 | ℹ️ |

## Coverage by Package

```
github.com/fieve/naturieux/internal/domain/species        100.0%
github.com/fieve/naturieux/internal/adapters/inaturalist   91.5%
github.com/fieve/naturieux/internal/domain/gamification    81.0%
github.com/fieve/naturieux/internal/domain/quiz            77.5%
github.com/fieve/naturieux/internal/application/quiz       75.4%
github.com/fieve/naturieux/internal/adapters/http          60.5%
total:                                                     72.1%
```

## Quality Gate Thresholds

| Threshold | Value | Description |
|-----------|-------|-------------|
| Minimum Coverage | 70% | Minimum test coverage required |
| Max Critical Issues | 0 | No unchecked errors or security issues |
| Max Security Issues | 0 | No high-severity vulnerabilities |
| Max Complexity | 15 | Maximum cyclomatic complexity per function |
| Max Duplicates | 5 | Maximum duplicate code blocks |

## Issue Categories

### Critical (Blocker)
- Unchecked errors (`errcheck`)
- Security vulnerabilities (`gosec`)
- Type errors (`typecheck`)
- Data races

### Major
- Static analysis issues (`staticcheck`)
- Unused code (`unused`)
- Inefficient patterns

### Minor
- Code style (`stylecheck`)
- Naming conventions
- Documentation

### Info
- Duplicate code (`dupl`)
- High complexity (`gocyclo`)
- Long functions (`funlen`)

---
*This report is auto-updated by CI on each push to main*
