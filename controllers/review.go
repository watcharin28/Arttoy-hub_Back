// controllers/review_controller.go
package controllers

import (
    "context"
    "net/http"
    "time"
    "arttoy-hub/database"
    "arttoy-hub/models"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateReview(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		ProductID string `json:"product_id" binding:"required"`
		Rating    int    `json:"rating" binding:"required"`
		Comment   string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userObjID, _ := primitive.ObjectIDFromHex(userID)
	productObjID, err := primitive.ObjectIDFromHex(input.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//  ตรวจสอบว่าเคยซื้อสินค้านี้ใน order ที่ completed แล้ว
	orderFilter := bson.M{
		"user_id": userObjID,
		"items.product_id": productObjID,
		"status": "completed",
	}
	count, err := db.OpenCollection("orders").CountDocuments(ctx, orderFilter)
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "คุณสามารถรีวิวได้เฉพาะสินค้าที่คุณเคยซื้อและได้รับแล้ว"})
		return
	}

	//  ตรวจสอบว่าผู้ใช้รีวิวสินค้านี้ไปแล้วหรือยัง
	reviewFilter := bson.M{
		"user_id":    userObjID,
		"product_id": productObjID,
	}
	existingReview := db.OpenCollection("reviews").FindOne(ctx, reviewFilter)
	if existingReview.Err() == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "คุณได้รีวิวสินค้านี้ไปแล้ว"})
		return
	}

	//  ดึงข้อมูลสินค้าเพื่อนำ seller_id มาเก็บใน review
	var product models.Product
	err = db.ProductCollection.FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	//  สร้างและบันทึก review
	review := models.Review{
		ProductID: productObjID,
		SellerID:  product.SellerID,
		UserID:    userObjID,
		Rating:    input.Rating,
		Comment:   input.Comment,
		CreatedAt: time.Now(),
	}

	_, err = db.OpenCollection("reviews").InsertOne(ctx, review)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review submitted successfully"})
}


func GetReviewsBySeller(c *gin.Context) {
    sellerID := c.Param("sellerId")
    sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cursor, err := db.OpenCollection("reviews").Find(ctx, bson.M{"seller_id": sellerObjID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
        return
    }

    var reviews []models.Review
    if err := cursor.All(ctx, &reviews); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse reviews"})
        return
    }

    c.JSON(http.StatusOK, reviews)
}

func GetMyReviews(c *gin.Context) {
	userID := c.GetString("user_id")
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.OpenCollection("reviews").Find(ctx, bson.M{"user_id": userObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	var reviews []models.Review
	if err := cursor.All(ctx, &reviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse reviews"})
		return
	}

	c.JSON(http.StatusOK, reviews)
}