package routes

import (
	"arttoy-hub/controllers"
	"arttoy-hub/middleware"
	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(r *gin.Engine) {
	// Public user routes
	r.POST("/Login", controllers.Login)       // Login route
	r.POST("/Register", controllers.Register) // Register route
	r.GET("/search", controllers.SearchProducts)
	// ใช้ AuthMiddleware กับ routes ที่ต้องการให้ login ก่อน
	userRoutes := r.Group("/api/user")
	userRoutes.Use(middlewares.AuthMiddleware()) // ต้อง login ก่อน
	{
		userRoutes.POST("/logout", controllers.Logout)
		userRoutes.PUT("/Profile", controllers.UpdateProfile)
		userRoutes.GET("/Profile", controllers.GetProfile)
		userRoutes.GET("/favorites", controllers.GetUserFavorites)
		userRoutes.DELETE("/favorites/:product_id", controllers.DeleteUserFavorite)
		userRoutes.POST("/favorites/:product_id", controllers.LikeProduct)
		userRoutes.GET("/favorites/status/:product_id", controllers.GetFavoriteStatus)
		userRoutes.PUT("/change-password", controllers.ChangePassword)
		userRoutes.PUT("/shipping-address", controllers.UpdateShippingAddress) // เพิ่มที่อยู่ใหม่
		userRoutes.GET("/addresses", controllers.GetUserAddresses)             // ดึงที่อยู่ทั้งหมด
		userRoutes.DELETE("/addresses/:address_id", controllers.DeleteAddress)
		userRoutes.PUT("/addresses/:address_id", controllers.UpdateAddress)
		// userRoutes.PUT("/update-address-field", controllers.UpdateUserWithAddressField) //เอาไว้อัพฟิลuser ที่ไม่มี
		userRoutes.POST("/products", controllers.AddProduct)
        userRoutes.POST("/become-seller",controllers.BecomeSeller)

	}
}

func SetupProductRoutes(r *gin.Engine) {
	// กลุ่มเส้นทางสำหรับ products
	products := r.Group("/api/products")
	{
		products.GET("", controllers.GetAllProducts)
		products.POST("", controllers.AddProduct)
		products.GET("/:id", controllers.GetProductByID)
		products.PUT("/:id", controllers.UpdateProduct)
		products.DELETE("/:id", controllers.DeleteProduct)
	}

}
func SetupCartRoutes(r *gin.Engine) {
	cart := r.Group("/api/cart", middlewares.AuthMiddleware()) 
	{
		cart.POST("/", controllers.AddToCart)              // เพิ่มสินค้าลงตะกร้า
		cart.GET("/", controllers.GetCart)                 //  ดูสินค้าทั้งหมดในตะกร้า
		cart.DELETE("/:product_id", controllers.RemoveFromCart) // ลบสินค้าออกจากตะกร้า
	}
}
func SetupOrderRoutes(r *gin.Engine) {
	order := r.Group("/api/orders", middlewares.AuthMiddleware())
	{
		order.POST("/", controllers.CreateOrder)   
		order.POST("/custom", controllers.CreateCustomOrder)     // สร้างคำสั่งซื้อจากตะกร้า
		order.GET("/", controllers.GetUserOrders)       //  ดูคำสั่งซื้อทั้งหมดของผู้ใช้
		order.GET("/:id", controllers.GetOrderByID)
		order.POST("/:id/pay", controllers.PayOrder)     //  ดูคำสั่งซื้อรายการเดียว
		order.POST("/:id/confirm", controllers.ConfirmOrderDelivery)
		order.PUT("/:id/tracking", controllers.UpdateTrackingNumber)

	}
}
func PaymentRoutes(r *gin.Engine) {
    payment := r.Group("/api/payment")
    {
        payment.POST("/charge", controllers.CreateTestCharge)
    }
}
func CategoryRoutes(r *gin.Engine) {
    category := r.Group("/api/categories")
    {
        category.GET("/", controllers.GetAllCategories)
        category.POST("/", controllers.AddCategory)
    }
}