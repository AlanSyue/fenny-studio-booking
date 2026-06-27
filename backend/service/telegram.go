package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/model"
)

// Notifier sends inquiry notifications to a Telegram chat via the Bot API.
type Notifier struct {
	botToken string
	chatID   string
	http     *http.Client
}

// NewTelegramNotifier builds a Notifier. If botToken or chatID is empty the
// resulting Notifier is a no-op (see NotifyInquiry).
func NewTelegramNotifier(botToken, chatID string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatID:   chatID,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// telegramMessage is the JSON payload for the sendMessage Bot API method.
type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// NotifyInquiry sends a formatted notification about a newly received inquiry.
// It is a no-op (returns nil) when the bot token or chat ID is not configured,
// so the system keeps working before .env is filled in.
func (n *Notifier) NotifyInquiry(inquiry model.Inquiry) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}

	text := buildInquiryMessage(inquiry)

	payload := telegramMessage{
		ChatID:    n.chatID,
		Text:      text,
		ParseMode: "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.http.Do(req)
	if err != nil {
		return fmt.Errorf("call telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// buildInquiryMessage assembles the HTML message field by field, skipping empty
// values. All user-provided values are HTML-escaped before insertion.
func buildInquiryMessage(inquiry model.Inquiry) string {
	var b strings.Builder

	b.WriteString("📩 <b>收到新詢問</b>\n")

	appendField(&b, "姓名", derefString(inquiry.Name))
	appendField(&b, "Email", inquiry.Email)

	if courses := courseNames(inquiry.Courses); courses != "" {
		appendField(&b, "課程", courses)
	}

	appendField(&b, "人數", derefString(inquiry.Headcount))
	appendField(&b, "城市", derefString(inquiry.City))
	appendField(&b, "區", derefString(inquiry.District))
	appendField(&b, "地址", derefString(inquiry.Address))
	appendField(&b, "預算", derefString(inquiry.Budget))
	appendField(&b, "其他聯絡方式", derefString(inquiry.AdditionalContact))

	return strings.TrimRight(b.String(), "\n")
}

// appendField writes "label: value\n" when value is non-empty, escaping value.
func appendField(b *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(label))
	b.WriteString("</b>: ")
	b.WriteString(html.EscapeString(value))
	b.WriteString("\n")
}

// courseNames joins non-empty course names with a Chinese enumeration comma.
func courseNames(courses []model.Course) string {
	names := make([]string, 0, len(courses))
	for _, c := range courses {
		if strings.TrimSpace(c.Name) != "" {
			names = append(names, c.Name)
		}
	}
	return strings.Join(names, "、")
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
