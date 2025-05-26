package models

import (
    "context"
    "time"

    "arttoy-hub/database"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Category struct {
    ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Name string             `json:"name" bson:"name"` // เช่น "Dimoo"
}

// เพิ่ม Category
func AddCategory(name string) (Category, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    category := Category{
        ID:   primitive.NewObjectID(),
        Name: name,
    }

    _, err := db.CategoryCollection.InsertOne(ctx, category)
    if err != nil {
        return Category{}, err
    }

    return category, nil
}

// ดึงทั้งหมด
func GetAllCategories() ([]Category, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var categories []Category
    cursor, err := db.CategoryCollection.Find(ctx, bson.M{})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &categories); err != nil {
        return nil, err
    }

    return categories, nil
}
