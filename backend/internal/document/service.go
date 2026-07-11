package document

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"office-file-sharing/backend/internal/shared/models"

	"github.com/google/uuid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type Service interface {
	Upload(uploaderID, targetOwnerID uuid.UUID, title, description, category, tags, priority, direction string, fileHeader *multipart.FileHeader) (*DocumentResponse, error)
	List(userID uuid.UUID, search string) ([]DocumentResponse, error)
	GetDetails(docID, authenticatedUserID uuid.UUID) (*DocumentDetailsResponse, error)
	GetFilePathForDownload(docID, authenticatedUserID uuid.UUID) (string, error)
	Replace(docID, authenticatedUserID, targetOwnerID uuid.UUID, title, description, category, tags, priority, direction string, fileHeader *multipart.FileHeader, remarks string) (*DocumentResponse, error)
	TakeAction(docID, authenticatedUserID uuid.UUID, req ActionRequest) (*DocumentResponse, error)
	Recall(docID, authenticatedUserID uuid.UUID) (*DocumentResponse, error)
	AppendNote(docID, authenticatedUserID uuid.UUID, note string, actorIP string) (*DocumentResponse, error)
	SaveDraft(docID, authenticatedUserID uuid.UUID, draft string) (*DocumentResponse, error)
	AddAttachment(docID, authenticatedUserID uuid.UUID, fileHeader *multipart.FileHeader) (*AttachmentResponse, error)
	GetAttachmentFilePathForDownload(attID, authenticatedUserID uuid.UUID) (string, error)
	GetNotifications(recipientID uuid.UUID) ([]models.Notification, error)
	GetReports(schoolID uuid.UUID) (interface{}, error)
}

type service struct {
	repo       Repository
	uploadsDir string
}

func NewService(repo Repository, uploadsDir string) Service {
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		log.Printf("Warning: Failed to create uploads directory: %v", err)
	}
	return &service{repo: repo, uploadsDir: uploadsDir}
}

func (s *service) Upload(uploaderID, targetOwnerID uuid.UUID, title, description, category, tags, priority, direction string, fileHeader *multipart.FileHeader) (*DocumentResponse, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	uniquePrefix := fmt.Sprintf("%d_", time.Now().Unix())
	safeFilename := uniquePrefix + filepath.Base(fileHeader.Filename)
	destPath := filepath.Join(s.uploadsDir, safeFilename)

	dst, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, err
	}

	uniqueNum := fmt.Sprintf("DOC-%d", time.Now().UnixNano()/1e6)

	// Fetch DocumentType configuration based on category slug
	slug := strings.ToLower(strings.ReplaceAll(category, " ", "-"))
	var schoolID *uuid.UUID
	var uploaderUser models.User
	if err := s.repo.(*repository).db.First(&uploaderUser, "id = ?", uploaderID).Error; err == nil {
		schoolID = uploaderUser.SchoolID
	}

	var docTypeID *uuid.UUID
	var dt *models.DocumentType
	if schoolID != nil {
		dt, err = s.repo.GetDocumentTypeBySlug(*schoolID, slug)
		if err == nil {
			docTypeID = &dt.ID
		}
	}

	assignedOwnerID := targetOwnerID

	docID := uuid.New()
	doc := &models.Document{
		ID:             docID,
		SchoolID:       schoolID,
		DocumentTypeID: docTypeID,
		Filename:       fileHeader.Filename,
		FilePath:       destPath,
		UploaderID:     uploaderID,
		CurrentOwnerID: assignedOwnerID,
		Status:         models.StatusPendingApproval,
		Title:          title,
		Description:    description,
		UniqueNumber:   uniqueNum,
		Tags:           tags,
		Category:       category,
		Priority:       fallbackString(priority, "Normal"),
		Direction:      fallbackString(direction, "Inward"),
		AssignedAt:     time.Now(),
		Version:        1,
		CurrentStage:   1,
	}

	// Calculate SLA deadline if type is resolved
	if dt != nil {
		deadline := time.Now().Add(time.Duration(dt.SlaHours) * time.Hour)
		doc.SlaDeadline = &deadline
	}

	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}

	// Setup initial pending approver records
	pendingApprover := &models.DocumentPendingApprover{
		ID:         uuid.New(),
		DocumentID: docID,
		UserID:     assignedOwnerID,
		Stage:      1,
		Status:     "Pending",
	}
	if err := s.repo.CreatePendingApprover(pendingApprover); err != nil {
		log.Printf("Warning: Failed to create pending approver: %v", err)
	}

	// Queue notifications asynchronously in the DB
	notifPayload := fmt.Sprintf(`{"document_title": "%s", "uploader_name": "%s"}`, title, uploaderUser.Name)
	newNotification := &models.Notification{
		ID:          uuid.New(),
		SchoolID:    schoolID,
		RecipientID: assignedOwnerID,
		DocumentID:  &docID,
		Channel:     "email",
		Template:    "action_required",
		Payload:     notifPayload,
		Status:      "pending",
	}
	if err := s.repo.(*repository).db.Create(newNotification).Error; err != nil {
		log.Printf("Warning: Failed to queue notification: %v", err)
	}

	historyRemarks := "Document submitted for approval"

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   schoolID,
		DocumentID: docID,
		ActorID:    uploaderID,
		TargetID:   &assignedOwnerID,
		Action:     models.ActionUploaded,
		Remarks:    historyRemarks,
		ActorRole:  uploaderUser.Role,
		Stage:      1,
		Version:    1,
		EventType:  "state_transition",
	}

	if err := s.repo.CreateHistory(history); err != nil {
		log.Printf("Warning: Failed to write upload workflow log: %v", err)
	}

	savedDoc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, err
	}

	return s.toDocumentResponse(savedDoc), nil
}

func (s *service) List(userID uuid.UUID, search string) ([]DocumentResponse, error) {
	docs, err := s.repo.ListByUser(userID, search)
	if err != nil {
		return nil, err
	}

	responses := make([]DocumentResponse, len(docs))
	for i, d := range docs {
		responses[i] = *s.toDocumentResponse(&d)
	}
	return responses, nil
}

func (s *service) GetDetails(docID, authenticatedUserID uuid.UUID) (*DocumentDetailsResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if err := s.authorizeDocAccess(doc, authenticatedUserID); err != nil {
		return nil, err
	}

	histories, err := s.repo.GetHistoryByDocumentID(docID)
	if err != nil {
		return nil, err
	}

	docDto := s.toDocumentResponse(doc)
	historyDtos := make([]HistoryResponse, len(histories))
	for i, h := range histories {
		historyDtos[i] = *s.toHistoryResponse(&h)
	}

	return &DocumentDetailsResponse{
		Document: *docDto,
		History:  historyDtos,
	}, nil
}

func (s *service) GetFilePathForDownload(docID, authenticatedUserID uuid.UUID) (string, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return "", errors.New("document not found")
	}

	if err := s.authorizeDocAccess(doc, authenticatedUserID); err != nil {
		return "", err
	}

	return doc.FilePath, nil
}

func (s *service) Replace(docID, authenticatedUserID, targetOwnerID uuid.UUID, title, description, category, tags, priority, direction string, fileHeader *multipart.FileHeader, remarks string) (*DocumentResponse, error) {
	oldDoc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if oldDoc.UploaderID != authenticatedUserID {
		return nil, errors.New("only the original uploader is authorized to replace or resubmit this document")
	}

	if oldDoc.Status != models.StatusSentBack {
		return nil, errors.New("document must be in 'Sent Back' status to be replaced or resubmitted")
	}

	// Resolve target path (either new uploaded file or keep old reference path)
	destPath := oldDoc.FilePath
	filename := oldDoc.Filename
	fileReplaced := false

	if fileHeader != nil {
		src, err := fileHeader.Open()
		if err == nil {
			defer src.Close()
			uniquePrefix := fmt.Sprintf("%d_", time.Now().Unix())
			safeFilename := uniquePrefix + filepath.Base(fileHeader.Filename)
			destPath = filepath.Join(s.uploadsDir, safeFilename)

			dst, err := os.Create(destPath)
			if err == nil {
				defer dst.Close()
				if _, err = io.Copy(dst, src); err == nil {
					filename = fileHeader.Filename
					fileReplaced = true
				}
			}
		}
	}

	// Create NEW Document version record instead of mutating the old row
	newDocID := uuid.New()
	newVersion := oldDoc.Version + 1

	var user models.User
	s.repo.(*repository).db.First(&user, "id = ?", authenticatedUserID)

	var dt *models.DocumentType
	if oldDoc.DocumentTypeID != nil {
		dt, _ = s.repo.GetDocumentTypeByID(*oldDoc.DocumentTypeID)
	}

	newDoc := &models.Document{
		ID:             newDocID,
		SchoolID:       oldDoc.SchoolID,
		DocumentTypeID: oldDoc.DocumentTypeID,
		Filename:       filename,
		FilePath:       destPath,
		UploaderID:     oldDoc.UploaderID,
		CurrentOwnerID: targetOwnerID,
		Status:         models.StatusPendingApproval,
		Title:          fallbackString(title, oldDoc.Title),
		Description:    fallbackString(description, oldDoc.Description),
		UniqueNumber:   oldDoc.UniqueNumber,
		Tags:           fallbackString(tags, oldDoc.Tags),
		Category:       fallbackString(category, oldDoc.Category),
		Priority:       fallbackString(priority, oldDoc.Priority),
		Direction:      fallbackString(direction, oldDoc.Direction),
		AssignedAt:     time.Now(),
		Version:        newVersion,
		ParentDocID:    &oldDoc.ID,
		CurrentStage:   1,
	}

	if dt != nil {
		deadline := time.Now().Add(time.Duration(dt.SlaHours) * time.Hour)
		newDoc.SlaDeadline = &deadline
	}

	if err := s.repo.Create(newDoc); err != nil {
		return nil, err
	}

	// Setup pending approver
	pendingApprover := &models.DocumentPendingApprover{
		ID:         uuid.New(),
		DocumentID: newDocID,
		UserID:     targetOwnerID,
		Stage:      1,
		Status:     "Pending",
	}
	s.repo.CreatePendingApprover(pendingApprover)

	// Update old doc status to reflect it's been superseded
	oldDoc.Status = models.StatusDraft // Archive old version out of active approval list
	s.repo.Save(oldDoc)

	wfAction := models.ActionResubmitted
	if fileReplaced {
		wfAction = models.ActionFileReplaced
	}

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   newDoc.SchoolID,
		DocumentID: newDocID,
		ActorID:    authenticatedUserID,
		TargetID:   &targetOwnerID,
		Action:     wfAction,
		Remarks:    remarks,
		ActorRole:  user.Role,
		Stage:      1,
		Version:    newVersion,
		EventType:  "state_transition",
	}
	_ = s.repo.CreateHistory(history)

	// Keep version chain linked in history
	oldHistory := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   oldDoc.SchoolID,
		DocumentID: oldDoc.ID,
		ActorID:    authenticatedUserID,
		TargetID:   &targetOwnerID,
		Action:     wfAction,
		Remarks:    fmt.Sprintf("Superseded by Version %d", newVersion),
		ActorRole:  user.Role,
		Stage:      oldDoc.CurrentStage,
		Version:    oldDoc.Version,
		EventType:  "state_transition",
	}
	_ = s.repo.CreateHistory(oldHistory)

	updatedDoc, _ := s.repo.GetByID(newDocID)
	return s.toDocumentResponse(updatedDoc), nil
}

func fallbackString(val, backup string) string {
	if val == "" {
		return backup
	}
	return val
}

func (s *service) TakeAction(docID, authenticatedUserID uuid.UUID, req ActionRequest) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	// Verify authorization: check current owner or stage-pending approvers
	authorized := doc.CurrentOwnerID == authenticatedUserID
	if !authorized {
		approvers, _ := s.repo.GetPendingApprovers(doc.ID, doc.CurrentStage)
		for _, a := range approvers {
			if a.UserID == authenticatedUserID && a.Status == "Pending" {
				authorized = true
				break
			}
		}
	}
	if !authorized {
		return nil, errors.New("you are not authorized to act on this document at its current stage")
	}

	var actorUser models.User
	s.repo.(*repository).db.First(&actorUser, "id = ?", authenticatedUserID)

	wfAction := models.WorkflowAction(req.Action)
	var newStatus models.DocumentStatus
	var nextOwnerID uuid.UUID

	switch wfAction {
	case models.ActionApproved:
		newStatus = models.StatusApproved
		nextOwnerID = authenticatedUserID

		// Mark current approver as done
		s.repo.MarkApproverStatus(doc.ID, authenticatedUserID, doc.CurrentStage, "Approved")

		// If a DocumentType and stages are defined, resolve if we need to advance stages
		if doc.DocumentTypeID != nil {
			dt, errDT := s.repo.GetDocumentTypeByID(*doc.DocumentTypeID)
			if errDT == nil {
				if req.TargetID != nil && *req.TargetID != authenticatedUserID {
					newStatus = models.StatusPendingApproval
					nextOwnerID = *req.TargetID
					doc.CurrentStage = doc.CurrentStage + 1

					// Register next stage pending approver
					nextApprover := &models.DocumentPendingApprover{
						ID:         uuid.New(),
						DocumentID: doc.ID,
						UserID:     *req.TargetID,
						Stage:      doc.CurrentStage,
						Status:     "Pending",
					}
					s.repo.CreatePendingApprover(nextApprover)

					// Set new SLA deadline
					deadline := time.Now().Add(time.Duration(dt.SlaHours) * time.Hour)
					doc.SlaDeadline = &deadline
					wfAction = models.ActionApproved // keep approved as action name
				}
			}
		}

	case models.ActionRejected:
		if strings.TrimSpace(req.Remarks) == "" {
			return nil, errors.New("rejection remarks/reason is required")
		}
		newStatus = models.StatusRejected
		nextOwnerID = doc.UploaderID
		s.repo.MarkApproverStatus(doc.ID, authenticatedUserID, doc.CurrentStage, "Rejected")

	case models.ActionSentBack:
		if strings.TrimSpace(req.Remarks) == "" {
			return nil, errors.New("remarks are required to send the document back for revision")
		}
		newStatus = models.StatusSentBack
		nextOwnerID = doc.UploaderID
		s.repo.MarkApproverStatus(doc.ID, authenticatedUserID, doc.CurrentStage, "Sent Back")

	case models.ActionForwarded, "Forward":
		if req.TargetID == nil {
			return nil, errors.New("target ID is required to forward this document")
		}
		newStatus = models.StatusPendingApproval
		nextOwnerID = *req.TargetID
		wfAction = models.ActionForwarded

		// Transition pending stage approver
		s.repo.MarkApproverStatus(doc.ID, authenticatedUserID, doc.CurrentStage, "Forwarded")
		doc.CurrentStage = doc.CurrentStage + 1

		nextApprover := &models.DocumentPendingApprover{
			ID:         uuid.New(),
			DocumentID: doc.ID,
			UserID:     *req.TargetID,
			Stage:      doc.CurrentStage,
			Status:     "Pending",
		}
		s.repo.CreatePendingApprover(nextApprover)

		if doc.DocumentTypeID != nil {
			if dt, errDT := s.repo.GetDocumentTypeByID(*doc.DocumentTypeID); errDT == nil {
				deadline := time.Now().Add(time.Duration(dt.SlaHours) * time.Hour)
				doc.SlaDeadline = &deadline
			}
		}

	case "Refer":
		if req.TargetID == nil {
			return nil, errors.New("target ID is required to refer this document for opinion")
		}
		newStatus = models.StatusPendingApproval
		nextOwnerID = *req.TargetID
		wfAction = "Referred"

		// Save current owner in referral placeholder
		originalOwner := doc.CurrentOwnerID
		doc.ReferralOwnerID = &originalOwner

	case "Return":
		if doc.ReferralOwnerID == nil {
			return nil, errors.New("this document has not been referred, cannot perform Return action")
		}
		newStatus = models.StatusPendingApproval
		nextOwnerID = *doc.ReferralOwnerID
		doc.ReferralOwnerID = nil
		wfAction = "Returned"

	case "Close":
		newStatus = models.StatusClosed
		nextOwnerID = doc.UploaderID
		wfAction = "Closed"

	case "Archive":
		newStatus = models.StatusArchived
		nextOwnerID = doc.UploaderID
		wfAction = "Archived"

	default:
		return nil, errors.New("invalid action name")
	}

	doc.Status = newStatus
	if doc.CurrentOwnerID != nextOwnerID {
		doc.CurrentOwnerID = nextOwnerID
		doc.AssignedAt = time.Now()
	}

	var token string
	if wfAction != models.ActionSentBack && wfAction != "Sent Back" {
		hash := sha256.New()
		hash.Write([]byte(fmt.Sprintf("%s-%s-%s-%d", doc.ID, authenticatedUserID, wfAction, time.Now().UnixNano())))
		token = strings.ToUpper(hex.EncodeToString(hash.Sum(nil))[:12])

		existingSigs, _ := s.repo.CountSignatures(doc.ID)
		filePathLower := strings.ToLower(doc.FilePath)
		if strings.HasSuffix(filePathLower, ".pdf") {
			if err := stampTextSignatureOnPDF(doc.FilePath, actorUser.Name, token, req.Remarks, existingSigs); err != nil {
				log.Printf("Error overlaying text signature on PDF: %v", err)
			}
		}
	}

	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	// Queue notification to the next reviewer or uploader on action taken
	targetNotifRecipient := doc.CurrentOwnerID
	template := "action_required"
	if doc.Status == models.StatusApproved || doc.Status == models.StatusRejected || doc.Status == models.StatusSentBack {
		targetNotifRecipient = doc.UploaderID
		template = string(doc.Status)
	}

	notifPayload := fmt.Sprintf(`{"document_title": "%s", "actor_name": "%s", "action": "%s"}`, doc.Title, actorUser.Name, req.Action)
	newNotification := &models.Notification{
		ID:          uuid.New(),
		SchoolID:    doc.SchoolID,
		RecipientID: targetNotifRecipient,
		DocumentID:  &doc.ID,
		Channel:     "email",
		Template:    strings.ToLower(strings.ReplaceAll(template, " ", "_")),
		Payload:     notifPayload,
		Status:      "pending",
	}
	_ = s.repo.(*repository).db.Create(newNotification).Error

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   doc.SchoolID,
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		TargetID:   req.TargetID,
		Action:     wfAction,
		Remarks:    req.Remarks,
		Signature:  token,
		ActorRole:  actorUser.Role,
		Stage:      doc.CurrentStage,
		Version:    doc.Version,
		EventType:  "state_transition",
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(doc.ID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) authorizeDocAccess(doc *models.Document, userID uuid.UUID) error {
	var user models.User
	if err := s.repo.(*repository).db.First(&user, "id = ?", userID).Error; err != nil {
		return errors.New("user not found")
	}

	// Principal has school-wide access
	if user.Role == "Principal" {
		if doc.SchoolID != nil && user.SchoolID != nil && *doc.SchoolID == *user.SchoolID {
			return nil
		}
		return errors.New("you are not authorized to view this document (outside school scope)")
	}

	// Owner or uploader has direct access
	if doc.UploaderID == userID || doc.CurrentOwnerID == userID {
		return nil
	}

	// Parent has access if document belongs to child
	if user.Role == "Parent" {
		var count int64
		s.repo.(*repository).db.Model(&models.ParentChild{}).
			Where("parent_id = ? AND child_id = ?", userID, doc.UploaderID).
			Count(&count)
		if count > 0 {
			return nil
		}
	}

	// Teacher has access to class submissions or history
	if user.Role == "Teacher" {
		// 1. Check if uploader is a Student in Teacher's ClassSection
		var uploaderUser models.User
		if err := s.repo.(*repository).db.First(&uploaderUser, "id = ?", doc.UploaderID).Error; err == nil {
			if uploaderUser.Role == "Student" && uploaderUser.ClassSection != "" && uploaderUser.ClassSection == user.ClassSection {
				return nil
			}
		}

		// 2. Check workflow histories
		histories, err := s.repo.GetHistoryByDocumentID(doc.ID)
		if err == nil {
			for _, h := range histories {
				if h.ActorID == userID {
					return nil
				}
			}
		}
	}

	// Verify stage pending approvers
	approvers, _ := s.repo.GetPendingApprovers(doc.ID, doc.CurrentStage)
	for _, a := range approvers {
		if a.UserID == userID {
			return nil
		}
	}

	return errors.New("you are not authorized to view or access this document")
}

func (s *service) toDocumentResponse(d *models.Document) *DocumentResponse {
	attachments := make([]AttachmentResponse, len(d.Attachments))
	for i, att := range d.Attachments {
		attachments[i] = AttachmentResponse{
			ID:         att.ID,
			DocumentID: att.DocumentID,
			Filename:   att.Filename,
			UploadedBy: att.UploadedBy,
			CreatedAt:  att.CreatedAt,
		}
	}

	return &DocumentResponse{
		ID:              d.ID,
		Filename:        d.Filename,
		FilePath:        d.FilePath,
		UploaderID:      d.UploaderID,
		CurrentOwnerID:  d.CurrentOwnerID,
		Status:          d.Status,
		Title:           d.Title,
		Description:     d.Description,
		UniqueNumber:    d.UniqueNumber,
		Tags:            d.Tags,
		Category:        d.Category,
		Priority:        d.Priority,
		Direction:       d.Direction,
		AssignedAt:      d.AssignedAt,
		ReferralOwnerID: d.ReferralOwnerID,
		NotingSheet:     d.NotingSheet,
		DraftSpace:      d.DraftSpace,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		Uploader:        d.Uploader,
		CurrentOwner:    d.CurrentOwner,
		Attachments:     attachments,
	}
}

func (s *service) toHistoryResponse(h *models.WorkflowHistory) *HistoryResponse {
	return &HistoryResponse{
		ID:         h.ID,
		DocumentID: h.DocumentID,
		ActorID:    h.ActorID,
		TargetID:   h.TargetID,
		Action:     h.Action,
		Remarks:    h.Remarks,
		Signature:  h.Signature,
		CreatedAt:  h.CreatedAt,
		Actor:      h.Actor,
		Target:     h.Target,
	}
}

func (s *service) Recall(docID, authenticatedUserID uuid.UUID) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if doc.UploaderID != authenticatedUserID {
		return nil, errors.New("only the original uploader is authorized to recall this document")
	}

	if doc.Status != models.StatusPendingApproval {
		return nil, errors.New("only documents in 'Pending Approval' status can be recalled")
	}

	// Update status to Sent Back (allows easy resubmission UI flow)
	doc.Status = models.StatusSentBack
	doc.CurrentOwnerID = doc.UploaderID
	doc.AssignedAt = time.Now()

	// Save updates
	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	// Update pending approver stage statuses as skipped
	s.repo.(*repository).db.Model(&models.DocumentPendingApprover{}).
		Where("document_id = ? AND stage = ? AND status = 'Pending'", doc.ID, doc.CurrentStage).
		Update("status", "Skipped")

	var actorUser models.User
	s.repo.(*repository).db.First(&actorUser, "id = ?", authenticatedUserID)

	// Write recall action to workflow history log
	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   doc.SchoolID,
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		Action:     "Recalled",
		Remarks:    "File recalled back to draft/revision stage by uploader",
		ActorRole:  actorUser.Role,
		Stage:      doc.CurrentStage,
		Version:    doc.Version,
		EventType:  "state_transition",
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(docID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) AppendNote(docID, authenticatedUserID uuid.UUID, note string, actorIP string) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if err := s.authorizeDocAccess(doc, authenticatedUserID); err != nil {
		return nil, err
	}

	var actorUser models.User
	if err := s.repo.(*repository).db.First(&actorUser, "id = ?", authenticatedUserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	timestampStr := time.Now().Format("2006-01-02 15:04:05 MST")
	entry := fmt.Sprintf("[%s] - %s (%s) IP: %s\n%s\n\n", timestampStr, actorUser.Name, actorUser.Role, actorIP, note)
	doc.NotingSheet = doc.NotingSheet + entry

	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	// Record general note addition to history
	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   doc.SchoolID,
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		Action:     "Note Added",
		Remarks:    note,
		ActorRole:  actorUser.Role,
		Stage:      doc.CurrentStage,
		Version:    doc.Version,
		ActorIP:    actorIP,
		EventType:  "note_added",
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(docID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) SaveDraft(docID, authenticatedUserID uuid.UUID, draft string) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	// Only original uploader or current owner can edit the draft letters/orders space
	if doc.UploaderID != authenticatedUserID && doc.CurrentOwnerID != authenticatedUserID {
		return nil, errors.New("not authorized to edit drafts for this document")
	}

	doc.DraftSpace = draft
	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	var actorUser models.User
	s.repo.(*repository).db.First(&actorUser, "id = ?", authenticatedUserID)

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		SchoolID:   doc.SchoolID,
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		Action:     "Draft Updated",
		Remarks:    "Updated draft letter/order template",
		ActorRole:  actorUser.Role,
		Stage:      doc.CurrentStage,
		Version:    doc.Version,
		EventType:  "draft_updated",
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(docID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) AddAttachment(docID, authenticatedUserID uuid.UUID, fileHeader *multipart.FileHeader) (*AttachmentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if err := s.authorizeDocAccess(doc, authenticatedUserID); err != nil {
		return nil, err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	uniquePrefix := fmt.Sprintf("%d_att_", time.Now().Unix())
	safeFilename := uniquePrefix + filepath.Base(fileHeader.Filename)
	destPath := filepath.Join(s.uploadsDir, safeFilename)

	dst, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, err
	}

	att := &models.Attachment{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		Filename:   fileHeader.Filename,
		FilePath:   destPath,
		UploadedBy: authenticatedUserID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.(*repository).db.Create(att).Error; err != nil {
		return nil, err
	}

	return &AttachmentResponse{
		ID:         att.ID,
		DocumentID: att.DocumentID,
		Filename:   att.Filename,
		UploadedBy: att.UploadedBy,
		CreatedAt:  att.CreatedAt,
	}, nil
}

func (s *service) GetAttachmentFilePathForDownload(attID, authenticatedUserID uuid.UUID) (string, error) {
	var att models.Attachment
	if err := s.repo.(*repository).db.First(&att, "id = ?", attID).Error; err != nil {
		return "", errors.New("attachment not found")
	}

	doc, err := s.repo.GetByID(att.DocumentID)
	if err != nil {
		return "", errors.New("parent document not found")
	}

	if err := s.authorizeDocAccess(doc, authenticatedUserID); err != nil {
		return "", err
	}

	return att.FilePath, nil
}

func (s *service) GetNotifications(recipientID uuid.UUID) ([]models.Notification, error) {
	var list []models.Notification
	err := s.repo.(*repository).db.Where("recipient_id = ?", recipientID).Order("created_at desc").Limit(20).Find(&list).Error
	if err != nil {
		return nil, err
	}

	// Automatically mark pending notifications as read/sent
	now := time.Now()
	s.repo.(*repository).db.Model(&models.Notification{}).
		Where("recipient_id = ? AND status = 'pending'", recipientID).
		Updates(map[string]interface{}{"status": "sent", "sent_at": &now})

	return list, nil
}

func (s *service) GetReports(schoolID uuid.UUID) (interface{}, error) {
	var docs []models.Document
	err := s.repo.(*repository).db.Preload("Uploader").Preload("CurrentOwner").Where("school_id = ?", schoolID).Find(&docs).Error
	if err != nil {
		return nil, err
	}

	var histories []models.WorkflowHistory
	err = s.repo.(*repository).db.Preload("Actor").Where("school_id = ?", schoolID).Order("created_at asc").Find(&histories).Error
	if err != nil {
		return nil, err
	}

	// 1. Avg turnaround time calculations
	docHistories := make(map[uuid.UUID][]models.WorkflowHistory)
	for _, h := range histories {
		docHistories[h.DocumentID] = append(docHistories[h.DocumentID], h)
	}

	var totalDuration time.Duration
	var count int
	for _, logs := range docHistories {
		if len(logs) > 1 {
			for i := 0; i < len(logs)-1; i++ {
				diff := logs[i+1].CreatedAt.Sub(logs[i].CreatedAt)
				totalDuration += diff
				count++
			}
		}
	}

	avgTurnaroundHours := 0.0
	if count > 0 {
		avgTurnaroundHours = totalDuration.Hours() / float64(count)
	}

	// 2. Count statuses and SLA breaches
	totalActiveFiles := 0
	totalApprovedFiles := 0
	slaBreaches := 0
	now := time.Now()

	for _, doc := range docs {
		if doc.Status == models.StatusPendingApproval || doc.Status == models.StatusDraft || doc.Status == models.StatusSentBack {
			totalActiveFiles++
		}
		if doc.Status == models.StatusApproved {
			totalApprovedFiles++
		}
		if (doc.Status == models.StatusPendingApproval || doc.Status == models.StatusSentBack) && doc.SlaDeadline != nil && doc.SlaDeadline.Before(now) {
			slaBreaches++
		}
	}

	// 3. User Pendency Breakdown with usernames, roles, and count
	type UserKey struct {
		Name string
		Role string
	}
	pendencyMap := make(map[UserKey]int)
	for _, doc := range docs {
		if doc.Status == models.StatusPendingApproval || doc.Status == models.StatusSentBack {
			ownerName := "Unknown"
			ownerRole := "Staff"
			if doc.CurrentOwnerID != uuid.Nil {
				ownerName = doc.CurrentOwner.Name
				ownerRole = doc.CurrentOwner.Role
			}
			pendencyMap[UserKey{Name: ownerName, Role: ownerRole}]++
		}
	}

	type UserPendency struct {
		Username     string `json:"username"`
		Role         string `json:"role"`
		PendingCount int    `json:"pending_count"`
	}
	userPendencies := []UserPendency{}
	for k, v := range pendencyMap {
		userPendencies = append(userPendencies, UserPendency{
			Username:     k.Name,
			Role:         k.Role,
			PendingCount: v,
		})
	}

	// 4. Category workloads
	categoryMap := make(map[string]int)
	for _, doc := range docs {
		if doc.Category != "" {
			categoryMap[doc.Category]++
		}
	}

	type CategoryWorkload struct {
		Category string `json:"category"`
		Count    int    `json:"count"`
	}
	categoryWorkloads := []CategoryWorkload{}
	for k, v := range categoryMap {
		categoryWorkloads = append(categoryWorkloads, CategoryWorkload{
			Category: k,
			Count:    v,
		})
	}

	// 5. Movement logs list
	type MovementLog struct {
		DocumentTitle string    `json:"document_title"`
		ActorName     string    `json:"actor_name"`
		Action        string    `json:"action"`
		Remarks       string    `json:"remarks"`
		Timestamp     time.Time `json:"timestamp"`
	}
	movements := []MovementLog{}
	for _, h := range histories {
		actorName := "System"
		if h.Actor.Name != "" {
			actorName = h.Actor.Name
		}
		movements = append(movements, MovementLog{
			DocumentTitle: h.Remarks, // Using remarks or description
			ActorName:     actorName,
			Action:        string(h.Action),
			Remarks:       h.Remarks,
			Timestamp:     h.CreatedAt,
		})
	}

	return map[string]interface{}{
		"total_active_files":   totalActiveFiles,
		"total_approved_files": totalApprovedFiles,
		"avg_turnaround_hours": avgTurnaroundHours,
		"sla_breaches":         slaBreaches,
		"user_pendency":        userPendencies,
		"category_workloads":   categoryWorkloads,
		"movements":            movements,
		"total_count":          len(docs),
	}, nil
}

func stampTextSignatureOnPDF(pdfPath string, actorName string, token string, remarks string, existingSigCount int) error {
	if !strings.HasSuffix(strings.ToLower(pdfPath), ".pdf") {
		return nil
	}

	tempPNG, err := generateTransparentSignaturePNG(actorName, token, remarks)
	if err != nil {
		return err
	}
	defer os.Remove(tempPNG)

	tempOutPDF := pdfPath + ".signed"
	offsetY := 20 + (existingSigCount * 65)
	desc := fmt.Sprintf("scale:0.5 abs, pos:br, off:-20 %d, rot:0", offsetY)

	wm, err := pdfcpu.ParseImageWatermarkDetails(tempPNG, desc, true, types.POINTS)
	if err != nil {
		return err
	}

	err = api.AddWatermarksFile(pdfPath, tempOutPDF, nil, wm, nil)
	if err != nil {
		return err
	}

	err = os.Rename(tempOutPDF, pdfPath)
	if err != nil {
		os.Remove(tempOutPDF)
		return err
	}

	return nil
}

func generateTransparentSignaturePNG(actorName, token, remarks string) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, 380, 110))

	green := color.RGBA{34, 197, 94, 255}
	drawLine(img, 15, 60, 25, 75, green, 3)
	drawLine(img, 25, 75, 42, 45, green, 3)

	textColor := color.RGBA{15, 23, 42, 255}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: basicfont.Face7x13,
	}

	d.Dot = fixed.P(55, 25)
	d.DrawString("Signature Valid")
	d.Dot = fixed.P(56, 25)
	d.DrawString("Signature Valid")

	dateStr := time.Now().Format("2006.01.02 15:04:05 MST")
	cleanRemarks := remarks
	if len(cleanRemarks) > 40 {
		cleanRemarks = cleanRemarks[:37] + "..."
	}
	if cleanRemarks == "" {
		cleanRemarks = "Approved"
	}

	d.Dot = fixed.P(55, 45)
	d.DrawString("Digitally signed by " + actorName)

	d.Dot = fixed.P(55, 63)
	d.DrawString("Date: " + dateStr)

	d.Dot = fixed.P(55, 81)
	d.DrawString("Reason: " + cleanRemarks)

	d.Dot = fixed.P(55, 99)
	d.DrawString("Token: " + token)

	tempDir := os.TempDir()
	tempPNGPath := filepath.Join(tempDir, fmt.Sprintf("sig_image_%d.png", time.Now().UnixNano()))
	f, err := os.Create(tempPNGPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		os.Remove(tempPNGPath)
		return "", err
	}

	return tempPNGPath, nil
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color, thickness int) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	steps := math.Abs(dx)
	if math.Abs(dy) > steps {
		steps = math.Abs(dy)
	}
	if steps == 0 {
		return
	}
	xInc := dx / steps
	yInc := dy / steps
	x := float64(x1)
	y := float64(y1)
	for i := 0; i <= int(steps); i++ {
		for tx := -thickness/2; tx <= thickness/2; tx++ {
			for ty := -thickness/2; ty <= thickness/2; ty++ {
				img.Set(int(x)+tx, int(y)+ty, col)
			}
		}
		x += xInc
		y += yInc
	}
}

func stampSignatureOnDocx(docxPath string, base64Signature string, existingSigCount int) error {
	if base64Signature == "" {
		return nil
	}

	parts := strings.Split(base64Signature, ",")
	base64Data := parts[len(parts)-1]

	dec, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(docxPath)
	if err != nil {
		return err
	}
	defer r.Close()

	tempOutDocx := docxPath + ".signed"
	out, err := os.Create(tempOutDocx)
	if err != nil {
		return err
	}
	defer out.Close()

	w := zip.NewWriter(out)
	defer w.Close()

	sigID := fmt.Sprintf("rIdSig%d", existingSigCount+1)
	sigFileName := fmt.Sprintf("sig%d.png", existingSigCount+1)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		var content []byte
		if f.Name == "word/document.xml" || f.Name == "word/_rels/document.xml.rels" || f.Name == "[Content_Types].xml" {
			content, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return err
			}
		}

		if f.Name == "word/document.xml" {
			idx := bytes.LastIndex(content, []byte("</w:body>"))
			if idx == -1 {
				return fmt.Errorf("could not find closing w:body tag in word/document.xml")
			}

			cx := 1828800
			cy := 731520
			xmlInsert := fmt.Sprintf(`
<w:p xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:pPr>
    <w:jc w:val="right"/>
  </w:pPr>
  <w:r>
    <w:rPr>
      <w:sz w:val="20"/>
      <w:b/>
    </w:rPr>
    <w:t>Signed electronically:</w:t>
    <w:br/>
  </w:r>
  <w:r>
    <w:drawing xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
      <wp:inline distT="0" distB="0" distL="0" distR="0">
        <wp:extent cx="%d" cy="%d"/>
        <wp:effectExtent l="0" t="0" r="0" b="0"/>
        <wp:docPr id="1000" name="Signature"/>
        <wp:cNvGraphicFramePr>
          <a:graphicFrameLocks noChangeAspect="1"/>
        </wp:cNvGraphicFramePr>
        <a:graphic>
          <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">
            <pic:pic>
              <pic:nvPicPr>
                <pic:cNvPr id="1000" name="Signature"/>
                <pic:cNvPicPr/>
              </pic:nvPicPr>
              <pic:blipFill>
                <a:blip r:embed="%s"/>
                <a:stretch>
                  <a:fillRect/>
                </a:stretch>
              </pic:blipFill>
              <pic:spPr>
                <a:xfrm>
                  <a:off x="0" y="0"/>
                  <a:ext cx="%d" cy="%d"/>
                </a:xfrm>
                <a:prstGeom prst="rect">
                  <a:avLst/>
                </a:prstGeom>
              </pic:spPr>
            </pic:pic>
          </a:graphicData>
        </a:graphic>
      </wp:inline>
    </w:drawing>
  </w:r>
</w:p>`, cx, cy, sigID, cx, cy)

			newContent := append(content[:idx], []byte(xmlInsert)...)
			newContent = append(newContent, content[idx:]...)

			fw, err := w.Create(f.Name)
			if err != nil {
				return err
			}
			_, err = fw.Write(newContent)
			if err != nil {
				return err
			}

		} else if f.Name == "word/_rels/document.xml.rels" {
			idx := bytes.LastIndex(content, []byte("</Relationships>"))
			if idx == -1 {
				return fmt.Errorf("could not find closing Relationships tag in word/_rels/document.xml.rels")
			}

			relInsert := fmt.Sprintf(`  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>
`, sigID, sigFileName)

			newContent := append(content[:idx], []byte(relInsert)...)
			newContent = append(newContent, content[idx:]...)

			fw, err := w.Create(f.Name)
			if err != nil {
				return err
			}
			_, err = fw.Write(newContent)
			if err != nil {
				return err
			}

		} else if f.Name == "[Content_Types].xml" {
			var newContent []byte
			if !bytes.Contains(content, []byte(`Extension="png"`)) {
				idx := bytes.Index(content, []byte("<Types"))
				if idx == -1 {
					return fmt.Errorf("could not find Types tag in [Content_Types].xml")
				}
				closeIdx := bytes.Index(content[idx:], []byte(">"))
				if closeIdx == -1 {
					return fmt.Errorf("could not find end of Types tag in [Content_Types].xml")
				}
				insertPos := idx + closeIdx + 1
				decl := []byte(`
  <Default Extension="png" ContentType="image/png"/>`)
				newContent = append(content[:insertPos], decl...)
				newContent = append(newContent, content[insertPos:]...)
			} else {
				newContent = content
			}

			fw, err := w.Create(f.Name)
			if err != nil {
				return err
			}
			_, err = fw.Write(newContent)
			if err != nil {
				return err
			}

		} else {
			fw, err := w.CreateHeader(&f.FileHeader)
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(fw, rc)
			rc.Close()
			if err != nil {
				return err
			}
		}
	}

	mediaPath := fmt.Sprintf("word/media/%s", sigFileName)
	fw, err := w.Create(mediaPath)
	if err != nil {
		return err
	}
	_, err = fw.Write(dec)
	if err != nil {
		return err
	}

	w.Close()
	out.Close()
	r.Close()

	err = os.Rename(tempOutDocx, docxPath)
	if err != nil {
		os.Remove(tempOutDocx)
		return err
	}

	return nil
}
