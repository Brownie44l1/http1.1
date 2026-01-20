package response

// Additional HTTP status codes for comprehensive support

const (
	// 1xx Informational
	StatusContinue           StatusCode = 100
	StatusSwitchingProtocols StatusCode = 101

	// 2xx Success
	StatusAccepted             StatusCode = 202
	StatusNonAuthoritativeInfo StatusCode = 203
	StatusResetContent         StatusCode = 205
	StatusPartialContent       StatusCode = 206

	// 3xx Redirection
	StatusMultipleChoices   StatusCode = 300
	StatusMovedPermanently  StatusCode = 301
	StatusFound             StatusCode = 302
	StatusSeeOther          StatusCode = 303
	StatusNotModified       StatusCode = 304
	StatusUseProxy          StatusCode = 305
	StatusTemporaryRedirect StatusCode = 307
	StatusPermanentRedirect StatusCode = 308

	// 4xx Client Error
	StatusUnauthorized                 StatusCode = 401
	StatusPaymentRequired              StatusCode = 402
	StatusForbidden                    StatusCode = 403
	StatusMethodNotAllowed             StatusCode = 405
	StatusNotAcceptable                StatusCode = 406
	StatusProxyAuthRequired            StatusCode = 407
	StatusRequestTimeout               StatusCode = 408
	StatusConflict                     StatusCode = 409
	StatusGone                         StatusCode = 410
	StatusLengthRequired               StatusCode = 411
	StatusPreconditionFailed           StatusCode = 412
	StatusRequestEntityTooLarge        StatusCode = 413
	StatusRequestURITooLong            StatusCode = 414
	StatusUnsupportedMediaType         StatusCode = 415
	StatusRequestedRangeNotSatisfiable StatusCode = 416
	StatusExpectationFailed            StatusCode = 417
	StatusTeapot                       StatusCode = 418 // RFC 2324
	StatusUnprocessableEntity          StatusCode = 422
	StatusTooManyRequests              StatusCode = 429

	// 5xx Server Error
	StatusNotImplemented          StatusCode = 501
	StatusBadGateway              StatusCode = 502
	StatusServiceUnavailable      StatusCode = 503
	StatusGatewayTimeout          StatusCode = 504
	StatusHTTPVersionNotSupported StatusCode = 505
)

func init() {
	// Extend statusText map with additional codes
	additionalStatusText := map[StatusCode]string{
		// 1xx
		StatusContinue:           "Continue",
		StatusSwitchingProtocols: "Switching Protocols",

		// 2xx
		StatusAccepted:             "Accepted",
		StatusNonAuthoritativeInfo: "Non-Authoritative Information",
		StatusResetContent:         "Reset Content",
		StatusPartialContent:       "Partial Content",

		// 3xx
		StatusMultipleChoices:   "Multiple Choices",
		StatusMovedPermanently:  "Moved Permanently",
		StatusFound:             "Found",
		StatusSeeOther:          "See Other",
		StatusNotModified:       "Not Modified",
		StatusUseProxy:          "Use Proxy",
		StatusTemporaryRedirect: "Temporary Redirect",
		StatusPermanentRedirect: "Permanent Redirect",

		// 4xx
		StatusUnauthorized:                 "Unauthorized",
		StatusPaymentRequired:              "Payment Required",
		StatusForbidden:                    "Forbidden",
		StatusMethodNotAllowed:             "Method Not Allowed",
		StatusNotAcceptable:                "Not Acceptable",
		StatusProxyAuthRequired:            "Proxy Authentication Required",
		StatusRequestTimeout:               "Request Timeout",
		StatusConflict:                     "Conflict",
		StatusGone:                         "Gone",
		StatusLengthRequired:               "Length Required",
		StatusPreconditionFailed:           "Precondition Failed",
		StatusRequestEntityTooLarge:        "Request Entity Too Large",
		StatusRequestURITooLong:            "Request URI Too Long",
		StatusUnsupportedMediaType:         "Unsupported Media Type",
		StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
		StatusExpectationFailed:            "Expectation Failed",
		StatusTeapot:                       "I'm a teapot",
		StatusUnprocessableEntity:          "Unprocessable Entity",
		StatusTooManyRequests:              "Too Many Requests",

		// 5xx
		StatusNotImplemented:          "Not Implemented",
		StatusBadGateway:              "Bad Gateway",
		StatusServiceUnavailable:      "Service Unavailable",
		StatusGatewayTimeout:          "Gateway Timeout",
		StatusHTTPVersionNotSupported: "HTTP Version Not Supported",
	}

	for code, text := range additionalStatusText {
		statusText[code] = text
	}
}

// StatusText returns the text description for a status code
func StatusText(code StatusCode) string {
	if text, ok := statusText[code]; ok {
		return text
	}
	return "Unknown Status"
}

// IsInformational returns true for 1xx status codes
func (code StatusCode) IsInformational() bool {
	return code >= 100 && code < 200
}

// IsSuccess returns true for 2xx status codes
func (code StatusCode) IsSuccess() bool {
	return code >= 200 && code < 300
}

// IsRedirect returns true for 3xx status codes
func (code StatusCode) IsRedirect() bool {
	return code >= 300 && code < 400
}

// IsClientError returns true for 4xx status codes
func (code StatusCode) IsClientError() bool {
	return code >= 400 && code < 500
}

// IsServerError returns true for 5xx status codes
func (code StatusCode) IsServerError() bool {
	return code >= 500 && code < 600
}

// IsError returns true for 4xx or 5xx status codes
func (code StatusCode) IsError() bool {
	return code.IsClientError() || code.IsServerError()
}
