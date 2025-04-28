package controllers
import (
    "context"
    "net/http"
    "time"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "github.com/gin-gonic/gin"
    "arttoy-hub/database"
	"arttoy-hub/models"
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
        Address     string `json:"address" binding:"required"`
        Province    string `json:"province" binding:"required"`
        PostalCode  string `json:"postalCode" binding:"required"`
        PhoneNumber string `json:"phoneNumber" binding:"required"`
        IsDefault   bool   `json:"isDefault"` // รับมาด้วย (optional)
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

    if input.IsDefault {
        // 1. Set isDefault ของที่อยู่ทั้งหมดเป็น false
        _, err = addressesCollection.UpdateOne(
            ctx,
            bson.M{"_id": objID},
            bson.M{"$set": bson.M{"addresses.$[].isDefault": false}},
        )
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing addresses"})
            return
        }
    }

    // 2. เพิ่มที่อยู่ใหม่
    newAddress := models.Address{
        ID:          primitive.NewObjectID(),
        Address:     input.Address,
        Province:    input.Province,
        PostalCode:  input.PostalCode,
        PhoneNumber: input.PhoneNumber,
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

    c.JSON(http.StatusOK, gin.H{"message": "Shipping address added successfully"})
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
// func UpdateUserWithAddressField(c *gin.Context) {
//     // ค้นหาผู้ใช้ทั้งหมดในฐานข้อมูลที่ไม่มีฟิลด์ addresses
//     ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//     defer cancel()

//     cursor, err := db.OpenCollection("users").Find(ctx, bson.M{"addresses": bson.M{"$exists": false}})
//     if err != nil {
//         c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find users"})
//         return
//     }
//     defer cursor.Close(ctx)

//     // อัปเดตทุกผู้ใช้ที่ไม่มีฟิลด์ addresses
//     for cursor.Next(ctx) {
//         var user models.User
//         if err := cursor.Decode(&user); err != nil {
//             continue
//         }

//         // ถ้าผู้ใช้ไม่มีที่อยู่, ให้เพิ่มที่อยู่ว่างๆ
//         _, err := db.OpenCollection("users").UpdateOne(
//             ctx,
//             bson.M{"_id": user.ID},
//             bson.M{"$set": bson.M{"addresses": []models.Address{}}}, // เพิ่มฟิลด์ addresses ว่าง
//         )
//         if err != nil {
//             log.Printf("Failed to update user %v: %v", user.ID, err)
//             continue
//         }
//     }

//     c.JSON(http.StatusOK, gin.H{"message": "User data updated successfully"})
// }
func SetDefaultAddress(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    var input struct {
        AddressID string `json:"addressId" binding:"required"`
    }

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    addressObjID, err := primitive.ObjectIDFromHex(input.AddressID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
        return
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    usersCollection := db.OpenCollection("users")

    // 1. Set ทุก address ของ user ให้ isDefault เป็น false
    _, err = usersCollection.UpdateOne(
        ctx,
        bson.M{"_id": userObjID},
        bson.M{"$set": bson.M{"addresses.$[].isDefault": false}},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear default addresses"})
        return
    }

    // 2. Set address ที่เลือก ให้ isDefault เป็น true
    filter := bson.M{
        "_id": userObjID,
        "addresses._id": addressObjID,
    }

    update := bson.M{
        "$set": bson.M{
            "addresses.$.isDefault": true,
        },
    }

    result, err := usersCollection.UpdateOne(ctx, filter, update)
    if err != nil || result.MatchedCount == 0 {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default address"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Default address set successfully"})
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
        Address    string `json:"address" binding:"required"`
        Province   string `json:"province" binding:"required"`
        PostalCode string `json:"postalCode" binding:"required"`
        PhoneNumber string `json:"phoneNumber" binding:"required"`
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

    filter := bson.M{
        "_id": userObjID,
        "addresses._id": addressObjID,
    }

    update := bson.M{
        "$set": bson.M{
            "addresses.$.address":    input.Address,
            "addresses.$.province":   input.Province,
            "addresses.$.postalCode": input.PostalCode,
            "addresses.$.phoneNumber": input.PhoneNumber,
        },
    }

    result, err := usersCollection.UpdateOne(ctx, filter, update)
    if err != nil || result.MatchedCount == 0 {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Address updated successfully"})
}
