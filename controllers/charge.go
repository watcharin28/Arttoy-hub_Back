package controllers

import (
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/omise/omise-go"
    "github.com/omise/omise-go/operations"
)

func CreateTestCharge(c *gin.Context) {
    client, err := omise.NewClient(
        os.Getenv("OMISE_PUBLIC_KEY"),
        os.Getenv("OMISE_SECRET_KEY"),
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Omise client init failed"})
        return
    }

    var req struct {
        Amount int64  `json:"amount"` // 500.00 = 50000
        Token  string `json:"token"`  // เช่น "tokn_test_visa_4242"
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    charge := &omise.Charge{}
    op := &operations.CreateCharge{
        Amount:   req.Amount,
        Currency: "thb",
        Card:     req.Token,
    }

    if err := client.Do(charge, op); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "charge_id": charge.ID,
        "status":    charge.Status,
        "paid":      charge.Paid,
    })
}
