package vision

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"google.golang.org/api/option"
)

type VisionService struct {
	client *vision.ImageAnnotatorClient
}

func NewVisionService() (*VisionService, error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create vision client: %v", err)
	}
	return &VisionService{client: client}, nil
}

func NewVisionServiceWithCredentials(credentialsPath string) (*VisionService, error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create vision client with credentials: %v", err)
	}
	return &VisionService{client: client}, nil
}

func (s *VisionService) Close() error {
	return s.client.Close()
}

func (s *VisionService) ValidateFormalWear(ctx context.Context, imageContent []byte) (bool, []string, error) {
	if len(imageContent) == 0 {
		return false, nil, fmt.Errorf("empty image content")
	}

	image, err := vision.NewImageFromReader(strings.NewReader(string(imageContent)))
	if err != nil {
		return false, nil, fmt.Errorf("failed to create image from reader: %v", err)
	}

	features := []*visionpb.Feature{
		{Type: visionpb.Feature_LABEL_DETECTION, MaxResults: 20},
		{Type: visionpb.Feature_FACE_DETECTION, MaxResults: 5},
	}

	requests := []*visionpb.AnnotateImageRequest{
		{
			Image:    image,
			Features: features,
		},
	}

	response, err := s.client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
		Requests: requests,
	})
	if err != nil {
		return false, nil, fmt.Errorf("vision API request failed: %v", err)
	}

	if len(response.Responses) == 0 {
		return false, nil, fmt.Errorf("no response from Vision API")
	}

	res := response.Responses[0]

	if res.Error != nil {
		return false, nil, fmt.Errorf("vision API returned error: %v", res.Error.Message)
	}

	// 1. LOGGING LABELS
	var labelDescriptions []string
	for _, label := range res.LabelAnnotations {
		labelDescriptions = append(labelDescriptions, label.Description)
	}
	slog.Info("Google Vision Check", "labels", labelDescriptions, "faces_count", len(res.FaceAnnotations))

	// 2. FACE DETECTION CHECK
	hasFace := false
	for _, face := range res.FaceAnnotations {
		if face.DetectionConfidence > 0.6 {
			hasFace = true
			break
		}
	}

	if !hasFace {
		slog.Info("Validation Failed: No face detected with high confidence")
		return false, labelDescriptions, nil
	}

	// 3. FORMAL WEAR LABEL CHECK
	isFormal := false
	requiredTags := []string{
		"suit", "tuxedo", "formal wear", "blazer", "white collar worker",
		"official", "businessperson", "tie", "necktie", "bow tie",
		"person", "man", "gentleman",
	}
	excludedTags := []string{"vehicle", "car", "motor vehicle"}

	// If the image is dominated by vehicle labels, it's likely the wrong upload
	// (a car photo instead of a formal headshot) — reject it.
	for _, desc := range labelDescriptions {
		for _, tag := range excludedTags {
			if strings.EqualFold(desc, tag) {
				slog.Info("Validation Failed: excluded (vehicle) tag detected", "tag", desc)
				return false, labelDescriptions, nil
			}
		}
	}

	for _, desc := range labelDescriptions {
		for _, tag := range requiredTags {
			if strings.EqualFold(desc, tag) {
				isFormal = true
				slog.Info("Validation Success: Found matching tag", "tag", desc)
				break
			}
		}
		if isFormal {
			break
		}
	}

	return isFormal, labelDescriptions, nil
}

// extractTextFromImage calls DOCUMENT_TEXT_DETECTION on a raw image and returns the full text.
func (s *VisionService) extractTextFromImage(ctx context.Context, content []byte) (string, error) {
	img, err := vision.NewImageFromReader(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("create image: %w", err)
	}
	resp, err := s.client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image:    img,
				Features: []*visionpb.Feature{{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION}},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("vision annotate image: %w", err)
	}
	if len(resp.Responses) == 0 {
		return "", fmt.Errorf("empty vision response")
	}
	r := resp.Responses[0]
	if r.Error != nil {
		return "", fmt.Errorf("vision error: %s", r.Error.Message)
	}
	if r.FullTextAnnotation == nil {
		return "", nil // no text found — valid result, not an error
	}
	return r.FullTextAnnotation.Text, nil
}

// extractTextFromPDF calls DOCUMENT_TEXT_DETECTION on an inline PDF (≤5 pages).
func (s *VisionService) extractTextFromPDF(ctx context.Context, content []byte) (string, error) {
	resp, err := s.client.BatchAnnotateFiles(ctx, &visionpb.BatchAnnotateFilesRequest{
		Requests: []*visionpb.AnnotateFileRequest{
			{
				InputConfig: &visionpb.InputConfig{
					Content:  content,
					MimeType: "application/pdf",
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION}},
				Pages:    []int32{1, 2, 3, 4, 5},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("vision annotate PDF: %w", err)
	}
	if len(resp.Responses) == 0 {
		return "", fmt.Errorf("empty PDF vision response")
	}
	if resp.Responses[0].Error != nil {
		return "", fmt.Errorf("vision PDF error: %s", resp.Responses[0].Error.Message)
	}
	var sb strings.Builder
	for _, pageResp := range resp.Responses[0].Responses {
		if pageResp.FullTextAnnotation != nil {
			sb.WriteString(pageResp.FullTextAnnotation.Text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// ValidateDocument extracts text from the document (image or PDF) and validates it
// based on docType. A validation failure is returned as a result with Valid=false,
// NOT as an error. Errors indicate Vision API failures only.
func (s *VisionService) ValidateDocument(ctx context.Context, req DocumentValidationRequest) (DocumentValidationResult, error) {
	var (
		text string
		err  error
	)
	if req.MimeType == "application/pdf" {
		text, err = s.extractTextFromPDF(ctx, req.Content)
	} else {
		text, err = s.extractTextFromImage(ctx, req.Content)
	}
	if err != nil {
		return DocumentValidationResult{}, fmt.Errorf("extract text: %w", err)
	}

	slog.Info("Document text extracted", "docType", req.DocType, "textLen", len(text))

	switch req.DocType {
	case "driverLicense":
		return validateDriverLicense(text, req.ExpectedName)
	case "tlcLicense":
		return validateTLCLicense(text, req.ExpectedName)
	case "carRegistration":
		return validateCarRegistration(text, req.ExpectedName)
	case "vehicleInspection":
		return validateVehicleInspection(text, req.ExpectedPlate)
	case "tlcDiamond":
		return validateTLCDiamond(text, req.ExpectedPlate)
	case "insuranceFiles":
		return validateInsurance(text)
	default:
		return DocumentValidationResult{}, fmt.Errorf("unknown docType %q", req.DocType)
	}
}
