package service

import (
	"strings"
	"testing"

	"github.com/AlanSyue/fenny-studio-booking/backend/model"
)

func TestBuildInquiryMessageIncludesInquiryNo(t *testing.T) {
	name := "王小明"
	inquiry := model.Inquiry{
		InquiryNo: "FEN-20260805-001",
		Name:      &name,
		Email:     "a@example.com",
	}

	got := buildInquiryMessage(inquiry)

	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.Contains(firstLine, "FEN-20260805-001") {
		t.Errorf("首行應包含詢問編號，實際得到 %q", firstLine)
	}
	if !strings.Contains(firstLine, "收到新詢問") {
		t.Errorf("首行應保留原標題，實際得到 %q", firstLine)
	}
}

func TestBuildInquiryMessageWithoutInquiryNo(t *testing.T) {
	inquiry := model.Inquiry{Email: "a@example.com"}

	got := buildInquiryMessage(inquiry)

	firstLine := strings.SplitN(got, "\n", 2)[0]
	if strings.HasSuffix(firstLine, " ") {
		t.Errorf("編號為空時首行不應留下多餘空白，實際得到 %q", firstLine)
	}
}
