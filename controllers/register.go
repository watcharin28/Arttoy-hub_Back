package controllers

import (
    "context"
    "net/http"
    "arttoy-hub/models"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "golang.org/x/crypto/bcrypt"
)

var collection *mongo.Collection

func InitMongo(client *mongo.Client) {
    collection = client.Database("arttoyhub_db").Collection("users")
}

func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func Register(c *gin.Context) {
    type RegisterInput struct {
        Username        string `json:"username" binding:"required"`
        Gmail           string `json:"gmail" binding:"required,email"`
        Phonenumber     string `json:"phonenumber" binding:"required"`
        Password        string `json:"password" binding:"required"`
        ConfirmPassword string `json:"confirmPassword" binding:"required"`
    }

    var input RegisterInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid input",
            "details": err.Error(), // รายละเอียดจาก binding
        })
        return
    }

    // ตรวจสอบว่ารหัสผ่านและการยืนยันรหัสผ่านตรงกัน
    if input.Password != input.ConfirmPassword {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Passwords do not match",
            "field": "password",
        })
        return
    }

    // ตรวจสอบความยาว
    if len(input.Username) < 4 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Username must be at least 4 characters",
            "field": "username",
        })
        return
    }
    if len(input.Password) < 6 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Password must be at least 6 characters",
            "field": "password",
        })
        return
    }

    // ตรวจสอบว่า username ซ้ำหรือไม่
    var existingUser models.User
    err := collection.FindOne(context.Background(), bson.M{"username": input.Username}).Decode(&existingUser)
    if err == nil {
        c.JSON(http.StatusConflict, gin.H{
            "error": "Username already exists",
            "field": "username",
        })
        return
    }

    // ตรวจสอบว่า gmail ซ้ำหรือไม่
    err = collection.FindOne(context.Background(), bson.M{"gmail": input.Gmail}).Decode(&existingUser)
    if err == nil {
        c.JSON(http.StatusConflict, gin.H{
            "error": "Email already exists",
            "field": "gmail",
        })
        return
    }

    // เข้ารหัสพาสเวิร์ด
    hashedPassword, err := hashPassword(input.Password)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    // สร้าง User ใหม่
    user := models.User{
        ID:          primitive.NewObjectID(),
        Username:    input.Username,
        Gmail:       input.Gmail,
        Phonenumber: input.Phonenumber,
        Password:    hashedPassword,
        LikedItems:   []string{},
    }

    // บันทึกผู้ใช้
    result, err := collection.InsertOne(context.Background(), user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Register Successful", "id": result.InsertedID})
}