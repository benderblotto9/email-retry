# ── Build stage ──────────────────────────────────────────
FROM registry.fedoraproject.org/fedora:42 AS builder

RUN dnf install -y golang gcc sqlite-devel && dnf clean all

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags "-s -w" -o /email-retry ./src

# ── Runtime stage ────────────────────────────────────────
FROM registry.fedoraproject.org/fedora-minimal:42

RUN microdnf install -y sqlite-libs && microdnf clean all

RUN useradd -r -s /sbin/nologin retrybot
USER retrybot
WORKDIR /home/retrybot

COPY --from=builder /email-retry /usr/local/bin/email-retry

ENTRYPOINT ["email-retry"]
CMD ["--config", "/home/retrybot/config.yaml"]
