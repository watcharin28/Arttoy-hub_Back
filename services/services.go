package services

import (
    "context"
    "time"
    "arttoy-hub/database"
    "arttoy-hub/models"

    "go.mongodb.org/mongo-driver/bson"
)

func SearchProductsService(keyword string) ([]models.Product, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    productCollection := db.OpenCollection("products")

    filter := bson.M{
        "$or": []bson.M{
            {"name": bson.M{"$regex": keyword, "$options": "i"}},
            {"description": bson.M{"$regex": keyword, "$options": "i"}},
        },
    }

    cursor, err := productCollection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var products []models.Product
    if err = cursor.All(ctx, &products); err != nil {
        return nil, err
    }

    return products, nil
}
