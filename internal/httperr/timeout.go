package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestTimeoutMessage is the canonical error text for an expired request deadline.
const RequestTimeoutMessage = "Request timeout"

// WriteRequestTimeout writes the canonical 408 body. The timeout middleware and handlers that
// abort a body read on context cancellation share it so both response paths are identical.
func WriteRequestTimeout(c *gin.Context) {
	c.JSON(http.StatusRequestTimeout, gin.H{"error": RequestTimeoutMessage})
}
