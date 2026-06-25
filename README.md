# lib-go-cities

[![CI](https://github.com/flockstore/lib-go-cities/actions/workflows/ci.yml/badge.svg)](https://github.com/flockstore/lib-go-cities/actions/workflows/ci.yml)
[![Package](https://github.com/flockstore/lib-go-cities/actions/workflows/package.yml/badge.svg)](https://github.com/flockstore/lib-go-cities/actions/workflows/package.yml)
[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/flockstore/lib-go-cities/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/flockstore/lib-go-cities.svg)](https://pkg.go.dev/github.com/flockstore/lib-go-cities)

Version: `1.0.0`

`lib-go-cities` is a fast, dependency-free Go library and CLI for validating and matching Colombian city text against a `cities.json` source compatible with [`flockstore/lib-cities-co`](https://github.com/flockstore/lib-cities-co).

The goal is to help marketplaces, checkout flows, and carrier integrations handle free-text city inputs. Users often cannot choose from a fixed list, or carriers provide confusing destination names. This package gives applications a confidence score for what the user wrote and returns the city code to use when the score meets the configured threshold. If the score is below the threshold, the CLI returns a JSON response with `"no coincidences"`.

## Install

```sh
go get github.com/flockstore/lib-go-cities@v1.0.0
go install github.com/flockstore/lib-go-cities/cmd/cities@v1.0.0
```

## CLI

All CLI responses are JSON.

```sh
cities -source ./cities.json -city "Bogota" -department "Cundinamarca" -threshold 0.80
```

Successful match:

```json
{
  "matched": true,
  "city": "Bogota, D.C.",
  "department": "CUNDINAMARCA",
  "code": "11001",
  "delivery": 1,
  "extras": true,
  "confidence": 0.98,
  "threshold": 0.8,
  "source": "./cities.json",
  "version": "1.0.0"
}
```

No match:

```json
{
  "matched": false,
  "message": "no coincidences",
  "reason": "LOW_THRESHOLD",
  "threshold": 0.8,
  "source": "./cities.json",
  "version": "1.0.0"
}
```

Flags:

- `-city`: required city text to search.
- `-department`: optional department text that improves matching.
- `-threshold`: minimum confidence from `0` to `1`; defaults to `0.75`.
- `-source`: JSON source path; defaults to `cities.json`.

## Library

```go
import "github.com/flockstore/lib-go-cities/platform"

matcher, err := platform.LoadFile("cities.json")
if err != nil {
	return err
}

match, found, err := matcher.Match(ctx, platform.SearchRequest{
	City:       "Medellin",
	Department: "Antioquia",
	Threshold:  0.80,
})
if err != nil {
	return err
}
if !found {
	return nil
}

fmt.Println(match.City.Code)
```

For async callers:

```go
result := <-matcher.MatchAsync(ctx, platform.SearchRequest{City: "Cali"})
```

## Matching Strategy

The matcher normalizes accents, punctuation, casing, and spacing before scoring. It also handles common noisy inputs where users write extra context in the city field:

- `Santiago de Cali` can resolve to `Cali`.
- `Cali valle del cauca` can resolve to `Cali` using the embedded department.
- `Bucaramanga (Santander)` can resolve to `Bucaramanga` using the embedded department.

When the matcher should not guess, it rejects the lookup with a machine-readable reason:

- `LOW_THRESHOLD`: the best candidate did not meet the requested confidence.
- `DUPLICATED`: the city name exists in multiple departments and no department evidence resolves it, for example `Armenia`.
- `INCONGRUENT`: the city was recognized, but the provided or embedded department conflicts with known records.
- `AMBIGUOUS`: multiple different cities are plausible at the same confidence.

Rejected duplicate and ambiguous responses can include `suggestions`:

```json
{
  "matched": false,
  "message": "no coincidences",
  "reason": "DUPLICATED",
  "suggestions": [
    {
      "city": "Armenia",
      "department": "ANTIOQUIA",
      "code": "05059",
      "delivery": 6,
      "confidence": 1
    },
    {
      "city": "Armenia",
      "department": "QUINDIO",
      "code": "63001",
      "delivery": 1,
      "confidence": 1
    }
  ],
  "threshold": 0.8,
  "source": "./cities.json",
  "version": "1.0.0"
}
```

## Testing And Benchmarks

Run the full test suite:

```sh
go test ./...
go vet ./...
```

Run real-source regression tests and benchmarks. These use local `cities.json` when it is present and skip cleanly without it:

```sh
go test ./platform -run 'TestRealSource' -count=1 -v
go test ./platform -bench 'BenchmarkMatchReal' -benchmem -run '^$'
```

Capture pprof profiles for the slowest fuzzy miss path:

```sh
mkdir -p profiles
go test ./platform -bench 'BenchmarkMatchRealMisses' -benchmem -run '^$' \
  -cpuprofile profiles/cpu-miss.pprof \
  -memprofile profiles/mem-miss.pprof
go tool pprof -top profiles/cpu-miss.pprof
go tool pprof -top -alloc_space profiles/mem-miss.pprof
```

Representative local results on Apple M4 Pro with the current `cities.json`:

```text
BenchmarkMatchRealExactWithDepartment    ~560 ns/op     ~715 B/op   11 allocs/op
BenchmarkMatchRealEmbeddedDepartment     ~555 ns/op     ~796 B/op   10 allocs/op
BenchmarkMatchRealKnownEdges             ~770 ns/op    ~1034 B/op   11 allocs/op
BenchmarkMatchRealMisses                 ~221 us/op     ~285 B/op    5 allocs/op
BenchmarkMatchRealParallel               ~360 ns/op     ~715 B/op   11 allocs/op
```

## Source Data

JSON and CSV files are intentionally ignored by git. Keep `cities.json` local, generated, or supplied by your application pipeline. The expected JSON shape is:

```json
[
  {
    "code": "11001",
    "name": "Bogota, D.C.",
    "normalized": "Bogota, D.C.",
    "department": "CUNDINAMARCA",
    "delivery": 1,
    "extras": true
  }
]
```
