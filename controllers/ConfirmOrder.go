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
	"regexp"
	"strings"
	"time"
)

func ConfirmOrderDelivery(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	userID := c.GetString("user_id")
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order models.Order
	err = db.OpenCollection("orders").FindOne(ctx, bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// ตรวจว่า order เป็นของผู้ซื้อที่ login อยู่
	if order.UserID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to confirm this order"})
		return
	}

	if order.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order not in paid state"})
		return
	}

	if len(order.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order has no items"})
		return
	}

	var product models.Product
	err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": order.Items[0].ProductID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Product not found"})
		return
	}

	var seller models.User
	err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": product.SellerID}).Decode(&seller)
	if err != nil || seller.SellerInfo == nil || seller.SellerInfo.RecipientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Seller or recipient not found"})
		return
	}

	client, err := omise.NewClient(
		os.Getenv("OMISE_PUBLIC_KEY"),
		os.Getenv("OMISE_SECRET_KEY"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise client error"})
		return
	}

	transfer := &omise.Transfer{}
	err = client.Do(transfer, &operations.CreateTransfer{
		Amount:    int64(order.Total * 100),
		Recipient: seller.SellerInfo.RecipientID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transfer failed", "details": err.Error()})
		return
	}

	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"status":      "completed",
			"transfer_id": transfer.ID,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transfer completed successfully",
		"transfer_id": transfer.ID,
	})
}

func UpdateTrackingNumber(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// ดึง userID จาก JWT context
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// รับ tracking_number จาก body
	var input struct {
		TrackingNumber string `json:"tracking_number"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.TrackingNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tracking number is required"})
		return
	}

	// ตรวจรูปแบบเบื้องต้น เช่น TH1234567890, KERRY999999
	validFormat := regexp.MustCompile(`^(TH|KERRY|FLASH)[0-9A-Z]{8,}$`)
	if !validFormat.MatchString(strings.ToUpper(input.TrackingNumber)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tracking number format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ดึง order
	var order models.Order
	err = db.OpenCollection("orders").FindOne(ctx, bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// ตรวจสิทธิ์: ผู้ขายต้องเป็นเจ้าของสินค้า
	if len(order.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order has no items"})
		return
	}
	var product models.Product
	err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": order.Items[0].ProductID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Product not found"})
		return
	}

	if product.SellerID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to update this order"})
		return
	}

	// อัปเดต tracking number และสถานะ
	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"tracking_number": input.TrackingNumber,
			"status":          "shipped",
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tracking number"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tracking number updated"})
}
