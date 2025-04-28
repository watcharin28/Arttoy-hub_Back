package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"arttoy-hub/models"
	"arttoy-hub/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func BecomeSeller(c *gin.Context) {
    var req models.BecomeSellerRequest

    // (0) ต้องรองรับ multipart/form-data
    if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
        return
    }

    // (1) ดึง userID จาก context
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ ID ผู้ใช้ใน context"})
        return
    }

    // (2) รับไฟล์ id_card_image
    file, _, err := c.Request.FormFile("id_card_image_url")  // แก้ไขจาก "id_card_image_url"
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upload image: " + err.Error()})
        return
    }

    // (3) อัปโหลดรูปบัตรประชาชนไปยัง GCS
    idCardImageURL, err := UploadImageToGCS(file, "image/jpeg", "id_card_image")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to GCS: " + err.Error()})
        return
    }

    // (4) ดึงข้อมูลฟอร์มทั่วไป
    req.FirstName = c.PostForm("first_name")
    req.LastName = c.PostForm("last_name")
    req.BankAccountName = c.PostForm("bank_account_name")
    req.BankName = c.PostForm("bank_name")
    req.BankAccountNumber = c.PostForm("bank_account_number")
    req.CitizenID = c.PostForm("citizen_id")
    req.IDCardImageURL = idCardImageURL

    // Validate ว่าข้อมูลที่จำเป็นต้องมี
    if req.FirstName == "" || req.LastName == "" || req.BankAccountName == "" ||
        req.BankName == "" || req.BankAccountNumber == "" || req.CitizenID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
        return
    }

    // (5) เตรียมข้อมูลลง MongoDB
    userIDObjectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
        return
    }

    sellerInfo := models.SellerInfo{
        FirstName:         req.FirstName,
        LastName:          req.LastName,
        BankAccountName:   req.BankAccountName,
        BankName:          req.BankName,
        BankAccountNumber: req.BankAccountNumber,
        CitizenID:         req.CitizenID,
        IDCardImageURL:    req.IDCardImageURL,
        IsVerified:        false,
    }

    // (6) อัปเดต User ใน MongoDB
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    userCollection := db.OpenCollection("users")
    update := bson.M{
        "$set": bson.M{
            "seller_info": sellerInfo,
            "is_seller":   true,
        },
    }

    result, err := userCollection.UpdateOne(ctx, bson.M{"_id": userIDObjectID}, update)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
        return
    }

    if result.MatchedCount == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Become seller request submitted successfully!"})
}

