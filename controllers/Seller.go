package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"os"
	"fmt"
	"time"
)

func BecomeSeller(c *gin.Context) {
	var req models.BecomeSellerRequest

	// รองรับ multipart/form-data
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		fmt.Println("❌ ParseMultipartForm failed:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		fmt.Println("❌ No user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ไม่พบ ID ผู้ใช้ใน context"})
		return
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// รับไฟล์บัตรประชาชน
	file, _, err := c.Request.FormFile("id_card_image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upload image: " + err.Error()})
		return
	}

	// อัปโหลดไป GCS
	idCardImageURL, err := UploadImageToGCS(file, "image/jpeg", "id_card_image")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to GCS: " + err.Error()})
		return
	}

	// ดึงข้อมูลจากฟอร์ม
	fmt.Println("✅ ParseMultipartForm ผ่านแล้ว")
	req.FirstName = c.PostForm("first_name")
	req.LastName = c.PostForm("last_name")
	fmt.Println("✅ First Name:", req.FirstName)
	req.BankAccountName = c.PostForm("bank_account_name")
	req.BankName = c.PostForm("bank_name")
	fmt.Println("✅ bank_name:", req.BankName)
	req.BankAccountNumber = c.PostForm("bank_account_number")
	req.CitizenID = c.PostForm("citizen_id")
	req.IDCardImageURL = idCardImageURL
	// สร้าง map ธนาคารให้ตรงกับ Omise
	var bankMap = map[string]string{
		"กสิกรไทย":   "kbank",
		"ไทยพาณิชย์": "scb",
		"กรุงเทพ":    "bbl",
		"กรุงศรี":    "bay",
		"กรุงไทย":    "ktb",
		// เพิ่มเติมได้ตาม Omise Docs
	}

	brandCode, ok := bankMap[req.BankName]
	if !ok {
		fmt.Println("❌ ไม่รองรับธนาคาร:", req.BankName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ชื่อธนาคารไม่ถูกต้องหรือไม่รองรับ"})
		return
	}

	// ตรวจว่าข้อมูลครบ
	if req.FirstName == "" || req.LastName == "" || req.BankAccountName == "" ||
		req.BankName == "" || req.BankAccountNumber == "" || req.CitizenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// ตรวจว่า "ชื่อ-นามสกุล" ตรงกับ "ชื่อบัญชี"
	fullName := req.FirstName + " " + req.LastName
	if fullName != req.BankAccountName {
		fmt.Println("❌ Missing field(s):", req)
		c.JSON(http.StatusBadRequest, gin.H{"error": "ชื่อบัญชีธนาคารไม่ตรงกับชื่อ-นามสกุลที่กรอก"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userCollection := db.OpenCollection("users")

	// ตรวจว่าบัตรประชาชนหรือเลขบัญชีซ้ำ
	filter := bson.M{
		"$or": []bson.M{
			{"seller_info.citizen_id": req.CitizenID},
			{"seller_info.bank_account_number": req.BankAccountNumber},
		},
	}
	count, err := userCollection.CountDocuments(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing seller data"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "เลขบัตรประชาชนหรือเลขบัญชีนี้เคยถูกใช้แล้ว"})
		return
	}

	// สร้าง Omise Recipient
	client, err := omise.NewClient(
		os.Getenv("OMISE_PUBLIC_KEY"),
		os.Getenv("OMISE_SECRET_KEY"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise client init failed"})
		return
	}

	recipient := &omise.Recipient{}
	err = client.Do(recipient, &operations.CreateRecipient{
		Name: req.BankAccountName,
		Type: "individual",
		BankAccount: &omise.BankAccountRequest{
			Brand:  brandCode,
			Number: req.BankAccountNumber,
			Name:   req.BankAccountName,
		},
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create recipient in Omise: " + err.Error()})
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
		IsVerified:        true,
		RecipientID:       recipient.ID,
	}

	// อัปเดต MongoDB
	update := bson.M{
		"$set": bson.M{
			"is_seller":   true,
			"seller_info": sellerInfo,
		},
	}
	result, err := userCollection.UpdateOne(ctx, bson.M{"_id": userObjID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Become seller success!",
		"recipient_id": recipient.ID,
	})
}
