package db

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

// Client ตัวแปร global สำหรับการเชื่อมต่อ MongoDB
var Client *mongo.Client

// ProductCollection ตัวแปรสำหรับ collection "products"
var ProductCollection *mongo.Collection
var CategoryCollection *mongo.Collection
var UserCollection *mongo.Collection
var ReviewCollection *mongo.Collection

// InitDB เริ่มต้นการเชื่อมต่อ MongoDB
func InitDB() {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI not set in .env")
	}

	// ตั้งค่า client options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// สร้าง context พร้อม timeout เพื่อเชื่อมต่อ
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// เชื่อมต่อ MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// ทดสอบการเชื่อมต่อด้วย ping
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	Client = client
	// กำหนด ProductCollection (ปรับชื่อ database ตามที่คุณใช้ใน MongoDB Atlas)
	ProductCollection = client.Database("arttoyhub_db").Collection("products")
	CategoryCollection = client.Database("arttoyhub_db").Collection("categories")
	UserCollection = client.Database("arttoyhub_db").Collection("users")
	ReviewCollection = client.Database("arttoyhub_db").Collection("reviews")

	log.Println("Connected to MongoDB Atlas!")
}

// DisconnectDB ยกเลิกการเชื่อมต่อ MongoDB
func DisconnectDB() {
	if Client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Client.Disconnect(ctx)
	if err != nil {
		log.Println("Failed to disconnect MongoDB:", err)
		return
	}
	log.Println("Disconnected from MongoDB")
}

// OpenCollection คืนค่าคอลเลกชันจากชื่อที่กำหนด
func OpenCollection(collectionName string) *mongo.Collection {
	return Client.Database("arttoyhub_db").Collection(collectionName)
}
