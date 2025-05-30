package controllers

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"

	"arttoy-hub/database"
	"arttoy-hub/models"
	"github.com/gin-gonic/gin"
	// "log"
)

// UpdateShippingAddress อัปเดตที่อยู่ของผู้ใช้
func UpdateShippingAddress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		Phone       string `json:"phone" binding:"required"`
		Address     string `json:"address" binding:"required"`
		Subdistrict string `json:"subdistrict" binding:"required"`
		District    string `json:"district" binding:"required"`
		Province    string `json:"province" binding:"required"`
		Zipcode     string `json:"zipcode" binding:"required"`
		IsDefault   bool   `json:"isDefault"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addressesCollection := db.OpenCollection("users")

	// แปลง addresses:null → [] ถ้ายังไม่มี
	_, _ = addressesCollection.UpdateOne(ctx, bson.M{
		"_id":      objID,
		"addresses": bson.M{"$type": "null"},
	}, bson.M{
		"$set": bson.M{"addresses": []interface{}{}},
	})

	// ถ้าเป็น default → set isDefault: false ให้ทุก address อื่น
	if input.IsDefault {
		_, err = addressesCollection.UpdateMany(
			ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{"addresses.$[elem].isDefault": false}},
			options.Update().SetArrayFilters(options.ArrayFilters{
				Filters: []interface{}{bson.M{"elem.isDefault": true}},
			}),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing addresses"})
			return
		}
	}

	//  3. เพิ่ม address ใหม่
	newAddress := models.Address{
		ID:          primitive.NewObjectID(),
		Name:        input.Name,
		Phone:       input.Phone,
		Address:     input.Address,
		Subdistrict: input.Subdistrict,
		District:    input.District,
		Province:    input.Province,
		Zipcode:     input.Zipcode,
		IsDefault:   input.IsDefault,
	}

	_, err = addressesCollection.UpdateOne(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$push": bson.M{"addresses": newAddress}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add new address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shipping address added successfully", "address": newAddress})
}


// UpdateShippingAddress อัปเดตที่อยู่ของผู้ใช้
// GetUserAddresses ดึงที่อยู่ทั้งหมดของผู้ใช้
func GetUserAddresses(c *gin.Context) {
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

	// เชื่อมต่อกับฐานข้อมูล
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = db.OpenCollection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// ส่งที่อยู่ทั้งหมดกลับ
	c.JSON(http.StatusOK, gin.H{"addresses": user.Addresses})
}

func DeleteAddress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addressID := c.Param("address_id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	addressObjID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := db.OpenCollection("users")

	// ดึง user ก่อน เพื่อเช็คว่าที่อยู่ที่จะลบเป็น default หรือเปล่า
	var user models.User
	err = usersCollection.FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var isDefault bool
	for _, addr := range user.Addresses {
		if addr.ID == addressObjID {
			isDefault = addr.IsDefault
			break
		}
	}

	// ลบ address
	_, err = usersCollection.UpdateOne(
		ctx,
		bson.M{"_id": userObjID},
		bson.M{"$pull": bson.M{"addresses": bson.M{"_id": addressObjID}}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	// ถ้า address ที่ลบเป็น default => ตั้ง address แรกที่เหลือให้เป็น default
	if isDefault {
		// ดึง user ใหม่หลังลบ
		err = usersCollection.FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user)
		if err == nil && len(user.Addresses) > 0 {
			firstAddressID := user.Addresses[0].ID
			_, _ = usersCollection.UpdateOne(
				ctx,
				bson.M{"_id": userObjID, "addresses._id": firstAddressID},
				bson.M{"$set": bson.M{"addresses.$.isDefault": true}},
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted successfully"})
}
func UpdateAddress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addressID := c.Param("address_id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		Phone       string `json:"phone" binding:"required"`
		Address     string `json:"address" binding:"required"`
		Subdistrict string `json:"subdistrict" binding:"required"`
		District    string `json:"district" binding:"required"`
		Province    string `json:"province" binding:"required"`
		Zipcode     string `json:"zipcode" binding:"required"`
		IsDefault   bool   `json:"isDefault"` // optional
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	addressObjID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection := db.OpenCollection("users")
	// ถ้า input ต้องการตั้งที่อยู่นี้ให้เป็น default
	if input.IsDefault {
		// Set isDefault = false สำหรับที่อยู่ทั้งหมดของ user ก่อน
		_, err := usersCollection.UpdateOne(
			ctx,
			bson.M{"_id": userObjID},
			bson.M{
				"$set": bson.M{"addresses.$[].isDefault": false},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset default addresses"})
			return
		}
	}
	filter := bson.M{
		"_id":           userObjID,
		"addresses._id": addressObjID,
	}

	update := bson.M{
		"$set": bson.M{
			"addresses.$.name":        input.Name,
			"addresses.$.phone":       input.Phone,
			"addresses.$.address":     input.Address,
			"addresses.$.subdistrict": input.Subdistrict,
			"addresses.$.district":    input.District,
			"addresses.$.province":    input.Province,
			"addresses.$.zipcode":     input.Zipcode,
			"addresses.$.isDefault":   input.IsDefault,
		},
	}

	result, err := usersCollection.UpdateOne(ctx, filter, update)
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address updated successfully"})
}
