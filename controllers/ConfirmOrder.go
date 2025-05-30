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

	//  ตรวจว่าเป็น order ของผู้ซื้อที่ login อยู่
	if order.UserID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to confirm this order"})
		return
	}

	//  ตรวจว่าคำสั่งซื้อต้องอยู่ในสถานะ "paid"
	if order.Status != "processing" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order not in paid state"})
		return
	}

	//  ตรวจว่าผู้ขายได้กรอกเลขพัสดุหรือยัง
	if order.TrackingNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ยังไม่สามารถยืนยันได้ เพราะผู้ขายยังไม่ได้กรอกเลขพัสดุ"})
		return
	}

	//  ดึงสินค้าแรก (สินค้ามือสองมีชิ้นเดียว)
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

	// ดึงข้อมูลผู้ขายจาก product
	var seller models.User
	err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": product.SellerID}).Decode(&seller)
	if err != nil || seller.SellerInfo == nil || seller.SellerInfo.RecipientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Seller or recipient not found"})
		return
	}

	// สร้าง Omise client
	client, err := omise.NewClient(
		os.Getenv("OMISE_PUBLIC_KEY"),
		os.Getenv("OMISE_SECRET_KEY"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise client error"})
		return
	}

	// สร้าง transfer ไปยัง recipient ของผู้ขาย
	transfer := &omise.Transfer{}
	err = client.Do(transfer, &operations.CreateTransfer{
		Amount:    int64(order.Total * 100), // แปลงเป็นสตางค์
		Recipient: seller.SellerInfo.RecipientID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transfer failed", "details": err.Error()})
		return
	}

	// อัปเดตสถานะ order เป็น completed และเก็บ transfer_id
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

	// ✅ รับทั้ง tracking_number และ sender_name
	var input struct {
		TrackingNumber string `json:"tracking_number"`
		SenderName     string `json:"sender_name"` // เพิ่มตรงนี้
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.TrackingNumber == "" || input.SenderName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tracking number and sender name are required"})
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

	// ตรวจสิทธิ์ผู้ขาย
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

	// ✅ อัปเดต tracking + sender + status
	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"tracking_number": input.TrackingNumber,
			"sender_name":     input.SenderName,
			"status":          "processing",
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tracking number"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tracking number and sender name updated"})
}

func AcceptOrderBySeller(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	sellerID := c.GetString("user_id")
	sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
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

	if len(order.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order has no items"})
		return
	}

	// ตรวจว่าเป็นสินค้าของผู้ขายคนนี้
	var product models.Product
	err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": order.Items[0].ProductID}).Decode(&product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Product not found"})
		return
	}

	if product.SellerID != sellerObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to accept this order"})
		return
	}

	if order.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only 'pending' orders can be accepted"})
		return
	}

	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"status": "shipping",
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order accepted successfully"})
}
func RejectOrderBySeller(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	sellerID := c.GetString("user_id")
	sellerObjID, err := primitive.ObjectIDFromHex(sellerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
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

	if product.SellerID != sellerObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to reject this order"})
		return
	}

	if order.Status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only 'paid' orders can be rejected"})
		return
	}

	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"status": "rejected",
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order rejected successfully"})
}
