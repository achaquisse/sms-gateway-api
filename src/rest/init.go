package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func Init(app *fiber.App) {
	SetupSwagger(app)

	app.Post("/messages", QueueSMSHandler)
	app.Get("/messages", ListMessagesHandler)
	app.Get("/reports", GetReportsHandler)

	app.Put("/devices", UpdateDeviceTopicsHandler)

	app.Get("/gateway/poll", PollMessagesHandler)
	app.Put("/gateway/status/:messageId", UpdateMessageStatusHandler)

	log.Info("REST API started")
}
