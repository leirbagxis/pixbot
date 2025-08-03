package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/fonini/go-pix/pix"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Error loading BOT_TOKEN")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(botToken, opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "pix", bot.MatchTypeCommand, pixHandler)

	fmt.Println("Bot Online")
	b.Start(ctx)
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}

func pixHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Uso correto: /pixx [chave] [valor]",
		})
		return
	}

	chave := args[1]
	var valor float64
	var err error

	if len(args) >= 3 {
		valor, err = strconv.ParseFloat(strings.ReplaceAll(args[2], ",", "."), 64)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Valor inválido. Use ponto ou vírgula como decimal (ex: 20.50).",
			})
			return
		}
	}

	options := pix.Options{
		Key:  chave,
		Name: "Recebedor",
		City: "Recife",
	}
	if valor > 0 {
		options.Amount = valor
	}

	pixCode, err := pix.Pix(options)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Erro ao gerar código Pix: %v", err),
		})
		return
	}

	qrOptions := pix.QRCodeOptions{
		Size:    256,
		Content: pixCode,
	}
	qrCodePng, err := pix.QRCode(qrOptions)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Erro ao gerar QRCode: %v", err),
		})
		return
	}

	photo := &models.InputFileUpload{
		Filename: "pix_qrcode.png",
		Data:     bytes.NewReader(qrCodePng),
	}

	caption := fmt.Sprintf(
		"💸 <b>Copia e cola do Pix:</b>\n<code>%s</code>\n\nClique e segure para copiar!\n",
		escapeHTML(pixCode),
	)
	if valor > 0 {
		caption += fmt.Sprintf("Valor: <b>R$ %.2f</b>\n", valor)
	}
	caption += "Use o código acima no app do seu banco para pagar 👍"

	_, err = b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:    update.Message.Chat.ID,
		Photo:     photo,
		Caption:   caption,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Printf("Erro ao enviar foto: %v", err)
	}
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Envie /pix [chave] [valor] para gerar um QRCode Pix com copia e cola.",
	})
}
