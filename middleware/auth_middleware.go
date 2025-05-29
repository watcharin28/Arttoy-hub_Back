package middlewares

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"context"
	"os"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		fmt.Println("Token from cookie:", tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			fmt.Println("Claims:", claims)
			userID, exists := claims["user_id"].(string)
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID missing in token"})
				c.Abort()
				return
			}
			c.Set("user_id", userID) // เก็บ user_id ลง context
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
		}
	}
}
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// ดึงข้อมูลผู้ใช้จาก MongoDB
		objID, _ := primitive.ObjectIDFromHex(userID)
		userCollection := db.OpenCollection("users")
		var user models.User
		err := userCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "ไม่พบข้อมูลผู้ใช้"})
			c.Abort()
			return
		}

		// เงื่อนไข: ถ้า gmail ตรงกับ admin
		if user.Gmail != "kkong@mail.com" {
			c.JSON(http.StatusForbidden, gin.H{"error": "เฉพาะผู้ดูแลระบบเท่านั้น"})
			c.Abort()
			return
		}

		c.Next()
	}
}
