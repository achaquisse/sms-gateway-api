package rest

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

func ServeOpenAPISpec(c *fiber.Ctx) error {
	specPath := "openapi.yml"

	content, err := os.ReadFile(specPath)
	if err != nil {
		return ReturnInternalError(c, "Failed to read OpenAPI specification")
	}

	c.Set("Content-Type", "application/x-yaml")
	return c.Send(content)
}

func SetupSwagger(app *fiber.App) {
	app.Get("/api/openapi.yaml", ServeOpenAPISpec)

	app.Get("/api/docs/*", swagger.New(swagger.Config{
		URL:          "/api/openapi.yaml",
		DeepLinking:  true,
		DocExpansion: "list",
		Title:        "Omniscience API Documentation",
	}))
}
