
package controllers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func PublicPing(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
