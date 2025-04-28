package models

type BecomeSellerRequest struct {
    FirstName          string `json:"first_name" binding:"required"`
    LastName           string `json:"last_name" binding:"required"`
    BankAccountName    string `json:"bank_account_name" binding:"required"`
    BankName           string `json:"bank_name" binding:"required"`
    BankAccountNumber  string `json:"bank_account_number" binding:"required"`
    CitizenID          string `json:"citizen_id" binding:"required"`
    IDCardImageURL     string `json:"id_card_image_url" binding:"required"`
}
