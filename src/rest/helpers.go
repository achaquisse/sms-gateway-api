package rest

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func ParseUintQueryParam(c *fiber.Ctx, paramName string, required bool) (uint, error) {
	paramStr := c.Query(paramName)
	if paramStr == "" {
		if required {
			return 0, fmt.Errorf("%s parameter is required", paramName)
		}
		return 0, nil
	}

	parsed, err := strconv.ParseUint(paramStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s format", paramName)
	}

	return uint(parsed), nil
}

func ParseOptionalUintQueryParam(c *fiber.Ctx, paramName string) (*uint, error) {
	paramStr := c.Query(paramName)
	if paramStr == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(paramStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format", paramName)
	}

	val := uint(parsed)
	return &val, nil
}

func ParseDateQueryParam(c *fiber.Ctx, paramName string) (*time.Time, error) {
	dateStr := c.Query(paramName)
	if dateStr == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format. Use YYYY-MM-DD", paramName)
	}

	return &parsed, nil
}

func ValidateDateString(dateStr, paramName string) error {
	if dateStr == "" {
		return nil
	}

	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid %s format. Use YYYY-MM-DD", paramName)
	}

	return nil
}

func GetDateRangeWithDefaults(startDate, endDate string) (string, string) {
	now := time.Now()

	if startDate == "" {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}

	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	return startDate, endDate
}

func ReturnBadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": message,
	})
}

func ReturnNotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": message,
	})
}

func ReturnInternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": message,
	})
}
