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
    // 1) ตรวจสอบว่าเข้ามาที่ Handler จริงหรือไม่
    log.Println("⚡️ Enter CreatePromptPayCustomOrder")

    // 2) ดึง user_id จาก context (JWT middleware ต้อง set ให้ถูกจังหวะ)
    userID := c.GetString("user_id")
    log.Printf("🔑 user_id from context: '%s'\n", userID)
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged in"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        log.Printf("❌ Invalid user ID (cannot convert to ObjectID): %v\n", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 3) เช็กว่า user ยังไม่มีออเดอร์ค้างในสถานะ unpaid|waiting_payment
    var existing models.Order
    err = db.OpenCollection("orders").FindOne(ctx, bson.M{
        "user_id": userObjID,
        "status":  bson.M{"$in": []string{"unpaid", "waiting_payment"}},
    }).Decode(&existing)

    if err == nil {
        log.Printf("⚠️ Found existing unpaid order: ID=%s, Total=%.2f, Status=%s\n",
            existing.ID.Hex(), existing.Total, existing.Status)
        c.JSON(http.StatusBadRequest, gin.H{
            "error":    "You already have an unpaid order",
            "order_id": existing.ID.Hex(),
            "total":    existing.Total,
            "status":   existing.Status,
        })
        return
    }

    // 4) รับ JSON จากผู้ใช้ (Bind เพื่อ map เข้า struct)
    var input struct {
        Items []struct {
            ID       string `json:"id"`       // ต้องตรงกับชื่อ key ในฝั่ง React
            Quantity int    `json:"quantity"` // ต้องตรงกับชื่อ key ในฝั่ง React
        } `json:"items"`
        AddressID string `json:"address_id"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        // ถ้า BindJSON ผิดพลาด ให้พิมพ์ raw body ว่าไปถึง backend จริง ๆ ว่ามีอะไรมา
        raw, _ := ioutil.ReadAll(c.Request.Body)
        log.Printf("❌ BindJSON Error: %v\n", err)
        log.Printf("👉 Raw request body: %s\n", string(raw))
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format", "detail": err.Error()})
        return
    }

    // ถ้า BindJSON ผ่าน แต่ไม่มี items เลย => error
    if len(input.Items) == 0 {
        log.Println("❌ No items provided in input") 
        c.JSON(http.StatusBadRequest, gin.H{"error": "No items provided"})
        return
    }

    // 5) พิมพ์ข้อมูลที่ Bind มาได้ ว่ามี items ไหนบ้าง และ address_id อะไร
    log.Printf("✅ Bound input.AddressID = %s\n", input.AddressID)
    for idx, it := range input.Items {
        log.Printf("    item[%d] => ID: %s, Quantity: %d\n", idx, it.ID, it.Quantity)
    }

    // 6) ดึงข้อมูล user จากฐานข้อมูล เพื่อไปหา address ต่อ
    var user models.User
    if err := db.OpenCollection("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err != nil {
        log.Printf("❌ User not found in DB: %v\n", err)
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }

    // 7) หา selectedAddr จาก input.AddressID หรือ default
    var selectedAddr *models.Address
    if input.AddressID != "" {
        addrID, err := primitive.ObjectIDFromHex(input.AddressID)
        if err != nil {
            log.Printf("❌ Invalid address ID: %v\n", err)
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
            log.Println("❌ Address not found in user's address list")
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
            log.Println("❌ No default address set for user")
            c.JSON(http.StatusBadRequest, gin.H{"error": "No default address set"})
            return
        }
    }

    // 8) คำนวณรวมราคาสินค้า และตรวจเช็กว่าแต่ละ product ยังไม่ถูกขาย
    var orderItems []models.OrderItem
    var total float64
    for _, item := range input.Items {
        productObjID, err := primitive.ObjectIDFromHex(item.ID)
        if err != nil {
            log.Printf("❌ Invalid product ID: %s, error: %v\n", item.ID, err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID: " + item.ID})
            return
        }

        var product models.Product
        err = db.OpenCollection("products").FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product)
        if err != nil {
            log.Printf("❌ Product not found: %s, error: %v\n", item.ID, err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Product not found: " + item.ID})
            return
        }
        if product.IsSold {
            log.Printf("❌ Product already sold: %s\n", item.ID)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Product already sold: " + item.ID})
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

    // 9) กำหนดค่าส่งและคำนวณ GrandTotal
    shippingFee := 40.0
    grandTotal := total + shippingFee

    // 10) สร้าง QR PromptPay กับ Omise (Create Source)
    payload := map[string]interface{}{
        "amount":   int(grandTotal * 100),
        "currency": "thb",
        "type":     "promptpay",
    }
    bodyBytes, _ := json.Marshal(payload)
    reqOmise, _ := http.NewRequest("POST", "https://api.omise.co/sources", bytes.NewBuffer(bodyBytes))
    reqOmise.SetBasicAuth(os.Getenv("OMISE_PUBLIC_KEY"), "")
    reqOmise.Header.Set("Content-Type", "application/json")

    respOmise, err := http.DefaultClient.Do(reqOmise)
    if err != nil {
        log.Printf("❌ Omise Create Source request failed: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QR source"})
        return
    }
    defer respOmise.Body.Close()
    if respOmise.StatusCode != http.StatusOK {
        rawBody, _ := ioutil.ReadAll(respOmise.Body)
        log.Printf("❌ Omise Create Source returned status %d: %s\n", respOmise.StatusCode, string(rawBody))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise Create Source failed"})
        return
    }

    var qr QRSourceResponse
    respBytes, _ := ioutil.ReadAll(respOmise.Body)
    if err := json.Unmarshal(respBytes, &qr); err != nil {
        log.Printf("❌ Failed to unmarshal QR response: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse QR response"})
        return
    }
    log.Printf("✅ QR Source ID = %s\n", qr.ID)

    // 11) สร้าง Charge กับ Omise (Create Charge)
    client, err := omise.NewClient(os.Getenv("OMISE_PUBLIC_KEY"), os.Getenv("OMISE_SECRET_KEY"))
    if err != nil {
        log.Printf("❌ Omise client init failed: %v\n", err)
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
        log.Printf("❌ Create Charge failed: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Create charge failed: " + err.Error()})
        return
    }
    log.Printf("✅ Created Omise charge ID = %s\n", charge.ID)

    // 12) สร้าง Order ใน MongoDB
    order := models.Order{
        UserID:          userObjID,
        Items:           orderItems,
        Total:           total,
        ShippingFee:     shippingFee,
        GrandTotal:      grandTotal,
        Status:          "waiting_payment",
        SourceID:        qr.ID,
        ChargeID:        charge.ID,
        ShippingAddress: *selectedAddr,
        CreatedAt:       time.Now(),
        ExpiredAt:       time.Now().Add(1 * time.Minute),
    }

    newOrder, err := models.CreateOrder(order)
    if err != nil {
        log.Printf("❌ CreateOrder DB failed: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Create order failed"})
        return
    }
    log.Printf("✅ New order created: ID=%s\n", newOrder.ID.Hex())

    // 13) เตรียมค่า qrImage กลับไปให้ frontend
    qrImage := qr.ScannableCode.Image.URI
    if qrImage == "" {
        qrImage = "https://cdn.omise.co/scannable_code/test_qr.png"
    }

    // 14) ส่ง response กลับ
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