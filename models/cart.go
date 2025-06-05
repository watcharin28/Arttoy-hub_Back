// 📁 models/cart.go
package models

import (
	"arttoy-hub/database"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
	"fmt"
	"log"
)

type CartItemWithProduct struct {
	ID         primitive.ObjectID `json:"id"`
	ProductID  primitive.ObjectID `json:"product_id"`
	Name       string             `json:"name"`
	ImageURL   string             `json:"product_image"`
	Price      float64            `json:"price"`
	Quantity   int                `json:"quantity"`
	AddedAt    time.Time          `json:"added_at"`
	SellerName string             `json:"seller_name"`
}
type CartItem struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	ProductID primitive.ObjectID `json:"product_id" bson:"product_id"`
	Quantity  int                `json:"quantity" bson:"quantity"` //จำนวน
	AddedAt   time.Time          `json:"added_at" bson:"added_at"` //เวลาที่เพิ่ม
}

func GetCartDetailsForUser(userID primitive.ObjectID) ([]CartItemWithProduct, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. ดึง cart จาก user_id
	var cartItems []CartItem
	cursor, err := db.OpenCollection("carts").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &cartItems); err != nil {
		return nil, err
	}

	// 2. รวบรวม product_id ทั้งหมด
	var productIDs []primitive.ObjectID
	for _, item := range cartItems {
		productIDs = append(productIDs, item.ProductID)
	}

	// 3. ดึงรายละเอียดสินค้าทั้งหมด
	var products []Product
	if len(productIDs) == 0 {
	fmt.Println("⚠️ ไม่มี productIDs ใน cart นี้เลย")
	return []CartItemWithProduct{}, nil // คืน array ว่างแทน
}
	cursor2, err := db.OpenCollection("products").Find(ctx, bson.M{
		"_id":     bson.M{"$in": productIDs},
		"is_sold": false,
	})
	if err != nil {
		return nil, err
	}
	if err := cursor2.All(ctx, &products); err != nil {
		 log.Printf("❌ decode products error: %v\n", err)
		return nil, err
	}

	// 4. สร้าง map[productID] = Product
	productMap := make(map[primitive.ObjectID]Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	// 5. รวมข้อมูลกลับคืน
	var result []CartItemWithProduct
	for _, item := range cartItems {
		if p, ok := productMap[item.ProductID]; ok {
			imageURL := ""
			if len(p.ImageURLs) > 0 {
				imageURL = p.ImageURLs[0]
			}

			// ดึงชื่อผู้ขาย
			var seller User
			sellerName := "Unknown" // Default fallback
			err := db.OpenCollection("users").FindOne(ctx, bson.M{"_id": p.SellerID}).Decode(&seller)
			if err == nil {
				sellerName = seller.Username
			} else {
				fmt.Println("❌ Error finding seller:", err) // Log ช่วย debug
			}

			result = append(result, CartItemWithProduct{
				ID:         item.ID,
				ProductID:  item.ProductID,
				Name:       p.Name,
				ImageURL:   imageURL,
				Price:      p.Price,
				Quantity:   item.Quantity,
				AddedAt:    item.AddedAt,
				SellerName: sellerName,
			})
		}
	}

	return result, nil
}

// เพิ่มสินค้าลงตะกร้า
func AddToCart(item CartItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	item.ID = primitive.NewObjectID()
	item.AddedAt = time.Now()

	_, err := db.OpenCollection("carts").InsertOne(ctx, item)
	return err
}

// ดึงสินค้าทั้งหมดในตะกร้าของผู้ใช้
func GetCartItemsByUser(userID primitive.ObjectID) ([]CartItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var items []CartItem
	cursor, err := db.OpenCollection("carts").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// ลบสินค้าออกจากตะกร้า
func RemoveFromCart(userID, productID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.OpenCollection("carts").DeleteOne(ctx, bson.M{
		"user_id":    userID,
		"product_id": productID,
	})
	return err
}
