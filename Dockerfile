ARG NODE_VERSION=24
ARG ALPINE_VERSION=3.22
ARG GO_VERSION=1.24

FROM node:${NODE_VERSION}-alpine${ALPINE_VERSION} as nodebuild
WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml tailwind.config.js ./
RUN pnpm install --frozen-lockfile

COPY templates ./templates
RUN pnpm run css:build

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as gobuild
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o unibocalendar -v

FROM alpine
WORKDIR /app
COPY --from=nodebuild /app/static ./static
COPY --from=gobuild /app/unibocalendar .
COPY templates/*.gohtml ./templates/

ENV PORT=8080
EXPOSE 8080

ENV GIN_MODE=release
VOLUME /app/data

LABEL org.opencontainers.image.source="https://github.com/VaiTon/unibocalendar"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.description="Calendario per i corsi Unibo V2"

CMD ["./unibocalendar"]
