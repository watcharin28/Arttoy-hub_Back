package utils

import (
    "fmt"
    "net/smtp"
    "os"
)

func SendEmail(toEmail, otp string) error {
    from := os.Getenv("EMAIL_FROM")
    pass := os.Getenv("EMAIL_PASS")

    msg := fmt.Sprintf(`Subject: ArtToyHub - OTP Verification

Dear user,

Your One-Time Password (OTP) for verifying your email is:

🔐 OTP: %s

Please enter this code to complete your registration.

Thank you,
ArtToyHub Team
`, otp)

    return smtp.SendMail(
        "smtp.gmail.com:587",
        smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
        from,
        []string{toEmail},
        []byte(msg),
    )
}
