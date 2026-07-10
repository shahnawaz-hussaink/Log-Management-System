package document

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
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
)

type Service interface {
	Upload(uploaderID, targetOwnerID uuid.UUID, title, description, category, tags string, fileHeader *multipart.FileHeader) (*DocumentResponse, error)
	List(userID uuid.UUID, search string) ([]DocumentResponse, error)
	GetDetails(docID, authenticatedUserID uuid.UUID) (*DocumentDetailsResponse, error)
	GetFilePathForDownload(docID, authenticatedUserID uuid.UUID) (string, error)
	Replace(docID, authenticatedUserID, targetOwnerID uuid.UUID, title, description, category, tags string, fileHeader *multipart.FileHeader, remarks string) (*DocumentResponse, error)
	TakeAction(docID, authenticatedUserID uuid.UUID, req ActionRequest) (*DocumentResponse, error)
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

func (s *service) Upload(uploaderID, targetOwnerID uuid.UUID, title, description, category, tags string, fileHeader *multipart.FileHeader) (*DocumentResponse, error) {
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

	docID := uuid.New()
	doc := &models.Document{
		ID:             docID,
		Filename:       fileHeader.Filename,
		FilePath:       destPath,
		UploaderID:     uploaderID,
		CurrentOwnerID: targetOwnerID,
		Status:         models.StatusPendingApproval,
		Title:          title,
		Description:    description,
		UniqueNumber:   uniqueNum,
		Tags:           tags,
		Category:       category,
	}

	if err := s.repo.Create(doc); err != nil {
		return nil, err
	}

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		DocumentID: docID,
		ActorID:    uploaderID,
		TargetID:   &targetOwnerID,
		Action:     models.ActionUploaded,
		Remarks:    "Document submitted for approval",
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

func (s *service) Replace(docID, authenticatedUserID, targetOwnerID uuid.UUID, title, description, category, tags string, fileHeader *multipart.FileHeader, remarks string) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if doc.UploaderID != authenticatedUserID {
		return nil, errors.New("only the original uploader is authorized to replace or resubmit this document")
	}

	if doc.Status != models.StatusSentBack {
		return nil, errors.New("document must be in 'Sent Back' status to be replaced or resubmitted")
	}

	fileReplaced := false
	if fileHeader != nil {
		src, err := fileHeader.Open()
		if err == nil {
			defer src.Close()
			uniquePrefix := fmt.Sprintf("%d_", time.Now().Unix())
			safeFilename := uniquePrefix + filepath.Base(fileHeader.Filename)
			destPath := filepath.Join(s.uploadsDir, safeFilename)

			dst, err := os.Create(destPath)
			if err == nil {
				defer dst.Close()
				if _, err = io.Copy(dst, src); err == nil {
					os.Remove(doc.FilePath)
					doc.Filename = fileHeader.Filename
					doc.FilePath = destPath
					fileReplaced = true
				}
			}
		}
	}

	if title != "" {
		doc.Title = title
	}
	if description != "" {
		doc.Description = description
	}
	if category != "" {
		doc.Category = category
	}
	if tags != "" {
		doc.Tags = tags
	}

	doc.Status = models.StatusPendingApproval
	doc.CurrentOwnerID = targetOwnerID

	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	wfAction := models.ActionResubmitted
	if fileReplaced {
		wfAction = models.ActionFileReplaced
	}

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		TargetID:   &targetOwnerID,
		Action:     wfAction,
		Remarks:    remarks,
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(doc.ID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) TakeAction(docID, authenticatedUserID uuid.UUID, req ActionRequest) (*DocumentResponse, error) {
	doc, err := s.repo.GetByID(docID)
	if err != nil {
		return nil, errors.New("document not found")
	}

	if doc.CurrentOwnerID != authenticatedUserID {
		return nil, errors.New("you are not authorized to act on this document as you are not the current owner")
	}

	var newStatus models.DocumentStatus
	var nextOwnerID uuid.UUID
	wfAction := models.WorkflowAction(req.Action)

	switch wfAction {
	case models.ActionApproved:
		newStatus = models.StatusApproved
		nextOwnerID = authenticatedUserID
	case models.ActionRejected:
		newStatus = models.StatusRejected
		nextOwnerID = doc.UploaderID
	case models.ActionSentBack:
		newStatus = models.StatusSentBack
		nextOwnerID = doc.UploaderID
	case models.ActionForwarded, "Forward":
		newStatus = models.StatusPendingApproval
		if req.TargetID == nil {
			return nil, errors.New("target ID is required to forward this document")
		}
		nextOwnerID = *req.TargetID
		wfAction = models.ActionForwarded
	default:
		return nil, errors.New("invalid action name")
	}

	doc.Status = newStatus
	doc.CurrentOwnerID = nextOwnerID

	if req.Signature != "" {
		existingSigs, _ := s.repo.CountSignatures(doc.ID)
		if err := stampSignatureOnPDF(doc.FilePath, req.Signature, existingSigs); err != nil {
			log.Printf("Error overlaying signature on PDF: %v", err)
		}
	}

	if err := s.repo.Save(doc); err != nil {
		return nil, err
	}

	history := &models.WorkflowHistory{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		ActorID:    authenticatedUserID,
		TargetID:   req.TargetID,
		Action:     wfAction,
		Remarks:    req.Remarks,
		Signature:  req.Signature,
	}
	_ = s.repo.CreateHistory(history)

	updatedDoc, _ := s.repo.GetByID(doc.ID)
	return s.toDocumentResponse(updatedDoc), nil
}

func (s *service) authorizeDocAccess(doc *models.Document, userID uuid.UUID) error {
	if doc.UploaderID == userID || doc.CurrentOwnerID == userID {
		return nil
	}

	histories, err := s.repo.GetHistoryByDocumentID(doc.ID)
	if err == nil {
		for _, h := range histories {
			if h.ActorID == userID {
				return nil
			}
		}
	}

	return errors.New("you are not authorized to view or access this document")
}

func (s *service) toDocumentResponse(d *models.Document) *DocumentResponse {
	return &DocumentResponse{
		ID:             d.ID,
		Filename:       d.Filename,
		FilePath:       d.FilePath,
		UploaderID:     d.UploaderID,
		CurrentOwnerID: d.CurrentOwnerID,
		Status:         d.Status,
		Title:          d.Title,
		Description:    d.Description,
		UniqueNumber:   d.UniqueNumber,
		Tags:           d.Tags,
		Category:       d.Category,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
		Uploader:       d.Uploader,
		CurrentOwner:   d.CurrentOwner,
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

func stampSignatureOnPDF(pdfPath string, base64Signature string, existingSigCount int) error {
	if base64Signature == "" {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(pdfPath), ".pdf") {
		return nil
	}
	parts := strings.Split(base64Signature, ",")
	base64Data := parts[len(parts)-1]

	dec, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return err
	}

	tempDir := os.TempDir()
	tempPNG := filepath.Join(tempDir, fmt.Sprintf("sig_temp_%d.png", time.Now().UnixNano()))
	err = os.WriteFile(tempPNG, dec, 0644)
	if err != nil {
		return err
	}
	defer os.Remove(tempPNG)

	tempOutPDF := pdfPath + ".signed"
	offsetX := -20 - (existingSigCount * 110)
	desc := fmt.Sprintf("scale:0.25, pos:br, off:%d 20", offsetX)
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
