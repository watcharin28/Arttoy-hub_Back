package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username     string             `json:"username" bson:"username"`
	Password     string             `json:"password" bson:"password"`
	Phonenumber  string             `json:"phonenumber" bson:"phonenumber"`
	Gmail        string             `json:"gmail" bson:"gmail"`
	ProfileImage string             `json:"profile_image,omitempty" bson:"profile_image,omitempty"`
	LikedItems   []string           `json:"likedItems" bson:"likedItems"`
	Addresses    []Address          `json:"addresses" bson:"addresses"`
	SellerInfo   *SellerInfo        `json:"seller_info,omitempty" bson:"seller_info,omitempty"`
	IsSeller     bool               `json:"is_seller" bson:"is_seller"`
}
type SellerInfo struct {
	FirstName         string `json:"first_name" bson:"first_name"`
	LastName          string `json:"last_name" bson:"last_name"`
	BankAccountName   string `json:"bank_account_name" bson:"bank_account_name"`
	BankName          string `json:"bank_name" bson:"bank_name"`
	BankAccountNumber string `json:"bank_account_number" bson:"bank_account_number"`
	CitizenID         string `json:"citizen_id" bson:"citizen_id"`
	IDCardImageURL    string `json:"id_card_image_url" bson:"id_card_image_url"`
	IsVerified        bool   `json:"is_verified" bson:"is_verified"`
}
