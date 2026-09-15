package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	italyDataDirectory              = "data/it-2026a"
	italyAgeWeightsSHA256           = "5abdc04bb817e5fc300ce71920c1dedf269551d3cc0065148264d26a39c84555"
	italyPlacesSHA256               = "27539c5025412c2f03b199cbd6fdea7b9bda8fb57f6d8665883c70eeaa7b05b6"
	italyPostalCodesSHA256          = "0fed3e2822c13a54c4255d2b83fe0fd63aea27b432d63a682b28f64f4981426c"
	italyPostalOverridesSHA256      = "92df74687d0b20b00edf82eb4d9f7aacf1afaea9830bcc998d9a8454ef5636ed"
	italyHumanReviewSHA256          = "e2fe588b80e75657c4206f05f169fc32c8c8f402e1f84b22caebe3251116eb3e"
	italyPostalOverrideReviewSHA256 = "056ac3731a16f9862a552537762fd93501f1631ef52028538473a621e5d07625"
	italyProposedPostalReviewSHA256 = "f6398dd8a922a1988017f7386679e835cfc85df2bdfd11dded542077dfd7e3d0"
)

type italyReviewRecord struct {
	MarketDataVersion    string  `json:"market_data_version"`
	D1AEmpiricalReview   string  `json:"d1a_empirical_review"`
	PostalOverrideReview string  `json:"postal_override_review"`
	PlausibilityReview   string  `json:"plausibility_review"`
	Reviewer             *string `json:"reviewer"`
	ReviewTimestamp      *string `json:"review_timestamp"`
	ReviewedArtifacts    map[string]struct {
		SHA256 string `json:"sha256"`
	} `json:"reviewed_artifacts"`
}

// TestItalyAgeWeights validates the frozen age distribution.
func TestItalyAgeWeights(t *testing.T) {

	rows := readItalyCSV(t, "age_weights.csv", italyAgeWeightsSHA256, []string{"age", "weight"})
	if len(rows) != 55 {
		t.Fatalf("expected 55 age rows, got %d", len(rows))
	}

	for position, row := range rows {
		age, err := strconv.Atoi(row[0])
		if err != nil {
			t.Fatalf("expected age at row %d to be numeric, got %q", position+2, row[0])
		}
		expectedAge := position + 21
		if age != expectedAge {
			t.Fatalf("expected age at row %d to be %d, got %d", position+2, expectedAge, age)
		}
		weight, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			t.Fatalf("expected weight at row %d to be uint64, got %q", position+2, row[1])
		}
		if weight == 0 {
			t.Fatalf("expected weight at row %d to be positive, got %d", position+2, weight)
		}
	}

}

// TestItalyPlaces validates the frozen municipality universe and population.
func TestItalyPlaces(t *testing.T) {

	rows := readItalyCSV(
		t,
		"places.csv",
		italyPlacesSHA256,
		[]string{"istat_code", "city", "state_prov", "region", "population"},
	)
	if len(rows) != 7896 {
		t.Fatalf("expected 7896 place rows, got %d", len(rows))
	}

	regions := map[string]struct{}{}
	previousCode := ""
	for position, row := range rows {
		code := row[0]
		if !isASCIIDigits(code, 6) {
			t.Fatalf("expected six-digit IstatCode at row %d, got %q", position+2, code)
		}
		if previousCode != "" && code <= previousCode {
			t.Fatalf("expected IstatCode after %q at row %d, got %q", previousCode, position+2, code)
		}
		previousCode = code
		if row[1] == "" {
			t.Fatalf("expected non-empty city at row %d, got %q", position+2, row[1])
		}
		stateProv := row[2]
		if len(stateProv) != 2 || stateProv[0] < 'A' || stateProv[0] > 'Z' || stateProv[1] < 'A' || stateProv[1] > 'Z' {
			t.Fatalf("expected two-letter StateProv at row %d, got %q", position+2, stateProv)
		}
		if row[3] == "" {
			t.Fatalf("expected non-empty region at row %d, got %q", position+2, row[3])
		}
		regions[row[3]] = struct{}{}
		population, err := strconv.ParseUint(row[4], 10, 64)
		if err != nil {
			t.Fatalf("expected population at row %d to be uint64, got %q", position+2, row[4])
		}
		if population == 0 {
			t.Fatalf("expected population at row %d to be positive, got %d", position+2, population)
		}
	}
	if len(regions) != 20 {
		t.Fatalf("expected 20 regions, got %d", len(regions))
	}

}

// TestItalyPostalCodes validates the frozen runtime postal-code sets.
func TestItalyPostalCodes(t *testing.T) {

	places := readItalyCSV(
		t,
		"places.csv",
		italyPlacesSHA256,
		[]string{"istat_code", "city", "state_prov", "region", "population"},
	)
	postalCodes := readItalyCSV(
		t,
		"postal_codes.csv",
		italyPostalCodesSHA256,
		[]string{"istat_code", "postal_codes"},
	)
	if len(postalCodes) != 7896 {
		t.Fatalf("expected 7896 postal-code rows, got %d", len(postalCodes))
	}
	if len(postalCodes) != len(places) {
		t.Fatalf("expected postal and place cardinality %d, got %d", len(places), len(postalCodes))
	}

	previousCode := ""
	for position, row := range postalCodes {
		code := row[0]
		if !isASCIIDigits(code, 6) {
			t.Fatalf("expected six-digit IstatCode at row %d, got %q", position+2, code)
		}
		if previousCode != "" && code <= previousCode {
			t.Fatalf("expected IstatCode after %q at row %d, got %q", previousCode, position+2, code)
		}
		previousCode = code
		if code != places[position][0] {
			t.Fatalf("expected postal IstatCode %q at row %d, got %q", places[position][0], position+2, code)
		}
		requireItalyCAPs(t, row[1], 8)
	}

}

// TestItalyPostalOverrides validates the approved build-time postal overrides.
func TestItalyPostalOverrides(t *testing.T) {

	places := readItalyCSV(
		t,
		"places.csv",
		italyPlacesSHA256,
		[]string{"istat_code", "city", "state_prov", "region", "population"},
	)
	placeCodes := make(map[string]struct{}, len(places))
	for _, row := range places {
		placeCodes[row[0]] = struct{}{}
	}
	overrides := readItalyCSV(
		t,
		filepath.Join("provenance", "postal_overrides.csv"),
		italyPostalOverridesSHA256,
		[]string{"istat_code", "postal_codes", "reason"},
	)
	if len(overrides) != 147 {
		t.Fatalf("expected 147 postal override rows, got %d", len(overrides))
	}

	previousCode := ""
	foundReggio := false
	for position, row := range overrides {
		code := row[0]
		if !isASCIIDigits(code, 6) {
			t.Fatalf("expected six-digit IstatCode at row %d, got %q", position+2, code)
		}
		if previousCode != "" && code <= previousCode {
			t.Fatalf("expected IstatCode after %q at row %d, got %q", previousCode, position+2, code)
		}
		previousCode = code
		if _, exists := placeCodes[code]; !exists {
			t.Fatalf("expected override IstatCode in places.csv, got %q", code)
		}
		caps := requireItalyCAPs(t, row[1], 0)
		if row[2] == "" {
			t.Fatalf("expected non-empty override reason at row %d, got %q", position+2, row[2])
		}
		if code == "080063" {
			foundReggio = true
			if len(caps) != 15 {
				t.Fatalf("expected Reggio di Calabria to retain 15 evidenced CAP, got %d", len(caps))
			}
		}
	}
	if !foundReggio {
		t.Fatal("expected Reggio di Calabria override, got none")
	}

}

// TestItalyReviewRecord validates the supplied PASS decisions and the reviewed
// artifact bytes.
func TestItalyReviewRecord(t *testing.T) {

	path := filepath.Join(italyDataDirectory, "provenance", "review-record.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected review record to be readable, got %v", err)
	}
	var record italyReviewRecord
	err = json.Unmarshal(data, &record)
	if err != nil {
		t.Fatalf("expected valid review record JSON, got %v", err)
	}
	if record.MarketDataVersion != "it-2026a" {
		t.Fatalf("expected market data version it-2026a, got %q", record.MarketDataVersion)
	}
	if record.D1AEmpiricalReview != "PASS" {
		t.Fatalf("expected empirical review PASS, got %q", record.D1AEmpiricalReview)
	}
	if record.PostalOverrideReview != "PASS" {
		t.Fatalf("expected postal override review PASS, got %q", record.PostalOverrideReview)
	}
	if record.PlausibilityReview != "PASS" {
		t.Fatalf("expected plausibility review PASS, got %q", record.PlausibilityReview)
	}
	if record.Reviewer != nil {
		t.Fatalf("expected reviewer null, got %q", *record.Reviewer)
	}
	if record.ReviewTimestamp != nil {
		t.Fatalf("expected review timestamp null, got %q", *record.ReviewTimestamp)
	}

	expected := map[string]string{
		"human-review.md":                 italyHumanReviewSHA256,
		"human-review-proposed-postal.md": italyProposedPostalReviewSHA256,
		"postal-override-review.md":       italyPostalOverrideReviewSHA256,
	}
	if len(record.ReviewedArtifacts) != len(expected) {
		t.Fatalf("expected %d reviewed artifacts, got %d", len(expected), len(record.ReviewedArtifacts))
	}
	for name, expectedSHA256 := range expected {
		artifact, exists := record.ReviewedArtifacts[name]
		if !exists {
			t.Fatalf("expected reviewed artifact %q, got none", name)
		}
		if artifact.SHA256 != expectedSHA256 {
			t.Fatalf("expected reviewed artifact %q SHA-256 %s, got %s", name, expectedSHA256, artifact.SHA256)
		}
		actualSHA256 := italyFileSHA256(t, filepath.Join(italyDataDirectory, "provenance", name))
		if actualSHA256 != expectedSHA256 {
			t.Fatalf("expected reviewed file %q SHA-256 %s, got %s", name, expectedSHA256, actualSHA256)
		}
	}

}

func isASCIIDigits(value string, width int) bool {
	if len(value) != width {
		return false
	}
	for position := range value {
		if value[position] < '0' || value[position] > '9' {
			return false
		}
	}
	return true
}

func italyFileSHA256(t *testing.T, path string) string {

	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %q to be readable, got %v", path, err)
	}
	digest := sha256.Sum256(data)

	return fmt.Sprintf("%x", digest)
}

func readItalyCSV(t *testing.T, name, expectedSHA256 string, expectedHeader []string) [][]string {

	t.Helper()
	path := filepath.Join(italyDataDirectory, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %q to be readable, got %v", path, err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if digest != expectedSHA256 {
		t.Fatalf("expected %q SHA-256 %s, got %s", path, expectedSHA256, digest)
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatalf("expected %q without BOM, got BOM", path)
	}
	if !bytes.HasSuffix(data, []byte{'\n'}) {
		t.Fatalf("expected %q with final LF, got different ending", path)
	}
	if bytes.Contains(data, []byte{'\r'}) {
		t.Fatalf("expected %q with LF line endings, got CR", path)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("expected %q to be valid CSV, got %v", path, err)
	}
	if len(records) == 0 {
		t.Fatalf("expected %q to contain a header, got no rows", path)
	}
	if !slices.Equal(records[0], expectedHeader) {
		t.Fatalf("expected %q header %v, got %v", path, expectedHeader, records[0])
	}

	return records[1:]
}

func requireItalyCAPs(t *testing.T, value string, maximum int) []string {

	t.Helper()
	if value == "" {
		t.Fatal("expected at least one CAP, got none")
	}
	caps := strings.Split(value, ";")
	if maximum > 0 && len(caps) > maximum {
		t.Fatalf("expected at most %d CAP, got %d in %q", maximum, len(caps), value)
	}
	if !slices.IsSorted(caps) {
		t.Fatalf("expected sorted CAP, got %q", value)
	}
	for position, cap := range caps {
		if !isASCIIDigits(cap, 5) {
			t.Fatalf("expected five-digit CAP at position %d, got %q", position+1, cap)
		}
		if strings.TrimSpace(cap) != cap {
			t.Fatalf("expected CAP without whitespace at position %d, got %q", position+1, cap)
		}
		if position > 0 && cap == caps[position-1] {
			t.Fatalf("expected unique CAP, got duplicate %q", cap)
		}
	}

	return caps
}
