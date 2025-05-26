package models

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Review struct {
    ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    ProductID primitive.ObjectID `json:"product_id" bson:"product_id"` // อ้างถึงสินค้าที่รีวิว
    UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`       // อ้างถึงผู้ใช้ที่เขียนรีวิว
    Rating    int                `json:"rating" bson:"rating"`         // ระดับคะแนน 1-5 ดาว
    Comment   string             `json:"comment" bson:"comment"`       // ข้อความรีวิว
    CreatedAt time.Time          `json:"created_at" bson:"created_at"` // เวลาเขียนรีวิว
}
