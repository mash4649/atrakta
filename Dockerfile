FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 go build -trimpath -o /out/atrakta ./cmd/atrakta

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=build /out/atrakta /usr/local/bin/atrakta

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/atrakta"]
