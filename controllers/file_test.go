package controllers

import (
	"auto-myself-api/app"
	"fmt"

	"github.com/gin-gonic/gin"
)

func TestFileGet(c *gin.Context, a *app.App) {
	fmt.Println("Hey")
}

func TestFileDelete(c *gin.Context, a *app.App) {
	fmt.Println("Hey")
}

func TestFilesList(c *gin.Context, a *app.App) {
	fmt.Println("Hey")
}

func TestFileUpload(c *gin.Context, a *app.App) {
	fmt.Println("Hey")
}
