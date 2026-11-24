# ---- build ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cpu-monitor ./cmd/cpu-monitor


FROM scratch
COPY --from=build /out/cpu-monitor /cpu-monitor
# Numeric UID so Kubernetes runAsNonRoot can verify it.
USER 65532:65532
ENTRYPOINT ["/cpu-monitor"]
