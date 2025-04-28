package controllers

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func Logout(c *gin.Context) {
    // ลบ cookie ที่ชื่อว่า "token"
    c.SetCookie(
        "token",     // ชื่อ cookie
        "",          // ค่าใหม่ = ค่าว่าง
        -1,          // อายุ cookie = หมดอายุทันที
        "/",         // path
        "localhost", // domain (แก้ทีหลังถ้า deploy จริง)
        false,       // secure (false สำหรับ localhost)
        true,        // httpOnly (กัน javascript อ่าน cookie)
    )

    c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
