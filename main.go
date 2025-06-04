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
	// โหลดค่า .env เฉพาะตอนพัฒนา local
	if _, err := os.Stat(".env"); err == nil {
		log.Println(" Loading .env for local development")
		_ = godotenv.Load()
	}

	// อ่าน path ไฟล์ credential จริงจาก env
	credentialPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialPath == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS env variable is missing")
	}

	// เรียก InitGCSFromFile แทน
	if err := gcs.InitGCSFromFile(credentialPath); err != nil {
		log.Fatalf("ไม่สามารถเชื่อมต่อ Google Cloud Storage: %v", err)
	}
	defer gcs.Close()

	// เริ่มเชื่อมต่อ MongoDB
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