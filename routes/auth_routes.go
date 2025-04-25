package routes

import (
    "arttoy-hub/controllers"
    "github.com/gin-gonic/gin"
    "arttoy-hub/middleware"
)

func SetupAuthRoutes(r *gin.Engine) {
        // Public user routes
        r.POST("/Login", controllers.Login) // Login route
        r.POST("/Register", controllers.Register) // Register route
    
        // ใช้ AuthMiddleware กับ routes ที่ต้องการให้ login ก่อน
        userRoutes := r.Group("/api/user")
        userRoutes.Use(middlewares.AuthMiddleware()) // ต้อง login ก่อน
        {
            userRoutes.PUT("/Profile", controllers.UpdateProfile)
            userRoutes.GET("/Profile", controllers.GetProfile)
            userRoutes.GET("/favorites", controllers.GetUserFavorites)
            userRoutes.DELETE("/favorites/:product_id", controllers.DeleteUserFavorite)
            userRoutes.POST("/favorites/:product_id", controllers.LikeProduct)
            userRoutes.GET("/favorites/status/:product_id", controllers.GetFavoriteStatus)
            

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