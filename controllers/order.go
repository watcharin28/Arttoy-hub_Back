package controllers

import (
	"arttoy-hub/models"
	"context"
	"net/http"
	"time"
	"os"
	"arttoy-hub/database"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	
    "github.com/omise/omise-go"
    "github.com/omise/omise-go/operations"
)

// สร้างคำสั่งซื้อจากตะกร้า
func CreateOrder(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cartCursor, err := db.OpenCollection("carts").Find(ctx, bson.M{"user_id": userObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items"})
		return
	}
	defer cartCursor.Close(ctx)

	var cartItems []models.CartItem
	if err := cartCursor.All(ctx, &cartItems); err != nil || len(cartItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	var orderItems []models.OrderItem
	var total float64

	for _, item := range cartItems {
		var product models.Product
		err := db.OpenCollection("products").FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
		if err != nil || product.IsSold {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product not available or already sold"})
			return
		}

		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Price:     product.Price,
		})
		total += product.Price

		// อัปเดตสถานะ is_sold
		_, err = db.OpenCollection("products").UpdateByID(ctx, product.ID, bson.M{"$set": bson.M{"is_sold": true}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product status"})
			return
		}
	}

	// ลบตะกร้าของผู้ใช้
	_, _ = db.OpenCollection("carts").DeleteMany(ctx, bson.M{"user_id": userObjID})

	order := models.Order{
		UserID: userObjID,
		Items:  orderItems,
		Total:  total,
		Status: "unpaid",
		CreatedAt: time.Now(),
	}
	newOrder, err := models.CreateOrder(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, newOrder)
}

func CreateCustomOrder(c *gin.Context) {
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

	var input struct {
		Items []struct {
			ProductID string `json:"product_id"`
		} `json:"items"`
	}

	if err := c.ShouldBindJSON(&input); err != nil || len(input.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or empty items"})
		return
	}

	var orderItems []models.OrderItem
	var total float64
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, i := range input.Items {
		productObjID, err := primitive.ObjectIDFromHex(i.ProductID)
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

		_, err = db.OpenCollection("products").UpdateByID(ctx, product.ID, bson.M{"$set": bson.M{"is_sold": true}})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
			return
		}

		_, _ = db.OpenCollection("carts").DeleteOne(ctx, bson.M{
			"user_id":    userObjID,
			"product_id": product.ID,
		})

		orderItems = append(orderItems, models.OrderItem{
			ProductID: product.ID,
			Price:     product.Price,
		})
		total += product.Price
	}

	order := models.Order{
		UserID: userObjID,
		Items:  orderItems,
		Total:  total,
		Status: "unpaid",
		CreatedAt: time.Now(),
	}
	newOrder, err := models.CreateOrder(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, newOrder)
}

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

func PayOrder(c *gin.Context) {
	orderID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// 🔐 ดึง user จาก JWT
	userID := c.GetString("user_id")
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Token string `json:"token"` // รับ token จาก frontend/Postman
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
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

	// 🔒 ตรวจว่าเป็นเจ้าของ order หรือไม่
	if order.UserID != userObjID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to pay for this order"})
		return
	}

	if order.Status == "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already paid"})
		return
	}

	// ✅ เรียก Omise เพื่อสร้าง charge
	client, err := omise.NewClient(
		os.Getenv("OMISE_PUBLIC_KEY"),
		os.Getenv("OMISE_SECRET_KEY"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise init failed"})
		return
	}

	charge := &omise.Charge{}
	err = client.Do(charge, &operations.CreateCharge{
		Amount:   int64(order.Total * 100), // บาท → สตางค์
		Currency: "thb",
		Card:     req.Token,
	})
	if err != nil || !charge.Paid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Charge failed", "details": err.Error()})
		return
	}

	_, err = db.OpenCollection("orders").UpdateByID(ctx, objID, bson.M{
		"$set": bson.M{
			"status":    "paid",
			"charge_id": charge.ID,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Payment successful",
		"charge_id": charge.ID,
	})
}



