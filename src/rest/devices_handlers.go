package rest

import (
	"sms-gateway-api/db"

	"github.com/gofiber/fiber/v2"
)

func UpdateDeviceTopicsHandler(c *fiber.Ctx) error {
	deviceKey := c.Get("X-Device-Key")
	if deviceKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing device key",
		})
	}

	device, err := db.GetDeviceByKey(deviceKey)
	if err != nil {
		return ReturnInternalError(c, "Failed to authenticate device")
	}

	if device == nil {
		device, err = db.CreateDevice(deviceKey, nil)
		if err != nil {
			return ReturnInternalError(c, "Failed to create device")
		}
	}

	var req DeviceConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return ReturnBadRequest(c, "Invalid request body")
	}

	if req.Topics == nil {
		return ReturnBadRequest(c, "topics is required")
	}

	if err := db.SetDeviceTopics(device.ID, req.Topics); err != nil {
		return ReturnInternalError(c, "Failed to update device topics")
	}

	response := SuccessResponse{
		Message: "Device configuration updated",
	}

	return c.JSON(response)
}
