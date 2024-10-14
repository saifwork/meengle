package responses

type ResponseDto struct {
	Success bool              `json:"success"`
	Error   *ErrorResponseDto `json:"error,omitempty"`
	Data    any               `json:"data"`
}

type ErrorResponseDto struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewSuccessResponse(data any) *ResponseDto {
	return &ResponseDto{
		Success: true,
		Error:   nil,
		Data:    data,
	}
}

func NewErrorResponse(code int, msg string, data any) *ResponseDto {
	return &ResponseDto{
		Success: false,
		Error: &ErrorResponseDto{
			Code:    code,
			Message: msg,
		},
		Data: data,
	}
}
