package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
)

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	apifyToken := os.Getenv("APIFY_TOKEN") // سيتم سحبه من إعدادات Railway بأمان

	if botToken == "" {
		log.Fatal("يرجى ضبط TELEGRAM_BOT_TOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	go func() {
		for update := range updates {
			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			chatID := update.Message.Chat.ID
			text := update.Message.Text

			if text == "/start" {
				bot.Send(tgbotapi.NewMessage(chatID, "⚡ أهلاً بك! أرسل أي رابط للتحميل السحابي الفوري."))
				continue
			}

			sentMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, "🔄 جاري معالجة الطلب..."))

			apiURL := "https://api.apify.com/v2/actors/easyapi~all-in-one-media-downloader/runs?token=" + apifyToken

			inputPayload := map[string]interface{}{
				"link": text,
			}
			jsonBody, _ := json.Marshal(inputPayload)

			resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
			if err != nil || resp.StatusCode != http.StatusCreated {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ فشل الاتصال بخدمة التحميل."))
				continue
			}
			defer resp.Body.Close()

			bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "🔥 **تم بدء معالجة الرابط في السحابة بنجاح!**"))
		}
	}()

	app := fiber.New()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(":" + port)
}
