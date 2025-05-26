package models

import (
    "go.mongodb.org/mongo-driver/bson/primitive"
    "time"
)

type Review struct {
    ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    ProductID primitive.ObjectID `json:"product_id" bson:"product_id"`
    SellerID  primitive.ObjectID `json:"seller_id" bson:"seller_id"`
    UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
    Rating    int                `json:"rating" bson:"rating"`         // 1-5
    Comment   string             `json:"comment" bson:"comment"`
    CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}