package controllers

import (
    "arttoy-hub/models"
    "github.com/gin-gonic/gin"
    "net/http"
)

func AddCategory(c *gin.Context) {
    var input struct {
        Name string `json:"name" binding:"required"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing category name"})
        return
    }

    category, err := models.AddCategory(input.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, category)
}

func GetAllCategories(c *gin.Context) {
    categories, err := models.GetAllCategories()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, categories)
}
