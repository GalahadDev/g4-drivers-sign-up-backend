package domains

import (
	"time"

	"github.com/google/uuid"
)

type DriverApplication struct {
	ID                   uuid.UUID              `json:"id"`
	UserID               uuid.UUID              `json:"user_id"`
	FullName             string                 `json:"full_name"`
	Address              string                 `json:"address"`
	PhoneNumber          string                 `json:"phone_number"`
	EmergencyNumber      string                 `json:"emergency_number"`
	DeviceType           string                 `json:"device_type"`     // "phone" | "tablet"
	DriverCategory       string                 `json:"driver_category"` // 'comfort' o 'luxury'
	VehicleType          string                 `json:"vehicle_type"`
	PassengerCapacity    int                    `json:"passenger_capacity"`
	DriverLicenseURL     string                 `json:"driver_license_url"`
	TLCLicenseURL        string                 `json:"tlc_license_url"`
	CarRegistrationURL   string                 `json:"car_registration_url"`
	VehicleInspectionURL string                 `json:"vehicle_inspection_url"`
	TLCDiamondURL        string                 `json:"tlc_diamond_url"`
	InsuranceFilesURLs   []string               `json:"insurance_files_urls"`
	ProfilePhotoURL      string                 `json:"profile_photo_url"`
	VehiclePhotosURLs    []string               `json:"vehicle_photos_urls"`
	AdditionalInfo       map[string]interface{} `json:"additional_info"`

	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
