FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY platform ./platform
COPY cmd ./cmd

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cities ./cmd/cities

FROM scratch

COPY --from=build /out/cities /cities
ENTRYPOINT ["/cities"]

