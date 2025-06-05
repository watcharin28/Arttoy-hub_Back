package controllers

import (
	"arttoy-hub/database"
	"arttoy-hub/models"
	"arttoy-hub/utils"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var otpStore = make(map[string]string)
var tempUserStore = make(map[string]models.User)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}
func InitMongo(client *mongo.Client) {
	collection = client.Database("arttoyhub_db").Collection("users")
}

// STEP 1: ขอ OTP ทางอีเมล
func RequestOTP(c *gin.Context) {
	type RegisterInput struct {
		Username        string `json:"username" binding:"required"`
		Gmail           string `json:"gmail" binding:"required,email"`
		Phonenumber     string `json:"phonenumber" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}

	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	if input.Password != input.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
		return
	}

	// ตรวจ username และ gmail ซ้ำ
	userCollection := db.Client.Database("arttoyhub_db").Collection("users")
	var existing models.User
	err := userCollection.FindOne(context.Background(), bson.M{"username": input.Username}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}
	err = userCollection.FindOne(context.Background(), bson.M{"gmail": input.Gmail}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}
	err = userCollection.FindOne(context.Background(), bson.M{"phonenumber": input.Phonenumber}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Phone number already exists"})
		return
	}

	// สุ่ม OTP
	rand.Seed(time.Now().UnixNano())
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))

	// เข้ารหัสรหัสผ่าน
	hashed, _ := hashPassword(input.Password)

	// เก็บ user ชั่วคราว
	tempUser := models.User{
		ID:          primitive.NewObjectID(),
		Username:    input.Username,
		Gmail:       input.Gmail,
		Phonenumber: input.Phonenumber,
		Password:    hashed,
		LikedItems:  []string{},
	}

	otpStore[input.Gmail] = otp
	tempUserStore[input.Gmail] = tempUser

	// ส่งอีเมล OTP
	if err := utils.SendEmail(input.Gmail, otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to email"})
}
func VerifyOTP(c *gin.Context) {
	type VerifyInput struct {
		Gmail string `json:"gmail" binding:"required,email"`
		OTP   string `json:"otp" binding:"required"`
	}

	var input VerifyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	expectedOTP := otpStore[input.Gmail]
	if expectedOTP != input.OTP {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect OTP"})
		return
	}

	user := tempUserStore[input.Gmail]
	userCollection := db.Client.Database("arttoyhub_db").Collection("users")
	result, err := userCollection.InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
		return
	}

	delete(otpStore, input.Gmail)
	delete(tempUserStore, input.Gmail)

	c.JSON(http.StatusOK, gin.H{"message": "Register successful", "user_id": result.InsertedID})
}
