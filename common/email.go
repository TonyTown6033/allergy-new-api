package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type EmailAttachment struct {
	FileName    string
	ContentType string
	Path        string
}

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func SendEmail(subject string, receiver string, content string) error {
	return SendEmailWithAttachments(subject, receiver, content, nil)
}

func SendEmailWithAttachments(subject string, receiver string, content string, attachments []EmailAttachment) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}

	mail, err := buildEmailMessage(subject, receiver, content, attachments)
	if err != nil {
		return err
	}
	return sendRawEmail(receiver, mail)
}

func buildEmailMessage(subject string, receiver string, content string, attachments []EmailAttachment) ([]byte, error) {
	id, err := generateMessageID()
	if err != nil {
		return nil, err
	}

	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	var header strings.Builder
	header.WriteString(fmt.Sprintf("To: %s\r\n", receiver))
	header.WriteString(fmt.Sprintf("From: %s <%s>\r\n", SystemName, SMTPFrom))
	header.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	header.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	header.WriteString(fmt.Sprintf("Message-ID: %s\r\n", id))
	header.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		header.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		header.WriteString(content)
		header.WriteString("\r\n")
		return []byte(header.String()), nil
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%q\r\n\r\n", writer.Boundary()))

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="UTF-8"`)
	htmlPart, err := writer.CreatePart(htmlHeader)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(htmlPart, content); err != nil {
		return nil, err
	}

	for _, attachment := range attachments {
		if strings.TrimSpace(attachment.Path) == "" {
			return nil, fmt.Errorf("attachment path is required")
		}
		data, err := os.ReadFile(attachment.Path)
		if err != nil {
			return nil, err
		}
		fileName := strings.TrimSpace(attachment.FileName)
		if fileName == "" {
			fileName = filepath.Base(attachment.Path)
		}
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", fmt.Sprintf("%s; name=%q", contentType, fileName))
		partHeader.Set("Content-Transfer-Encoding", "base64")
		partHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, err
		}
		if err := writeBase64Lines(part, data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return append([]byte(header.String()), body.Bytes()...), nil
}

func writeBase64Lines(w io.Writer, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	for len(encoded) > 76 {
		if _, err := io.WriteString(w, encoded[:76]+"\r\n"); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if _, err := io.WriteString(w, encoded+"\r\n"); err != nil {
		return err
	}
	return nil
}

func sendRawEmail(receiver string, mail []byte) error {
	auth := smtp.PlainAuth("", SMTPAccount, SMTPToken, SMTPServer)
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	if SMTPPort == 465 || SMTPSSLEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         SMTPServer,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", SMTPServer, SMTPPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(SMTPFrom); err != nil {
			return err
		}
		receiverEmails := strings.Split(receiver, ";")
		for _, receiver := range receiverEmails {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(mail)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else if isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer) {
		auth = LoginAuth(SMTPAccount, SMTPToken)
		err = smtp.SendMail(addr, auth, SMTPFrom, to, mail)
	} else {
		err = smtp.SendMail(addr, auth, SMTPFrom, to, mail)
	}
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}
