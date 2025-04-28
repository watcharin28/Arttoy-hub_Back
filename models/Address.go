package models
import "go.mongodb.org/mongo-driver/bson/primitive"

type Address struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Address     string             `json:"address" bson:"address"`
    Province    string             `json:"province" bson:"province"`
    PostalCode  string             `json:"postalCode" bson:"postalCode"`
    PhoneNumber string             `json:"phoneNumber" bson:"phoneNumber"`
    IsDefault   bool               `json:"isDefault" bson:"isDefault"` // optional แต่แนะนำมี
}
