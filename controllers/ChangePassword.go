package controllers

import (
    "context"
    "net/http"
    "time"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "golang.org/x/crypto/bcrypt"
    "github.com/gin-gonic/gin"
    "arttoy-hub/database"
	"arttoy-hub/models"
)

// ChangePassword เปลี่ยนรหัสผ่านผู้ใช้
func ChangePassword(c *gin.Context) {
    type PasswordInput struct {
        OldPassword string `json:"oldPassword" binding:"required"`
        NewPassword string `json:"newPassword" binding:"required"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
    }
	

    var input PasswordInput
	if input.NewPassword != input.ConfirmPassword {
        c.JSON(http.StatusBadRequest, gin.H{"error": "New password and confirmation do not match"})
        return
    }


    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    objID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var user models.User
    err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
        return
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword))
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
        return
    }

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
        return
    }

    _, err = db.OpenCollection("users").UpdateOne(ctx,
        bson.M{"_id": objID},
        bson.M{"$set": bson.M{"password": string(hashedPassword)}},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
