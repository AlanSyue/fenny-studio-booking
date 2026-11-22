package handler

import (
	"errors"
	"strings"
	"testing"
)

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

func validInquiry() createInquiryRequest {
	return createInquiryRequest{
		Name:              strp("王小明"),
		Email:             "someone@example.com",
		CourseIDs:         []int{1, 2},
		Headcount:         strp("10-20人"),
		City:              strp("臺北市"),
		District:          strp("大安區"),
		Address:           strp("忠孝東路四段 1 號"),
		Budget:            strp("每人 1500"),
		AdditionalContact: strp("LINE: abc"),
	}
}

func TestValidateInquiryAcceptsValid(t *testing.T) {
	req := validInquiry()
	if err := validateInquiry(&req); err != nil {
		t.Fatalf("valid inquiry rejected: %v", err)
	}
	// Optional fields may be nil.
	req = createInquiryRequest{Email: "a@b.co", City: strp("臺北市"), District: strp("大安區")}
	if err := validateInquiry(&req); err != nil {
		t.Fatalf("minimal inquiry rejected: %v", err)
	}
	// Limits are in characters, not bytes: 200 CJK chars is fine.
	req = validInquiry()
	req.Name = strp(strings.Repeat("花", maxNameLen))
	if err := validateInquiry(&req); err != nil {
		t.Fatalf("name at limit rejected: %v", err)
	}
	req = validInquiry()
	req.Headcount = strp("30")
	if err := validateInquiry(&req); err != nil {
		t.Fatalf("numeric headcount rejected: %v", err)
	}
}

func TestValidateInquiryRejects(t *testing.T) {
	long := func(n int) *string { return strp(strings.Repeat("x", n)) }
	ids := make([]int, maxCourseIDs+1)
	for i := range ids {
		ids[i] = i + 1
	}

	cases := []struct {
		name   string
		mutate func(r *createInquiryRequest)
	}{
		{"missing email", func(r *createInquiryRequest) { r.Email = "" }},
		{"missing city", func(r *createInquiryRequest) { r.City = nil }},
		{"empty district", func(r *createInquiryRequest) { r.District = strp("") }},
		{"email no at", func(r *createInquiryRequest) { r.Email = "not-an-email" }},
		{"email no domain dot", func(r *createInquiryRequest) { r.Email = "a@localhost" }},
		{"email display name", func(r *createInquiryRequest) { r.Email = "Bob <bob@example.com>" }},
		{"email spaces", func(r *createInquiryRequest) { r.Email = " bob@example.com" }},
		{"email too long", func(r *createInquiryRequest) { r.Email = strings.Repeat("a", 250) + "@example.com" }},
		{"name too long", func(r *createInquiryRequest) { r.Name = long(maxNameLen + 1) }},
		{"city too long", func(r *createInquiryRequest) { r.City = long(maxCityLen + 1) }},
		{"district too long", func(r *createInquiryRequest) { r.District = long(maxDistrictLen + 1) }},
		{"address too long", func(r *createInquiryRequest) { r.Address = long(maxAddressLen + 1) }},
		{"headcount too long", func(r *createInquiryRequest) { r.Headcount = long(maxHeadcountLen + 1) }},
		{"budget too long", func(r *createInquiryRequest) { r.Budget = long(maxBudgetLen + 1) }},
		{"additional contact too long", func(r *createInquiryRequest) { r.AdditionalContact = long(maxAdditionalContactLen + 1) }},
		{"headcount zero", func(r *createInquiryRequest) { r.Headcount = strp("0") }},
		{"headcount negative", func(r *createInquiryRequest) { r.Headcount = strp("-5") }},
		{"headcount huge", func(r *createInquiryRequest) { r.Headcount = strp("99999999") }},
		{"too many courses", func(r *createInquiryRequest) { r.CourseIDs = ids }},
		{"non-positive course id", func(r *createInquiryRequest) { r.CourseIDs = []int{1, 0} }},
		{"non-positive legacy course id", func(r *createInquiryRequest) { r.CourseID = intp(-1) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := validInquiry()
			c.mutate(&req)
			err := validateInquiry(&req)
			var ve *validationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected validationError, got %v", err)
			}
		})
	}
}
