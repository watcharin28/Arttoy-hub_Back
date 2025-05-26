package controllers

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "arttoy-hub/services" // <- เรียกใช้ Service
)

func SearchProducts(c *gin.Context) {
	keyword := c.Query("keyword")
	categoryParam := c.Query("category")

	var categoryList []string
	if categoryParam != "" {
		categoryList = strings.Split(categoryParam, ",") // แยกหมวดหมู่จาก comma
	}

	products, err := services.SearchProductsService(keyword, categoryList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}
