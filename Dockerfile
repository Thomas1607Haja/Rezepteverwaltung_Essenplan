# Image der Home-Assistant-App „Rezeptplaner“.
#
# Gebaut wird es von der GitHub Action (.github/workflows), nie auf dem
# Green. Kontext ist der Hauptordner des Repositorys, weil Weboberfläche
# (web/) und Server (server/) gebraucht werden.
#
# Drei Stufen: Die ersten beiden laufen auf der Plattform des
# Build-Rechners (--platform=$BUILDPLATFORM) und erzeugen die fertige
# Programmdatei per Cross-Build. Die letzte enthält nur diese Datei, keine
# Werkzeuge – deshalb braucht der Build für ARM64 keine Emulation.

# --- 1. Weboberfläche bauen ---
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/web
# Erst nur die Paketlisten: Solange sie sich nicht ändern, nimmt Docker die
# installierten Pakete aus dem Zwischenspeicher.
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# Ausgabe nach ../server/internal/web/dist/app (siehe vite.config.js)
RUN npm run build

# --- 2. Server für die Zielplattform übersetzen ---
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS server
# Zertifikate der Zertifizierungsstellen, damit der URL-Import
# HTTPS-Seiten prüfen kann. Werden in Stufe 3 übernommen.
RUN apk add --no-cache ca-certificates
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
COPY --from=web /src/server/internal/web/dist/app ./internal/web/dist/app
# TARGETOS/TARGETARCH setzt Docker passend zur gewünschten Plattform
# (hier linux/arm64). timetzdata bettet die Zeitzonendaten ein (ca. 450 KB),
# weil das leere Image keine mitbringt.
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -tags timetzdata -ldflags="-s -w" -o /out/rezeptplaner ./cmd/rezeptplaner

# --- 3. Fertiges Image: nur Programmdatei und Zertifikate ---
FROM scratch
COPY --from=server /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=server /out/rezeptplaner /rezeptplaner

# Verbindet das Paket auf ghcr.io mit dem Repository.
LABEL org.opencontainers.image.source="https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan"

# Port 8099 ist der Standard-Port für Ingress. /data ist der Datenordner
# der App (Teil der Home-Assistant-Backups). 172.30.32.2 ist der
# Ingress-Vermittler; nur er darf die App ansprechen.
ENV REZEPTPLANER_LISTEN_ADDR=":8099" \
    REZEPTPLANER_DATA_DIR="/data" \
    REZEPTPLANER_TRUSTED_PROXY="172.30.32.2"

ENTRYPOINT ["/rezeptplaner"]
