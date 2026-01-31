package rest

type DeviceConfigRequest struct {
	Topics []string `json:"topics" validate:"required"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}
