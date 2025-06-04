package main

import (
	"arttoy-hub/controllers"
	"arttoy-hub/database"
	"arttoy-hub/gcs"
	"arttoy-hub/routes"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	// โหลดค่า Credential JSON จาก environment (Render จะใส่ให้ใน Settings)
	if _, err := os.Stat(".env"); err == nil {
		log.Println(" Loading .env for local development")
		_ = godotenv.Load()
	}
	credentialJson := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
	if credentialJson == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS_JSON is missing in environment")
	}

	// Init GCS โดยส่ง credential JSON เข้าไป
	if err := gcs. InitGCS(credentialJson); err != nil {
		log.Fatalf("ไม่สามารถเชื่อมต่อ Google Cloud Storage: %v", err)
	}
	defer gcs.Close()

	// เริ่ม MongoDB
	db.InitDB()
	defer db.DisconnectDB()

	controllers.InitMongo(db.Client)

	// ตั้งค่า router
	r := gin.Default()
	routes.SetupRoutes(r)

	log.Println("Starting server on :8080")
	c := cron.New()
	c.AddFunc("@every 1m", controllers.DeleteExpiredOrders)
	c.Start()

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
