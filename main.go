package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
)

var cache = struct {
	sync.RWMutex
	items map[string]MediaInfo
}{items: make(map[string]MediaInfo)}

type MediaInfo struct {
	Title  string
	Expiry time.Time
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("يرجى ضبط متغير TELEGRAM_BOT_TOKEN في إعدادات Railway")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
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
				welcomeMsg := `⚡ **البوت الخارق للتحميل السحابي المتعدد** ⚡

أهلاً بك! أنا نظام تحميل سحابي متطور يعمل بأعلى سرعة مدعوم بلغة Go وخوادم سحابية مؤمنة.

🔄 **قم بإرسال الرابط المطلوب الآن للبدء!**`
				msg := tgbotapi.NewMessage(chatID, welcomeMsg)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
				continue
			}

			cache.RLock()
			if cached, found := cache.items[text]; found && time.Now().Before(cached.Expiry) {
				cache.RUnlock()
				bot.Send(tgbotapi.NewMessage(chatID, "⚡ [استجابة فورية من السيرفر الخارق]\n📌 العنوان: "+cached.Title))
				continue
			}
			cache.RUnlock()

			sentMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, "🚀 جاري التحليل السحابي واستخراج الروابط..."))

			cmd := exec.Command("yt-dlp", "--dump-json", "--no-warnings", text)
			output, err := cmd.Output()
			if err != nil {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ عذراً، الرابط غير مدعوم أو أن المنصة فرضت قيوداً. حاول مجدداً."))
				continue
			}

			var rawData map[string]interface{}
			json.Unmarshal(output, &rawData)
			title, _ := rawData["title"].(string)
			if title == "" {
				title = "محتوى رقمي"
			}

			cache.Lock()
			cache.items[text] = MediaInfo{
				Title:  title,
				Expiry: time.Now().Add(2 * time.Hour),
			}
			cache.Unlock()

			bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "🔥 **تم الاستخراج بنجاح تام!**\n\n📌 **العنوان:** "+title))
		}
	}()

	app := fiber.New()
	app.Get("/download", func(c *fiber.Ctx) error {
		targetURL := c.Query("url")
		if targetURL == "" {
			return c.Status(400).JSON(fiber.Map{"error": "الرجاء إرسال الرابط المطلوب"})
		}
		return c.JSON(fiber.Map{"status": "success", "url": targetURL})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("النظام الخارق يعمل على المنفذ " + port)
	app.Listen(":" + port)
}
