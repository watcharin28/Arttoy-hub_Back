package controllers

import (
    "context"
    "net/http"
    "os"
    "time"
    "arttoy-hub/models"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "golang.org/x/crypto/bcrypt"
    "github.com/golang-jwt/jwt/v4"
    // "fmt"
)
var collection *mongo.Collection
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ตรวจสอบพาสเวิร์ด
func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// สร้าง JWT Token
func generateJWT(userID string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    })
    return token.SignedString(jwtSecret)
}

// Handler สำหรับ Login ด้วยเบอร์โทรศัพท์ + รหัสผ่าน
func Login(c *gin.Context) {
    type LoginInput struct {
        Phonenumber string `json:"phonenumber" binding:"required"`
        Password    string `json:"password" binding:"required"`
    }

    var input LoginInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    // ค้นหาผู้ใช้ตามเบอร์โทรศัพท์
    var user models.User
    err := collection.FindOne(context.Background(), bson.M{"phonenumber": input.Phonenumber}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid phone number or password"})
        return
    }

    // ตรวจสอบพาสเวิร์ด
    if !checkPasswordHash(input.Password, user.Password) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid phone number or password"})
        return
    }

    // สร้าง JWT Token
    token, err := generateJWT(user.ID.Hex())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }
    // ตั้งค่า cookie
    http.SetCookie(c.Writer, &http.Cookie{
        Name:     "token",
        Value:    token,
        Path:     "/",
        Domain:   "", // ต้องตรงกับ backend
        MaxAge:   3600 * 24,                      // 1 วัน
        Secure:   true,                           // HTTPS เท่านั้น
        HttpOnly: false,
        SameSite: http.SameSiteNoneMode,          // ← สำคัญ!
    })

    c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}
// c.Writer.Header().Set("Set-Cookie", fmt.Sprintf("token=%s; SameSite=None; Path=/;", token)) // ลบ HttpOnly ออก
    


