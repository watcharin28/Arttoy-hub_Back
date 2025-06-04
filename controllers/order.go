package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/omise/omise-go"
	"github.com/omise/omise-go/operations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ordersCol := db.OpenCollection("orders")
	productsCol := db.OpenCollection("products")

	cursor, err := ordersCol.Find(ctx, bson.M{
		"items": bson.M{
			"$elemMatch": bson.M{
				"seller_id": sellerObjID,
			},
		},
		"status": bson.M{
			"$in": []string{"pending", "shipping", "processing", "completed"}, // ✅ เฉพาะสถานะที่ต้องแสดง
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

	// เติมข้อมูลสินค้าให้แต่ละ order item
	for i := range orders {
		for j := range orders[i].Items {
			productID := orders[i].Items[j].ProductID
			var product models.Product
			err := productsCol.FindOne(ctx, bson.M{"_id": productID}).Decode(&product)
			if err == nil {
				orders[i].Items[j].Item = &product
			}
		}
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

var charge *omise.Charge

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ตรวจสอบออเดอร์ที่ยังไม่จ่าย
	var existing models.Order
	err = db.OpenCollection("orders").FindOne(ctx, bson.M{
		"user_id": userObjID,
		"status":  bson.M{"$in": []string{"unpaid", "waiting_payment"}},
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "You already have an unpaid order",
			"order_id": existing.ID.Hex(),
			"total":    existing.Total,
			"status":   existing.Status,
		})
		return
	}

	// รับข้อมูลจากผู้ใช้
	var input struct {
		Items []struct {
			ID       string `json:"id"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
		AddressID string `json:"address_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// ดึงที่อยู่
	var user models.User
	if err := db.OpenCollection("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var selectedAddr *models.Address
	if input.AddressID != "" {
		addrID, err := primitive.ObjectIDFromHex(input.AddressID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
			return
		}
		for _, addr := range user.Addresses {
			if addr.ID == addrID {
				selectedAddr = &addr
				break
			}
		}
		if selectedAddr == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Address not found"})
			return
		}
	} else {
		for _, addr := range user.Addresses {
			if addr.IsDefault {
				selectedAddr = &addr
				break
			}
		}
		if selectedAddr == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No default address set"})
			return
		}
	}

	// รวมรายการสินค้าและราคารวมสินค้า
	var orderItems []models.OrderItem
	var total float64

	for _, item := range input.Items {
		productObjID, err := primitive.ObjectIDFromHex(item.ID)
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

		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}

		orderItems = append(orderItems, models.OrderItem{
			ProductID: product.ID,
			SellerID:  product.SellerID,
			Price:     product.Price,
			Quantity:  qty,
		})

		total += product.Price * float64(qty)
	}

	// ✅ เพิ่มค่าส่งและยอดรวมทั้งหมด
	shippingFee := 40.0
	grandTotal := total + shippingFee

	// สร้าง QR กับ Omise
	payload := map[string]interface{}{
		"amount":   int(grandTotal * 100),
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

	client, err := omise.NewClient(os.Getenv("OMISE_PUBLIC_KEY"), os.Getenv("OMISE_SECRET_KEY"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise client init failed"})
		return
	}

	chargeOp := &operations.CreateCharge{
		Amount:   int64(grandTotal * 100),
		Currency: "thb",
		Source:   qr.ID,
	}

	var charge omise.Charge
	if err := client.Do(&charge, chargeOp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create charge failed: " + err.Error()})
		return
	}

	// ✅ สร้างคำสั่งซื้อ
	order := models.Order{
		UserID:      userObjID,
		Items:       orderItems,
		Total:       total,
		ShippingFee: shippingFee,
		GrandTotal:  grandTotal,
		Status:      "waiting_payment",
		SourceID:    qr.ID,
		ChargeID:    charge.ID,
		CreatedAt:   time.Now(),
		ExpiredAt:   time.Now().Add(1 * time.Minute),
	}

	newOrder, err := models.CreateOrder(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create order failed"})
		return
	}

	qrImage := qr.ScannableCode.Image.URI
	if qrImage == "" {
		qrImage = "https://cdn.omise.co/scannable_code/test_qr.png"
	}

	// ✅ ส่ง response กลับ
	c.JSON(http.StatusOK, gin.H{
		"order_id":     newOrder.ID.Hex(),
		"qr_image":     qrImage,
		"source_id":    qr.ID,
		"charge_id":    charge.ID,
		"total":        total,
		"shipping_fee": shippingFee,
		"grand_total":  grandTotal,
		"address_used": selectedAddr,
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

	// ✅ ตรวจสอบว่า order หมดอายุหรือยัง
	if time.Now().After(order.ExpiredAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This order has expired"})
		return
	}

	// ✅ แจ้ง Omise (mocked, ไม่มี check response)
	req, _ := http.NewRequest("POST", "https://api.omise.co/sources/"+order.SourceID+"/mark_as_paid", nil)
	req.SetBasicAuth(os.Getenv("OMISE_SECRET_KEY"), "")
	http.DefaultClient.Do(req)

	// ✅ อัปเดต order เป็น paid/pending
	db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"status":  "pending",
			"paid_at": time.Now(),
		},
	})

	// ✅ ตั้ง is_sold ให้สินค้าใน order
	for _, item := range order.Items {
		db.OpenCollection("products").UpdateByID(ctx, item.ProductID, bson.M{
			"$set": bson.M{"is_sold": true},
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order marked as paid"})
}
func DeleteExpiredOrders() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"status":     "waiting_payment",
		"expired_at": bson.M{"$lt": time.Now()},
	}

	result, err := db.OpenCollection("orders").DeleteMany(ctx, filter)
	if err != nil {
		log.Printf("❌ Failed to delete expired orders: %v", err)
		return
	}

	log.Printf("✅ Deleted %v expired orders", result.DeletedCount)
}
