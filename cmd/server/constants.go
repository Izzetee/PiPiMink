// Package server provides the HTTP server and API routes
package server

// Constants for API rate limiting
const (
	// DefaultRateLimit defines the default requests per minute
	DefaultRateLimit = 60

	// MaxRequestBodySize is the maximum size of a request body in bytes (1MB)
	MaxRequestBodySize = 1024 * 1024

	// DefaultTimeout is the default timeout for API requests in milliseconds
	DefaultTimeout = 30000

	// DefaultTokenApproximationFactor is used to estimate token counts from character counts
	DefaultTokenApproximationFactor = 4
)

// HTTP status text constants
const (
	// StatusMessageSuccess is the standard success message
	StatusMessageSuccess = "success"

	// StatusMessageError is the standard error message
	StatusMessageError = "error"
)

// Default model constants
const (
	// DefaultModel is used when no model is specified
	DefaultModel = "gpt-4-turbo"

	// DefaultDisabledModelMessage is returned when a disabled model is requested
	DefaultDisabledModelMessage = "This model is currently disabled"

	// DefaultUnavailableSourceMessage is returned when an invalid source is requested
	DefaultUnavailableSourceMessage = "Unknown or unavailable source"
)

// Source identifier constants
const (
	// OpenAISource is the identifier for OpenAI models
	OpenAISource = "openai"

	// LocalSource is the identifier for local models
	LocalSource = "local"

	// AzureKIFoundrySource is the identifier for AzureKIFoundry models
	AzureKIFoundrySource = "azurekifoundry"
)
