package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
)

type ApifyRunResponse struct {
	Data struct {
		ID               string `json:"id"`
		DefaultDatasetID string `json:"defaultDatasetId"`
		Status           string `json:"status"`
	} `json:"data"`
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	apifyToken := os.Getenv("APIFY_TOKEN")

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
				bot.Send(tgbotapi.NewMessage(chatID, "⚡ أهلاً بك في بوت التحميل الخارق! أرسل أي رابط لنبدأ المعالجة الفورية."))
				continue
			}

			sentMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, "🚀 **جاري المعالجة بسرعة صاروخية...**"))

			apiURL := "https://api.apify.com/v2/actors/easyapi~all-in-one-media-downloader/runs?token=" + apifyToken
			inputPayload := map[string]interface{}{
				"link": text,
			}
			jsonBody, _ := json.Marshal(inputPayload)

			resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ خطأ في الاتصال بالخدمة."))
				continue
			}

			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ رفضت المنصة الطلب."))
				continue
			}

			var runResp ApifyRunResponse
			json.Unmarshal(bodyBytes, &runResp)
			runID := runResp.Data.ID
			datasetID := runResp.Data.DefaultDatasetID

			if runID == "" || datasetID == "" {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "❌ لم يتم استلام معرف التشغيل بنجاح."))
				continue
			}

			// نظام التحقق الخارق (Polling) كل ثانيتين لمعرفة هل انتهى التحميل أم لا
			var mediaLink string
			for i := 0; i < 15; i++ { // يحاول لمدة تصل إلى 30 ثانية كحد أقصى
				time.Sleep(2 * time.Second)

				statusURL := "https://api.apify.com/v2/actor-runs/" + runID + "?token=" + apifyToken
				statusResp, err := http.Get(statusURL)
				if err != nil {
					continue
				}
				statusBytes, _ := io.ReadAll(statusResp.Body)
				statusResp.Body.Close()

				var statusObj ApifyRunResponse
				json.Unmarshal(statusBytes, &statusObj)

				// إذا اكتمل التحميل بنجاح في السحابة
				if statusObj.Data.Status == "SUCCEEDED" {
					datasetURL := "https://api.apify.com/v2/datasets/" + datasetID + "/items?token=" + apifyToken
					getResp, err := http.Get(datasetURL)
					if err != nil {
						continue
					}
					dataBytes, _ := io.ReadAll(getResp.Body)
					getResp.Body.Close()

					var items []map[string]interface{}
					json.Unmarshal(dataBytes, &items)

					if len(items) > 0 {
						item := items[0]
						if val, ok := item["videoUrl"].(string); ok && val != "" {
							mediaLink = val
						} else if val, ok := item["url"].(string); ok && val != "" {
							mediaLink = val
						} else if val, ok := item["downloadUrl"].(string); ok && val != "" {
							mediaLink = val
						}
					}
					break
				} else if statusObj.Data.Status == "FAILED" || statusObj.Data.Status == "ABORTED" {
					break
				}
			}

			if mediaLink != "" {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "🔥 **تم التحميل بنجاح وبسرعة فائقة!**\n\n🔗 الرابط المباشر للملف:\n"+mediaLink))
			} else {
				bot.Send(tgbotapi.NewEditMessageText(chatID, sentMsg.MessageID, "⚠️ استغرقت العملية وقتاً أطول من المعتاد أو فشل استخراج الرابط، جرب مجدداً."))
			}
		}
	}()

	app := fiber.New()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app.Listen(":" + port)
}
