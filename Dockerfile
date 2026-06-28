FROM golang:1.25.4-bookworm AS builder

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gcc \
    zlib1g-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOTDBOT_VERSION=$(grep 'github.com/AshokShau/gotdbot' go.mod | awk '{print $2}') \
    && go run github.com/AshokShau/gotdbot/scripts/tools@${GOTDBOT_VERSION}
RUN go run setup_ntgcalls.go

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .

FROM debian:12-slim AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    wget \
    unzip \
    curl \
    lsb-release \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN wget -O /usr/local/bin/yt-dlp \
    https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/yt-dlp_linux \
    && chmod +x /usr/local/bin/yt-dlp

RUN mkdir -p /opt/deno/bin \
    && wget -O /tmp/deno.zip \
        https://github.com/denoland/deno/releases/latest/download/deno-x86_64-unknown-linux-gnu.zip \
    && unzip /tmp/deno.zip -d /opt/deno/bin \
    && chmod +x /opt/deno/bin/deno \
    && ln -sf /opt/deno/bin/deno /usr/local/bin/deno \
    && rm /tmp/deno.zip

# Même UID/GID que le premier user Linux du VPS (souvent 1000) pour que le bind-mount
# des cookies (~/yt-dlp/cookies.txt) soit lisible/écrivable sans chmod 666.
ARG APP_UID=1000
ARG APP_GID=1000
RUN groupadd -g "${APP_GID}" app && useradd -u "${APP_UID}" -g app -m -d /home/app app

ENV DENO_INSTALL="/opt/deno"
ENV PATH="${DENO_INSTALL}/bin:${PATH}"
ENV HOME="/home/app"

COPY --from=builder --chown=app:app /app/main /usr/local/bin/app
COPY --from=builder --chown=app:app /app/libtdjson.so.* /home/app/

RUN chown -R app:app /opt/deno

USER app

WORKDIR /home/app
ENTRYPOINT ["app"]
