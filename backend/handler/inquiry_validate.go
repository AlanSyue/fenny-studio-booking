package handler

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Limits for the public inquiry form. String limits are in characters (runes),
// matching Postgres VARCHAR semantics, and are no larger than the columns.
const (
	maxInquiryBodyBytes = 64 << 10

	maxNameLen              = 200 // VARCHAR(200)
	maxEmailLen             = 254 // RFC 5321 path limit; column is VARCHAR(255)
	maxCityLen              = 50  // VARCHAR(50)
	maxDistrictLen          = 50  // VARCHAR(50)
	maxAddressLen           = 255 // VARCHAR(255)
	maxHeadcountLen         = 50  // VARCHAR(50)
	maxBudgetLen            = 1000
	maxAdditionalContactLen = 2000

	maxCourseIDs = 50

	// Headcount is free text ("10-20人", "50人以上", ...). When a client sends
	// a plain number we still keep it in a sane range.
	minHeadcount = 1
	maxHeadcount = 10000
)

// validationError is returned for client mistakes that map to 400.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func invalidf(format string, args ...any) error {
	return &validationError{msg: fmt.Sprintf(format, args...)}
}

func checkLen(field string, v *string, max int) error {
	if v != nil && utf8.RuneCountInString(*v) > max {
		return invalidf("%s is too long (max %d characters)", field, max)
	}
	return nil
}

// isValidEmail does a basic syntax check: a bare addr-spec (no display name),
// a non-empty local part and a dotted domain.
func isValidEmail(s string) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	at := strings.LastIndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// validateInquiry enforces required fields, length limits and basic sanity on
// a public inquiry submission. It returns a *validationError on bad input.
func validateInquiry(req *createInquiryRequest) error {
	if req.Email == "" {
		return invalidf("Email is required")
	}
	if req.City == nil || *req.City == "" {
		return invalidf("City is required")
	}
	if req.District == nil || *req.District == "" {
		return invalidf("District is required")
	}

	email := req.Email
	if err := checkLen("email", &email, maxEmailLen); err != nil {
		return err
	}
	if !isValidEmail(req.Email) {
		return invalidf("Email format is invalid")
	}

	for _, f := range []struct {
		name string
		v    *string
		max  int
	}{
		{"name", req.Name, maxNameLen},
		{"city", req.City, maxCityLen},
		{"district", req.District, maxDistrictLen},
		{"address", req.Address, maxAddressLen},
		{"headcount", req.Headcount, maxHeadcountLen},
		{"budget", req.Budget, maxBudgetLen},
		{"additional_contact", req.AdditionalContact, maxAdditionalContactLen},
	} {
		if err := checkLen(f.name, f.v, f.max); err != nil {
			return err
		}
	}

	if req.Headcount != nil {
		if n, err := strconv.Atoi(strings.TrimSpace(*req.Headcount)); err == nil {
			if n < minHeadcount || n > maxHeadcount {
				return invalidf("headcount must be between %d and %d", minHeadcount, maxHeadcount)
			}
		}
	}

	if len(req.CourseIDs) > maxCourseIDs {
		return invalidf("too many courses (max %d)", maxCourseIDs)
	}
	for _, id := range req.CourseIDs {
		if id <= 0 {
			return invalidf("invalid course id")
		}
	}
	if req.CourseID != nil && *req.CourseID <= 0 {
		return invalidf("invalid course id")
	}
	return nil
}
