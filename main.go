package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
)

type Media struct {
	URL       string `json:"url"`
	Quality   string `json:"quality"`
	Extension string `json:"extension"`
	Type      string `json:"type"`
}

func main() {
	// الحصول على التوكن من متغيرات البيئة (أماناً عند الرفع لـ GitHub) أو وضعه مباشرة
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		botToken = "ضع_التوكن_هنا_للاختبار_المحلي"
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// تشغيل البوت في الخلفية
	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID
			text := update.Message.Text

			if text == "/start" {
				msg := tgbotapi.NewMessage(chatID, "أهلاً بك! أرسل أي رابط وسأقوم باستخراج روابط التحميل لك فوراً.")
				bot.Send(msg)
				continue
			}

			msg := tgbotapi.NewMessage(chatID, "جاري معالجة الرابط...")
			sentMsg, _ := bot.Send(msg)

			cmd := exec.Command("yt-dlp", "--dump-json", "--skip-download", text)
			output, err := cmd.Output()
			if err != nil {
				edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "عذراً، فشل في معالجة الرابط.")
				bot.Send(edit)
				continue
			}

			var rawData map[string]interface{}
			json.Unmarshal(output, &rawData)
			title, _ := rawData["title"].(string)

			edit := tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "تم الاستخراج بنجاح!\nالعنوان: "+title)
			bot.Send(edit)
		}
	}()

	// تشغيل الـ API
	app := fiber.New()

	app.Get("/download", func(c *fiber.Ctx) error {
		targetURL := c.Query("url")
		if targetURL == "" {
			return c.Status(400).JSON(fiber.Map{"error": "الرجاء إرسال الرابط"})
		}
		return c.JSON(fiber.Map{"status": "success", "url": targetURL})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("يعمل السيرفر والبوت بنجاح على المنفذ " + port)
	app.Listen(":" + port)
}
