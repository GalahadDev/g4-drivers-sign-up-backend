package vision

import (
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

	for _, desc := range labelDescriptions {
		for _, tag := range excludedTags {
			if strings.EqualFold(desc, tag) {
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
