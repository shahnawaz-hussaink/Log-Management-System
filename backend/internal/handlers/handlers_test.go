package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"office-file-sharing/backend/internal/db"
	"office-file-sharing/backend/internal/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestReplaceDocument_OptionalFile(t *testing.T) {
	db.InitDB()

	var alice models.User
	if err := db.DB.First(&alice, "email = ?", "alice@office.com").Error; err != nil {
		t.Fatalf("Could not find test user Alice: %v", err)
	}

	var bob models.User
	if err := db.DB.First(&bob, "email = ?", "bob@office.com").Error; err != nil {
		t.Fatalf("Could not find test user Bob: %v", err)
	}

	doc := models.Document{
		ID:             uuid.New(),
		Filename:       "test.pdf",
		FilePath:       "test.pdf",
		UploaderID:     alice.ID,
		CurrentOwnerID: alice.ID,
		Status:         models.StatusSentBack,
		Title:          "Test Document",
		UniqueNumber:   "DOC-TEST-12345",
		Category:       "Document",
		Description:    "Test description",
		Tags:           "test, mock",
	}
	if err := db.DB.Create(&doc).Error; err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}
	defer func() {
		db.DB.Where("document_id = ?", doc.ID).Delete(&models.WorkflowHistory{})
		db.DB.Delete(&doc)
	}()

	e := echo.New()
	
	form := url.Values{}
	form.Add("uploader_id", alice.ID.String())
	form.Add("target_owner_id", bob.ID.String())
	form.Add("remarks", "Testing resubmit without file change")
	
	req := httptest.NewRequest(http.MethodPut, "/api/documents/"+doc.ID.String()+"/replace", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(doc.ID.String())

	err := ReplaceDocument(c)
	if err != nil {
		t.Fatalf("ReplaceDocument returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", rec.Code, rec.Body.String())
	}

	var updatedDoc models.Document
	if err := db.DB.First(&updatedDoc, "id = ?", doc.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve updated doc: %v", err)
	}

	if updatedDoc.Status != models.StatusPendingApproval {
		t.Errorf("Expected status %v, got %v", models.StatusPendingApproval, updatedDoc.Status)
	}

	if updatedDoc.CurrentOwnerID != bob.ID {
		t.Errorf("Expected owner %v, got %v", bob.ID, updatedDoc.CurrentOwnerID)
	}

	var history models.WorkflowHistory
	if err := db.DB.Where("document_id = ? AND action = ?", doc.ID, models.ActionResubmitted).First(&history).Error; err != nil {
		t.Fatalf("Workflow history for resubmission not found: %v", err)
	}

	if history.Remarks != "Testing resubmit without file change" {
		t.Errorf("Expected remarks %q, got %q", "Testing resubmit without file change", history.Remarks)
	}
}
