// 📁 models/order.go
package models

import (
	"context"
	"time"
	"arttoy-hub/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderItem struct {
	ProductID primitive.ObjectID `json:"product_id" bson:"product_id"`
	SellerID  primitive.ObjectID `json:"seller_id" bson:"seller_id"`
	Price     float64            `json:"price" bson:"price"`
	Quantity  int                `bson:"quantity" json:"quantity"`
	Item      *Product           `json:"item,omitempty" bson:"-"` 
}

type Order struct {
	ID              primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id" bson:"user_id"`
	Items           []OrderItem        `json:"items" bson:"items"`
	Total           float64            `json:"total" bson:"total"`
	ShippingFee     float64            `json:"shipping_fee" bson:"shipping_fee"`       
	GrandTotal      float64            `json:"grand_total" bson:"grand_total"`
	ChargeID        string             `json:"charge_id,omitempty" bson:"charge_id,omitempty"`
	TransferID      string             `json:"transfer_id,omitempty" bson:"transfer_id,omitempty"`
	Status          string             `json:"status" bson:"status"`
	TrackingNumber  string             `json:"tracking_number,omitempty" bson:"tracking_number,omitempty"`
	SenderName      string             `json:"sender_name,omitempty" bson:"sender_name,omitempty"`
	ShippingAddress Address `bson:"shipping_address" json:"shippingAddress"`
	CreatedAt       time.Time          `json:"created_at" bson:"created_at"`
	SourceID        string             `json:"source_id,omitempty" bson:"source_id,omitempty"`
	PaidAt          time.Time          `json:"paid_at,omitempty" bson:"paid_at,omitempty"`
	ExpiredAt   time.Time          `bson:"expired_at" json:"expired_at"`
	
}

func CreateOrder(order Order) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	order.ID = primitive.NewObjectID()
	order.CreatedAt = time.Now()

	_, err := db.OpenCollection("orders").InsertOne(ctx, order)
	return order, err
}

func GetOrdersByUser(userID primitive.ObjectID) ([]Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var orders []Order
	cursor, err := db.OpenCollection("orders").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func GetOrderByID(orderID primitive.ObjectID) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order Order
	err := db.OpenCollection("orders").FindOne(ctx, bson.M{"_id": orderID}).Decode(&order)
	return order, err
}
