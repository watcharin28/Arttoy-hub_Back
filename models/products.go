package models

import (
    "context"
    "time"
    "arttoy-hub/database"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

type Product struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Name        string             `json:"name" bson:"name"`
    Description string             `json:"description" bson:"description"`
    Price       float64            `json:"price" bson:"price"`
    Stock       int                `json:"stock" bson:"stock"`
    CategoryID  primitive.ObjectID `json:"category_id" bson:"category_id"`
    ImageURL    string             `json:"product_image" bson:"product_image"`
    Rating      float64            `json:"rating" bson:"rating"`
}

// เพิ่มสินค้า
func AddProduct(product Product) (Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    product.ID = primitive.NewObjectID()
    _, err := db.ProductCollection.InsertOne(ctx, product)
    if err != nil {
        return Product{}, err
    }
    return product, nil
}

// ดึงสินค้าทั้งหมด
func GetAllProducts() ([]Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var products []Product
    cursor, err :=db.ProductCollection.Find(ctx, bson.M{})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &products); err != nil {
        return nil, err
    }
    return products, nil
}
func GetProductByID(id string) (Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return Product{}, err
    }

    var product Product
    err = db.ProductCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&product)
    if err != nil {
        return Product{}, err
    }
    return product, nil
}
func UpdateProduct(id string, updatedProduct Product) (Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return Product{}, err
    }

    update := bson.M{
        "$set": bson.M{
            "name":        updatedProduct.Name,
            "description": updatedProduct.Description,
            "price":       updatedProduct.Price,
            "stock":       updatedProduct.Stock,
            "category_id": updatedProduct.CategoryID,
            "image_url":   updatedProduct.ImageURL,
            "rating":      updatedProduct.Rating,
        },
    }

    _, err = db.ProductCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
    if err != nil {
        return Product{}, err
    }

    updatedProduct.ID = objID
    return updatedProduct, nil
}
func DeleteProduct(id string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    result, err := db.ProductCollection.DeleteOne(ctx, bson.M{"_id": objID})
    if err != nil {
        return err
    }
    if result.DeletedCount == 0 {
        return mongo.ErrNoDocuments
    }
    return nil
}
