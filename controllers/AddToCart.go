// 📁 controllers/cart_controller.go
package controllers

import (
    "arttoy-hub/models"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson/primitive"
	
    "net/http"
	
)

// เพิ่มสินค้าเข้าตะกร้า
func AddToCart(c *gin.Context) {
    var input struct {
        ProductID string `json:"product_id"`
        Quantity  int    `json:"quantity"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    productObjID, err2 := primitive.ObjectIDFromHex(input.ProductID)
    if err != nil || err2 != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IDs"})
        return
    }

    cartItem := models.CartItem{
        UserID:    userObjID,
        ProductID: productObjID,
        Quantity:  input.Quantity,
    }

    if err := models.AddToCart(cartItem); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to cart"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Added to cart"})
}

//  ดูตะกร้าของผู้ใช้
func GetCart(c *gin.Context) {
    userID := c.GetString("user_id") // ดึงจาก JWT หรือ middleware
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    items, err := models.GetCartDetailsForUser(userObjID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart items"})
        return
    }

    //  ป้องกันกรณีได้ null → ให้เป็น array ว่างแทน
    if items == nil {
        items = []models.CartItemWithProduct{}
    }

    c.JSON(http.StatusOK, gin.H{"cart": items})
}



// ลบสินค้าจากตะกร้า
func RemoveFromCart(c *gin.Context) {
    userID := c.GetString("user_id")
    productID := c.Param("product_id")

    if userID == "" || productID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing IDs"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    productObjID, err2 := primitive.ObjectIDFromHex(productID)
    if err != nil || err2 != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IDs"})
        return
    }

    if err := models.RemoveFromCart(userObjID, productObjID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Item removed from cart"})
}
