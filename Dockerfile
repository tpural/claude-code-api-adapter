# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/claude-code-api-adapter ./cmd/claude-code-api-adapter

# Runtime stage
FROM node:22-alpine

RUN npm install -g @anthropic-ai/claude-code && \
    npm cache clean --force

COPY --from=builder /bin/claude-code-api-adapter /usr/local/bin/claude-code-api-adapter

RUN mkdir -p /data/sessions
ENV SESSIONS_DIR=/data/sessions
ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["claude-code-api-adapter"]
