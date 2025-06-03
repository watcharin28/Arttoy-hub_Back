package routes

import (
	// "net/http"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// CORS Middleware
	r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"https://incandescent-pastelito-cadd99.netlify.app"}, // ตั้งค่า origin ที่จะอนุญาต
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // method ที่อนุญาต
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"}, // headers ที่อนุญาต
        AllowCredentials: true, // อนุญาตให้ใช้ cookies และ credentials
    }))
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.AbortWithStatus(204)
	})

	// รวม routes 
	SetupAuthRoutes(r)   
	SetupProductRoutes(r)
	SetupCartRoutes(r)
	SetupOrderRoutes(r)
	PaymentRoutes(r)
	CategoryRoutes(r)
	SetupReviewRoutes(r)
	SetupSellerRoutes(r)
	
}