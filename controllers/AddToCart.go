// 📁 controllers/cart_controller.go
package controllers

import (
    "arttoy-hub/models"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson/primitive"
	"fmt"
    "net/http"
    "time"
	
)

// เพิ่มสินค้าเข้าตะกร้า
func AddToCart(c *gin.Context) {
    var input struct {
        ProductID string `json:"product_id"`
        Quantity  int    `json:"quantity"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        fmt.Println("JSON binding error:", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    //  ป้องกัน quantity เป็น 0 หรือค่าติดลบ
    if input.Quantity <= 0 {
        fmt.Println("Invalid quantity:", input.Quantity)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be greater than 0"})
        return
    }

    userID := c.GetString("user_id")
    if userID == "" {
        fmt.Println("Unauthorized: user_id not found")
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    productObjID, err2 := primitive.ObjectIDFromHex(input.ProductID)
    if err != nil || err2 != nil {
        fmt.Println("Invalid ObjectIDs:", err, err2)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IDs"})
        return
    }

    cartItem := models.CartItem{
        ID:        primitive.NewObjectID(),
        UserID:    userObjID,
        ProductID: productObjID,
        Quantity:  input.Quantity,
        AddedAt:   time.Now(),
    }

    fmt.Printf("🛒 Attempting to insert cart item: %+v\n", cartItem)

    if err := models.AddToCart(cartItem); err != nil {
        fmt.Println("Failed to insert cart item:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to cart"})
        return
    }

    fmt.Println("✅ Cart item inserted successfully")
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
        fmt.Println("❌ ERROR จาก GetCartDetailsForUser:", err)
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
