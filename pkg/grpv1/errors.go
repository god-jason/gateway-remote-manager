package grpv1

import "time"

// 统一错误码（对应设计文档第 15 节）
const (
	ErrOK                    uint32 = 0
	ErrInvalidMessage        uint32 = 1000
	ErrUnsupportedVersion    uint32 = 1001
	ErrUnsupportedType       uint32 = 1002
	ErrInvalidArgument       uint32 = 1003
	ErrUnauthorized          uint32 = 1004
	ErrForbidden             uint32 = 1005
	ErrDeviceNotFound        uint32 = 1006
	ErrDeviceOffline         uint32 = 1007
	ErrSessionNotFound       uint32 = 1008
	ErrSessionLimit          uint32 = 1009
	ErrTaskNotFound          uint32 = 1010
	ErrTaskTimeout           uint32 = 1011
	ErrTaskConflict          uint32 = 1012
	ErrFileNotFound          uint32 = 1013
	ErrFilePermissionDenied  uint32 = 1014
	ErrFileChecksumFailed    uint32 = 1015
	ErrStreamFlowControl     uint32 = 1016
	ErrRateLimited           uint32 = 1017
	ErrInternalError         uint32 = 1018
	ErrDeviceDisabled        uint32 = 1019
	ErrDeviceAuthFailed      uint32 = 1020
)

var errText = map[uint32]string{
	ErrInvalidMessage:       "INVALID_MESSAGE",
	ErrUnsupportedVersion:   "UNSUPPORTED_VERSION",
	ErrUnsupportedType:      "UNSUPPORTED_TYPE",
	ErrInvalidArgument:      "INVALID_ARGUMENT",
	ErrUnauthorized:         "UNAUTHORIZED",
	ErrForbidden:            "FORBIDDEN",
	ErrDeviceNotFound:       "DEVICE_NOT_FOUND",
	ErrDeviceOffline:        "DEVICE_OFFLINE",
	ErrSessionNotFound:      "SESSION_NOT_FOUND",
	ErrSessionLimit:         "SESSION_LIMIT",
	ErrTaskNotFound:         "TASK_NOT_FOUND",
	ErrTaskTimeout:          "TASK_TIMEOUT",
	ErrTaskConflict:         "TASK_CONFLICT",
	ErrFileNotFound:         "FILE_NOT_FOUND",
	ErrFilePermissionDenied: "FILE_PERMISSION_DENIED",
	ErrFileChecksumFailed:   "FILE_CHECKSUM_FAILED",
	ErrStreamFlowControl:    "STREAM_FLOW_CONTROL",
	ErrRateLimited:          "RATE_LIMITED",
	ErrInternalError:        "INTERNAL_ERROR",
	ErrDeviceDisabled:       "DEVICE_DISABLED",
	ErrDeviceAuthFailed:     "DEVICE_AUTH_FAILED",
}

// CodeText 错误码名称
func CodeText(code uint32) string {
	if t, ok := errText[code]; ok {
		return t
	}
	return "UNKNOWN"
}

// GrpError 协议错误
type GrpError struct {
	Code      uint32
	Message   string
	Detail    string
	Retryable bool
	RequestID string
}

func (e *GrpError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return CodeText(e.Code)
}

// NewError 构造错误
func NewError(code uint32, msg string) *GrpError {
	return &GrpError{Code: code, Message: msg}
}

// 时间源，便于测试注入
var currentTime = time.Now