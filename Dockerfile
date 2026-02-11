FROM golang:1.25.6-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: 静的リンクバイナリの生成
# -ldflags="-w -s": デバッグ情報を削除してサイズを削減
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
	-ldflags="-w -s" \
	-o cargohold \
	./cmd/cargohold/main.go

FROM alpine:3.23

LABEL description="cargohold - Git LFS Server"

WORKDIR /app

# ca-certificates: HTTPS通信に必要
# tzdata: タイムゾーン設定に必要
# wget: ヘルスチェックに必要
RUN apk add --no-cache ca-certificates tzdata wget && \
	addgroup -g 1000 app && \
	adduser -D -u 1000 -G app app

ENV TZ=Asia/Tokyo

COPY --from=builder /build/cargohold /app/cargohold

RUN mkdir -p /app/config && \
	chown -R app:app /app

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/healthz || exit 1

CMD ["/app/cargohold"]
