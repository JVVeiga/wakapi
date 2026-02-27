# Stage 1: Build frontend assets (Tailwind CSS + Iconify icons)
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend
WORKDIR /src
RUN apk add --no-cache brotli
COPY package.json ./
COPY scripts/ ./scripts/
COPY static/ ./static/
COPY views/ ./views/
COPY tailwind.config.js ./
RUN npm install && npm run build:all:compress

# Stage 2: Build Go binary
FROM --platform=$BUILDPLATFORM golang:alpine AS build-env
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . .

# Overwrite dist assets with freshly built ones from frontend stage
COPY --from=frontend /src/static/assets/css/app.dist.css static/assets/css/app.dist.css
COPY --from=frontend /src/static/assets/css/app.dist.css.br static/assets/css/app.dist.css.br
COPY --from=frontend /src/static/assets/js/icons.dist.js static/assets/js/icons.dist.js
COPY --from=frontend /src/static/assets/js/icons.dist.js.br static/assets/js/icons.dist.js.br

ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 GOEXPERIMENT=greenteagc,jsonv2 go build -ldflags "-s -w" -v -o wakapi main.go

WORKDIR /staging
RUN mkdir ./data ./app && \
    cp /src/wakapi app/ && \
    cp /src/config.default.yml app/config.yml && \
    sed -i 's/listen_ipv6: ::1/listen_ipv6: "-"/g' app/config.yml

# Run Stage

# When running the application using `docker run`, you can pass environment variables
# to override config values using `-e` syntax.
# Available options can be found in [README.md#-configuration](README.md#-configuration)

# Note on the distroless image:
# we could use `base:nonroot`, which already includes ca-certificates and tz, but that one it actually larger than alpine,
# probably because of glibc, whereas alpine uses musl. The `static:nonroot`, doesn't include any libc implementation, because only meant for true static binaries without cgo, etc.

FROM gcr.io/distroless/static:nonroot
WORKDIR /app

# See README.md and config.default.yml for all config options
ENV ENVIRONMENT=prod \
    WAKAPI_DB_TYPE=sqlite3 \
    WAKAPI_DB_USER='' \
    WAKAPI_DB_PASSWORD='' \
    WAKAPI_DB_HOST='' \
    WAKAPI_DB_NAME=/data/wakapi.db \
    WAKAPI_PASSWORD_SALT='' \
    WAKAPI_LISTEN_IPV4='0.0.0.0' \
    WAKAPI_INSECURE_COOKIES='true' \
    WAKAPI_ALLOW_SIGNUP='true'

COPY --from=build-env --chown=nonroot:nonroot --chmod=0444 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-env --chown=nonroot:nonroot --chmod=0444 /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=build-env --chown=nonroot:nonroot /staging/app /app
COPY --from=build-env --chown=nonroot:nonroot /staging/data /data

LABEL org.opencontainers.image.url="https://github.com/muety/wakapi" \
    org.opencontainers.image.documentation="https://github.com/muety/wakapi" \
    org.opencontainers.image.source="https://github.com/muety/wakapi" \
    org.opencontainers.image.title="Wakapi" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.description="A minimalist, self-hosted WakaTime-compatible backend for coding statistics"

USER nonroot

EXPOSE 3000

ENTRYPOINT ["/app/wakapi"]
