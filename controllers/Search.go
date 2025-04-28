package controllers

import (
    "net/http"
    

    "github.com/gin-gonic/gin"
    "arttoy-hub/services" // <- เรียกใช้ Service
)

func SearchProducts(c *gin.Context) {
    keyword := c.Query("keyword")
    
    if keyword == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Keyword is required"})
        return
    }

    products, err := services.SearchProductsService(keyword)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, products)
}
