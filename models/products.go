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
    Category    string             `json:"category" bson:"category"`               // จากชื่อ
    Model       string             `json:"model" bson:"model"`
    Color       string             `json:"color" bson:"color"`
    Size        string             `json:"size" bson:"size"`
    ImageURLs   []string           `json:"product_image" bson:"product_image"`
    Rating      float64            `json:"rating" bson:"rating"`
    SellerID    primitive.ObjectID `json:"seller_id" bson:"seller_id"`
    IsSold      bool               `json:"is_sold" bson:"is_sold"`
    CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
}

// เพิ่มสินค้าใหม่
func AddProduct(product Product) (Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    product.ID = primitive.NewObjectID()
    product.CreatedAt = time.Now()

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

    // ✅ ดึงเฉพาะสินค้าที่ยังไม่ถูกขาย
    filter := bson.M{"is_sold": false}

    cursor, err := db.ProductCollection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    if err = cursor.All(ctx, &products); err != nil {
        return nil, err
    }
    return products, nil
}


// ดึงสินค้าโดย ID
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

// อัปเดตสินค้า
func UpdateProduct(id string, updatedProduct Product) (Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return Product{}, err
    }

    update := bson.M{
        "$set": bson.M{
            "name":         updatedProduct.Name,
            "description":  updatedProduct.Description,
            "price":        updatedProduct.Price,
            "category":     updatedProduct.Category,
            "model":        updatedProduct.Model,
            "color":        updatedProduct.Color,
            "size":         updatedProduct.Size,
            "product_image": updatedProduct.ImageURLs,
            "rating":       updatedProduct.Rating,
            "seller_id":    updatedProduct.SellerID,
            "is_sold":      updatedProduct.IsSold,
        },
    }

    _, err = db.ProductCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
    if err != nil {
        return Product{}, err
    }

    updatedProduct.ID = objID
    return updatedProduct, nil
}

// ลบสินค้า
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
