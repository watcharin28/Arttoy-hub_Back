package routes

import (
    "arttoy-hub/controllers"
    "github.com/gin-gonic/gin"
)

func SetupAuthRoutes(r *gin.Engine) {
    // Public auth routes
    r.POST("/register", controllers.Register) 
    r.POST("/login", controllers.Login)       
}

func SetupProductRoutes(r *gin.Engine) {
    // Product routes
    r.POST("/products", controllers.AddProduct)    // เพิ่มสินค้า
    r.GET("/products", controllers.GetAllProducts) // แสดงสินค้าทั้งหมด
	r.GET("/products/:id", controllers.GetProductByID)
	r.PUT("/products/:id", controllers.UpdateProduct)
	r.DELETE("/products/:id", controllers.DeleteProduct)
}

