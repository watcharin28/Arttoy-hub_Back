package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Report struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	Issue     string             `json:"issue" bson:"issue"`
	Status    string             `json:"status" bson:"status"` // เช่น: "รอตรวจสอบ", "แก้ไขแล้ว"
	CreatedAt primitive.DateTime `json:"created_at" bson:"created_at"`
}
