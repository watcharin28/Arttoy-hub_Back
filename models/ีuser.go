package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
    ID       primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Username string             `json:"username" bson:"username"`
    Password string             `json:"password" bson:"password"`
    Phonenumber string          `json:"phonenumber" bson:"phonenumber"`
    Gmail string                `json:"gmail" bson:"gmail"`
}