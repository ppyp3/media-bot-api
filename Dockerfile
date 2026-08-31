FROM golang:1.21-alpine

# تثبيت الحزم المطلوبة
RUN apk add --no-cache python3 py3-pip ffmpeg curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

# نسخ ملفات المشروع أولاً
COPY . .

# ترتيب وتحديث الاعتماديات تلقائياً داخل الحاوية
RUN go mod tidy

# بناء المشروع
RUN go build -o main main.go

EXPOSE 8080

CMD ["./main"]
