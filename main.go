package main

import (
	"arttoy-hub/controllers"
	"arttoy-hub/database"
	"arttoy-hub/routes"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// โหลดไฟล์ .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning Error loading .env file:", err)
	}

	// เริ่มต้นการเชื่อมต่อ MongoDB
	db.InitDB()
	defer db.DisconnectDB() // ยกเลิกการเชื่อมต่อเมื่อโปรแกรมจบ

	// ส่ง MongoDB client ไปยัง controllers
	controllers.InitMongo(db.Client)

	// ตั้งค่า Gin router
	r := gin.Default()

	// เรียก routes
	routes.SetupRoutes(r)
       // protected := r.Group("/protected")
    // protected.Use(controllers.AuthMiddleware())
    // protected.GET("/data", func(c *gin.Context) {
    //     c.JSON(http.StatusOK, gin.H{"message": "This is protected data"})
    // })
	// เริ่มเซิร์ฟเวอร์
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}