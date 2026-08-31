FROM golang:1.21-alpine

# تثبيت الحزم المطلوبة وتحديث yt-dlp لآخر إصدار لتجنب مشاكل الحظر
RUN apk add --no-cache python3 py3-pip ffmpeg curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o main main.go

EXPOSE 8080

CMD ["./main"]
