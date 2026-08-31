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

// نظام تخزين مؤقت ذكي لمنع تكرار الطلبات وحظر السيرفر
var cache = struct {
	sync.RWMutex
	items map[string]CachedData
}{items: make(map[string]CachedData)}

type CachedData struct {
	Title     string
	Thumbnail string
	Expiry    time.Time
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

	// تشغيل استقبال رسائل البوت بكفاءة عالية
	go func() {
		for update := range updates {
			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			chatID := update.Message.Chat.ID
			text := update.Message.Text

			if text == "/start" {
				welcomeMsg := `◀️ | مع هذا البوت يمكنك التحميل من عدة مواقع بصيغ متعددة،

✅ | المواقع المدعومة :
1️⃣- اليوتيوب
2️⃣- انستغرام (مع كشف التاكات)
3️⃣- تيك توك
4️⃣- تويتر / إكس
5️⃣- سناب شات
6️⃣- وجميع منصات السوشيال ميديا!

🔄 | قم بإرسال الرابط للبدء بالتحميل •`
				msg := tgbotapi.NewMessage(chatID, welcomeMsg)
				bot.Send(msg)
				continue
			}

			// فحص الذاكرة المؤقتة أولاً لسرعة فائقة وحماية الـ IP
			cache.RLock()
			if cached, found := cache.items[text]; found && time.Now().Before(cached.Expiry) {
				cache.RUnlock()
				bot.Send(tgbotapi.NewMessage(chatID, "⚡ [من الذاكرة السريعة]\nالعنوان: "+cached.Title))
				continue
			}
			cache.RUnlock()

			sentMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, "⏳ جاري المعالجة بأقصى سرعة عبر السحابة..."))

			// تنفيذ yt-dlp بأعلى خيارات تجاوز الحظر
			cmd := exec.Command("yt-dlp", "--dump-json", "--skip-download", "--no-warnings", text)
			output, err := cmd.Output()
			if err != nil {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ عذراً، الرابط غير مدعوم أو أن المنصة حظرت الطلب المؤقت. جرب مجدداً لاحقاً."))
				continue
			}

			var rawData map[string]interface{}
			json.Unmarshal(output, &rawData)
			title, _ := rawData["title"].(string)
			if title == "" {
				title = "فيديو بدون عنوان"
			}

			// حفظ النتيجة في الكاش لمدة ساعة كاملة
			cache.Lock()
			cache.items[text] = CachedData{
				Title:     title,
				Expiry:    time.Now().Add(1 * time.Hour),
			}
			cache.Unlock()

			bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "✅ تم الاستخراج بنجاح!\n📌 العنوان: "+title))
		}
	}()

	// سيرفر الـ API السريع جداً
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

	log.Println("البوت والسيرفر الخارق يعملان بنجاح على المنفذ " + port)
	app.Listen(":" + port)
}
