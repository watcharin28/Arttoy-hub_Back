package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	// "github.com/omise/omise-go"
	// "github.com/omise/omise-go/operations"
)

func GetUserOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	orders, err := models.GetOrdersByUser(userObjID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}
	c.JSON(http.StatusOK, orders)
}
func GetSellerOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	sellerObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ordersCol := db.OpenCollection("orders")

	// หาคำสั่งซื้อที่มีสินค้าของ seller คนนั้น
	cursor, err := ordersCol.Find(ctx, bson.M{
		"items": bson.M{
			"$elemMatch": bson.M{
				"seller_id": sellerObjID,
			},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch seller orders"})
		return
	}
	defer cursor.Close(ctx)

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse seller orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func GetOrderByID(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := models.GetOrderByID(objID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}



type QRSourceResponse struct {
	ID            string `json:"id"`
	ScannableCode struct {
		Image struct {
			URI string `json:"uri"`
		} `json:"image"`
	} `json:"scannable_code"`
}

// สร้าง QR PromptPay Order
func CreatePromptPayCustomOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 🔒 ตรวจสอบว่ามีออเดอร์ที่ยังไม่จ่ายอยู่หรือไม่
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existing models.Order
	err = db.OpenCollection("orders").FindOne(ctx, bson.M{
		"user_id": userObjID,
		"status": bson.M{"$in": []string{"unpaid", "waiting_payment"}},
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "You already have an unpaid order",
			"order_id":  existing.ID.Hex(),
			"total":     existing.Total,
			"status":    existing.Status,
		})
		return
	}

	//  รับ product_id หลายตัว
	var input struct {
		Items []string `json:"items"` // ex: ["product_id1", "product_id2"]
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var orderItems []models.OrderItem
	var total float64

	for _, pid := range input.Items {
		productObjID, err := primitive.ObjectIDFromHex(pid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
			return
		}

		var product models.Product
		err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product)
		if err != nil || product.IsSold {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product not available or already sold"})
			return
		}

		orderItems = append(orderItems, models.OrderItem{
			ProductID: product.ID,
			Price:     product.Price,
		})
		total += product.Price
	}

	// 🧾 สร้าง QR กับ Omise
	payload := map[string]interface{}{
		"amount":   int(total * 100),
		"currency": "thb",
		"type":     "promptpay",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.omise.co/sources", bytes.NewBuffer(bodyBytes))
	req.SetBasicAuth(os.Getenv("OMISE_PUBLIC_KEY"), "")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create QR failed"})
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var qr QRSourceResponse
	json.Unmarshal(body, &qr)

	// 💾 สร้าง order ใหม่
	order := models.Order{
		UserID:    userObjID,
		Items:     orderItems,
		Total:     total,
		Status:    "waiting_payment",
		SourceID:  qr.ID,
		CreatedAt: time.Now(),
	}
	newOrder, err := models.CreateOrder(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create order failed"})
		return
	}

	// 📤 ส่ง QR กลับ
	qrImage := qr.ScannableCode.Image.URI
	if qrImage == "" {
		qrImage = "https://cdn.omise.co/scannable_code/test_qr.png"
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":  newOrder.ID.Hex(),
		"qr_image":  qrImage,
		"source_id": qr.ID,
	})
}


// ม็อคว่า “จ่ายแล้ว” (เฉพาะ test mode)
func MarkPromptPayOrderPaid(c *gin.Context) {
	orderID := c.Param("id")
	objID, _ := primitive.ObjectIDFromHex(orderID)
	userID := c.GetString("user_id")
	userObjID, _ := primitive.ObjectIDFromHex(userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order models.Order
	err := db.OpenCollection("orders").FindOne(ctx, bson.M{"_id": objID}).Decode(&order)
	if err != nil || order.UserID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}
	if order.Status != "waiting_payment" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order not in waiting_payment state"})
		return
	}

	http.NewRequest("POST", "https://api.omise.co/sources/"+order.SourceID+"/mark_as_paid", nil)
	req, _ := http.NewRequest("POST", "https://api.omise.co/sources/"+order.SourceID+"/mark_as_paid", nil)
	req.SetBasicAuth(os.Getenv("OMISE_SECRET_KEY"), "")
	http.DefaultClient.Do(req)

	db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{"status": "paid", "paid_at": time.Now()},
	})

	// ✅ อัปเดต is_sold ให้สินค้าใน order
	for _, item := range order.Items {
		db.OpenCollection("products").UpdateByID(ctx, item.ProductID, bson.M{
			"$set": bson.M{"is_sold": true},
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order marked as paid"})
}
