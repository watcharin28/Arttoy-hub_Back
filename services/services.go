package services

import (
    "context"
    "time"
    "arttoy-hub/database"
    "arttoy-hub/models"

    "go.mongodb.org/mongo-driver/bson"
)

func SearchProductsService(keyword string, categoryList []string) ([]models.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"is_sold": false,}

	if keyword != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": keyword, "$options": "i"}},
			{"description": bson.M{"$regex": keyword, "$options": "i"}},
			{"category": bson.M{"$regex": keyword, "$options": "i"}},
			{"model": bson.M{"$regex": keyword, "$options": "i"}},
		}
	}

	if len(categoryList) > 0 {
		filter["category"] = bson.M{"$in": categoryList}
	}

	cursor, err := db.ProductCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}


