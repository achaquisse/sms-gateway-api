package main

import (
	"log"
	"sms-gateway-api/db"
	"sms-gateway-api/rest"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	if err := db.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database successfully")

	if err := db.InitSchema(); err != nil {
		log.Printf("Warning: Failed to initialize schema: %v", err)
	}

	if err := db.RunMigrations(); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}

	version, err := db.GetCurrentVersion()
	if err != nil {
		log.Printf("Warning: Failed to get current schema version: %v", err)
	} else {
		log.Printf("Database schema version: %d", version)
	}

	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Device-Key",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	rest.Init(app)

	log.Println("Starting server on :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
