package helpers

import (
	"bnsp2/server/config"
	"bytes"
	"fmt"
	"io"

	"gopkg.in/gomail.v2"
)

func SendOTPEmail(to string, otp string) error {

	m := gomail.NewMessage()
	m.SetHeader("From", "subairibairi689@gmail.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Kode Verifikasi Email")

	body := fmt.Sprintf("Kode OTP anda adalah: %s", otp)

	m.SetBody("text/plain", body)

	d := gomail.NewDialer(
		"smtp.gmail.com",
		587,
		"subairibairi689@gmail.com",
		"osuk ykvs vwor dbeh",
	)

	return d.DialAndSend(m)
}

func SendResetPasswordEmail(to string, token string) error {
	config.LoadEnv()

	m := gomail.NewMessage()
	m.SetHeader("From", "subairibairi689@gmail.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Reset Password")

	// link reset (sesuaikan dengan frontend kamu)
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.GetEnv("DOMAIN", "http://localhost:5173"), token)

	body := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif;">

			<h2>Reset Password</h2>
			<p>Klik tombol di bawah untuk mereset password Anda:</p>

			<a href="%s" 
			   style="
				display: inline-block;
				padding: 12px 20px;
				font-size: 16px;
				color: white;
				background-color: #2563eb;
				text-decoration: none;
				border-radius: 8px;
			   ">
				Reset Password
			</a>

			<p style="margin-top:20px;">
				Link ini hanya berlaku selama 10 menit.
			</p>

			<p>Jika Anda tidak meminta reset password, abaikan email ini.</p>

		</body>
		</html>
	`, resetLink)

	m.SetBody("text/html", body)

	d := gomail.NewDialer(
		"smtp.gmail.com",
		587,
		"subairibairi689@gmail.com",
		"osuk ykvs vwor dbeh",
	)

	return d.DialAndSend(m)
}

func SendInvoiceEmail(to string, pdfBuffer *bytes.Buffer, invoiceID string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "subairibairi689@gmail.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Invoice "+invoiceID)

	m.SetBody("text/html", "<p>Invoice terlampir</p>")

	m.Attach(invoiceID+".pdf", gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(pdfBuffer.Bytes())
		return err
	}))

	d := gomail.NewDialer(
		"smtp.gmail.com",
		587,
		"subairibairi689@gmail.com",
		"osuk ykvs vwor dbeh",
	)

	return d.DialAndSend(m)
}
