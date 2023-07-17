FROM node:current-alpine as nodebuild
WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml tailwind.config.js ./
RUN pnpm install --frozen-lockfile

COPY templates ./templates
RUN pnpm run css:build

FROM golang:alpine as gobuild
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o unibocalendar

FROM alpine
WORKDIR /app
COPY --from=nodebuild /app/static ./static
COPY --from=gobuild /app/unibocalendar .

ENV PORT=8080
EXPOSE 8080

ENV GIN_MODE=release

CMD ["./unibocalendar"]




