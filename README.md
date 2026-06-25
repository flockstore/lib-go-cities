# lib-go-cities

[![CI](https://github.com/flockstore/lib-go-cities/actions/workflows/ci.yml/badge.svg)](https://github.com/flockstore/lib-go-cities/actions/workflows/ci.yml)
[![Package](https://github.com/flockstore/lib-go-cities/actions/workflows/package.yml/badge.svg)](https://github.com/flockstore/lib-go-cities/actions/workflows/package.yml)
[![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)](https://github.com/flockstore/lib-go-cities/releases)
[![Coverage](https://img.shields.io/badge/coverage-%E2%89%A590%25-brightgreen.svg)](https://github.com/flockstore/lib-go-cities/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/flockstore/lib-go-cities.svg)](https://pkg.go.dev/github.com/flockstore/lib-go-cities)

Version: `1.0.1`

`lib-go-cities` is a dependency-free Go library and CLI for matching Colombian city text against a `cities.json` file compatible with [`flockstore/lib-cities-co`](https://github.com/flockstore/lib-cities-co).

Use it when users type city names manually and you need to decide which city code is safe to send to a carrier or marketplace integration. The matcher returns a confidence score and refuses to guess when the input is duplicated, ambiguous, incongruent, or below threshold.

## What It Solves

Users can write:

- `Cali`
- `Cali valle del cauca`
- `Bucaramanga (Santander)`
- `Santiago de Cali`
- `Armenia`

The first four can resolve to one city. `Armenia` is rejected unless department evidence is present because the source contains more than one Armenia.

## Install

As a Go dependency:

```sh
go get github.com/flockstore/lib-go-cities@v1.0.1
```

As a CLI:

```sh
go install github.com/flockstore/lib-go-cities/cmd/cities@v1.0.1
```

As a container from GHCR:

```sh
docker pull ghcr.io/flockstore/lib-go-cities:v1.0.1
docker run --rm -v "$PWD/cities.json:/data/cities.json:ro" \
  ghcr.io/flockstore/lib-go-cities:v1.0.1 \
  -source /data/cities.json -city "Cali valle del cauca" -threshold 0.8
```

`cities.json` is not bundled in the binary or image. Provide it from your application, build pipeline, or local data source.

## CLI Usage

All CLI responses are JSON.

```sh
cities -source ./cities.json -city "Bogota" -department "Cundinamarca" -threshold 0.80
```

Flags:

- `-source`: JSON source path. Defaults to `cities.json`.
- `-city`: required user city text.
- `-department`: optional department text. This improves matching.
- `-threshold`: required confidence from `0` to `1`. Defaults to `0.75`.

Matched response:

```json
{
  "matched": true,
  "city": "Cali",
  "department": "VALLE DEL CAUCA",
  "code": "76001",
  "delivery": 1,
  "extras": true,
  "confidence": 1,
  "threshold": 0.8,
  "source": "./cities.json",
  "version": "1.0.1"
}
```

Rejected response:

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
  "version": "1.0.1"
}
```

Rejection reasons:

- `LOW_THRESHOLD`: the best candidate did not meet the requested confidence.
- `DUPLICATED`: the city exists in multiple departments and no department evidence resolves it.
- `INCONGRUENT`: city and department evidence conflict.
- `AMBIGUOUS`: multiple different cities are plausible at the same confidence.

## Library Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/flockstore/lib-go-cities/platform"
)

func main() {
	matcher, err := platform.LoadFile("cities.json")
	if err != nil {
		panic(err)
	}

	match, found, err := matcher.Match(context.Background(), platform.SearchRequest{
		City:       "Medellin",
		Department: "Antioquia",
		Threshold:  0.80,
	})
	if err != nil {
		panic(err)
	}
	if !found {
		fmt.Println(match.Reason)
		return
	}

	fmt.Println(match.City.Code)
}
```

`LoadFile` reads and closes the JSON file during loading. Matching does not keep the file open.

For async callers:

```go
result := <-matcher.MatchAsync(ctx, platform.SearchRequest{City: "Cali"})
```

## Source Data

JSON and CSV files are intentionally ignored by git. Keep `cities.json` local, generated, or supplied by your application pipeline.

Expected JSON shape:

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

`code` can be a JSON string or number. The library keeps it as a string so leading zeros are not lost.

## Testing And Benchmarks

Run tests, vet, and coverage:

```sh
go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -n 1
go vet ./...
```

CI enforces at least `90%` total coverage.

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
BenchmarkMatchRealExactWithDepartment    ~584 ns/op     ~715 B/op   11 allocs/op
BenchmarkMatchRealEmbeddedDepartment     ~571 ns/op     ~796 B/op   10 allocs/op
BenchmarkMatchRealKnownEdges             ~783 ns/op    ~1034 B/op   11 allocs/op
BenchmarkMatchRealMisses                 ~224 us/op     ~285 B/op    5 allocs/op
BenchmarkMatchRealParallel               ~341 ns/op     ~715 B/op   11 allocs/op
```

## Publishing

On every commit to `main`, GitHub Actions:

- runs formatting, tests, coverage, vet, and CLI compile checks;
- uploads CLI binaries for Linux, macOS, and Windows as workflow artifacts;
- publishes the CLI container to `ghcr.io/flockstore/lib-go-cities` with `latest`, `v1.0.1`, and commit SHA tags.
