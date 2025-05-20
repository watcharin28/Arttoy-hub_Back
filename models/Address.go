package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Address struct ที่รองรับข้อมูลใหม่
type Address struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Name        string             `json:"name" bson:"name"`
    Phone       string             `json:"phone" bson:"phone"`
    Address     string             `json:"address" bson:"address"`
    Subdistrict string             `json:"subdistrict" bson:"subdistrict"`
    District    string             `json:"district" bson:"district"`
    Province    string             `json:"province" bson:"province"`
    Zipcode     string             `json:"zipcode" bson:"zipcode"`
    IsDefault   bool               `json:"isDefault" bson:"isDefault"` // optional แต่แนะนำมี
}
