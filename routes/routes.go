package routes

import (
	// "net/http"
	 "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// CORS Middleware
	r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:5173"}, // ตั้งค่า origin ที่จะอนุญาต
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // method ที่อนุญาต
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"}, // headers ที่อนุญาต
        AllowCredentials: true, // อนุญาตให้ใช้ cookies และ credentials
    }))

	// รวม routes 
	SetupAuthRoutes(r)   
	SetupProductRoutes(r)
	SetupCartRoutes(r)
	SetupOrderRoutes(r)
	PaymentRoutes(r)
	CategoryRoutes(r)
	SetupReviewRoutes(r)
}