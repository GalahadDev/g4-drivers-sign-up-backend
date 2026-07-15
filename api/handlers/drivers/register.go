package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"g4-services/api/config"
	"g4-services/api/database"
	"g4-services/api/middleware"
	storage "g4-services/api/services"
	"g4-services/api/services/email"

	"github.com/jackc/pgx/v5/pgconn"
)

// RegisterDriver godoc
// @Summary      Register as Driver
// @Description  Submit driver application with documents and photos
// @Tags         drivers
// @Accept       multipart/form-data
// @Produce      json
// @Param        full_name            formData  string  true  "Full Name"
// @Param        address              formData  string  true  "Address"
// @Param        phone_number         formData  string  true  "Phone Number"
// @Param        emergency_number     formData  string  true  "Emergency Contact"
// @Param        device_type          formData  string  true  "Device Type (phone/tablet)"
// @Param        vehicle_type         formData  string  true  "Vehicle Type"
// @Param        passenger_capacity   formData  int     true  "Passenger Capacity"
// @Param        driver_category      formData  string  true  "Category (comfort/luxury)"
// @Param        driver_license       formData  file    true  "Driver's License"
// @Param        tlc_license          formData  file    true  "TLC License"
// @Param        car_registration     formData  file    true  "Car Registration"
// @Param        vehicle_inspection   formData  file    true  "Vehicle Inspection"
// @Param        tlc_diamond          formData  file    true  "TLC Diamond"
// @Param        profile_photo        formData  file    false "Profile Photo"
// @Param        vehicle_photos       formData  file    true  "Vehicle Photos (Array)"
// @Param        insurance_files      formData  file    true  "Insurance Files (Array)"
// @Param        additional_info      formData  string  false "JSON String with extras"
// @Success      201  {string}  string "{"status":"success"}"
// @Failure      400  {string}  string "Bad Request"
// @Failure      500  {string}  string "Server Error"
// @Router       /drivers/register [post]
// @Security     BearerAuth
func RegisterDriver(cfg *config.AppConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Cap the entire request body (all files + fields) to bound disk/RAM usage (audit A2).
		// ParseMultipartForm alone only limits in-memory bytes; the rest spills to disk unbounded.
		const maxUploadBytes = 32 << 20 // 32 MiB
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "File too large or invalid format", http.StatusBadRequest)
			return
		}

		userID, ok := r.Context().Value(middleware.ContextKeyUserID).(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Reject duplicate application early, before any file uploads
		var alreadyExists bool
		if err := database.Pool.QueryRow(r.Context(),
			"SELECT EXISTS(SELECT 1 FROM driver_applications WHERE user_id = $1)", userID,
		).Scan(&alreadyExists); err != nil {
			slog.Error("Failed to check existing application", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if alreadyExists {
			http.Error(w, "Application already submitted", http.StatusConflict)
			return
		}

		fullName := strings.TrimSpace(r.FormValue("full_name"))
		address := strings.TrimSpace(r.FormValue("address"))
		phoneNumber := strings.TrimSpace(r.FormValue("phone_number"))
		emergencyNumber := strings.TrimSpace(r.FormValue("emergency_number"))
		deviceType := strings.TrimSpace(r.FormValue("device_type"))
		vehicleType := strings.TrimSpace(r.FormValue("vehicle_type"))
		driverCategory := strings.TrimSpace(r.FormValue("driver_category"))

		if fullName == "" || address == "" || phoneNumber == "" || emergencyNumber == "" ||
			deviceType == "" || vehicleType == "" {
			http.Error(w, "Missing required text fields (name, address, phone, vehicle info)", http.StatusBadRequest)
			return
		}

		passengerCapacity, err := strconv.Atoi(r.FormValue("passenger_capacity"))
		if err != nil || passengerCapacity <= 0 {
			http.Error(w, "Invalid passenger capacity", http.StatusBadRequest)
			return
		}

		if driverCategory != "comfort" && driverCategory != "luxury" {
			http.Error(w, "Invalid category. Must be 'comfort' or 'luxury'", http.StatusBadRequest)
			return
		}

		uploadRequired := func(formKey string) (string, error) {
			file, header, err := r.FormFile(formKey)
			if err != nil {
				return "", fmt.Errorf("missing required file: %s", formKey)
			}
			defer file.Close()

			folder := fmt.Sprintf("%s/docs", userID)
			url, err := storage.UploadFile(file, header, folder, cfg)
			if err != nil {
				// Log the internal storage error but don't leak it to the client (audit M2).
				slog.Error("Storage upload failed", "key", formKey, "error", err)
				return "", fmt.Errorf("could not upload %s, please try again", formKey)
			}
			return url, nil
		}

		uploadOptional := func(formKey string) string {
			file, header, err := r.FormFile(formKey)
			if err != nil {
				return ""
			}
			defer file.Close()

			folder := fmt.Sprintf("%s/avatars", userID)
			url, err := storage.UploadFile(file, header, folder, cfg)
			if err != nil {
				slog.Error("Optional upload failed", "key", formKey, "error", err)
				return ""
			}
			return url
		}

		var driverLicenseURL, tlcLicenseURL, carRegistrationURL, vehicleInspectionURL, tlcDiamondURL string

		if driverLicenseURL, err = uploadRequired("driver_license"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if tlcLicenseURL, err = uploadRequired("tlc_license"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if carRegistrationURL, err = uploadRequired("car_registration"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if vehicleInspectionURL, err = uploadRequired("vehicle_inspection"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if tlcDiamondURL, err = uploadRequired("tlc_diamond"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		profilePhotoURL := uploadOptional("profile_photo")

		uploadMultipleRequired := func(formKey string, subfolder string) ([]string, error) {
			var urls []string
			files := r.MultipartForm.File[formKey]

			if len(files) == 0 {
				return nil, fmt.Errorf("missing required files list: %s", formKey)
			}

			// Restructure: folder became userID/subfolder
			folder := fmt.Sprintf("%s/%s", userID, subfolder)

			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err == nil {
					url, err := storage.UploadFile(file, fileHeader, folder, cfg)
					if err == nil {
						urls = append(urls, url)
					}
					file.Close()
				}
			}

			if len(urls) == 0 {
				return nil, fmt.Errorf("failed to upload any files for: %s", formKey)
			}
			return urls, nil
		}

		vehiclePhotosURLs, err := uploadMultipleRequired("vehicle_photos", "vehicles")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		insuranceFilesURLs, err := uploadMultipleRequired("insurance_files", "docs")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		additionalInfoStr := r.FormValue("additional_info")
		if additionalInfoStr == "" {
			additionalInfoStr = "{}"
		}

		var additionalMap map[string]interface{}
		if err := json.Unmarshal([]byte(additionalInfoStr), &additionalMap); err == nil {
			if refCode, ok := additionalMap["referralCode"].(string); ok && refCode != "" {
				refCode = strings.TrimSpace(refCode)
				if refCode != "" {
					slog.Info("Attempting to apply referral code from form", "user_id", userID, "code", refCode)
					// Update profile only if referred_by_code is NULL
					// We don't overwrite existing referrals
					_, err := database.Pool.Exec(r.Context(),
						`UPDATE profiles SET referred_by_code = $1 WHERE id = $2 AND referred_by_code IS NULL`,
						refCode, userID)
					if err != nil {
						slog.Error("Failed to update referral code from form", "error", err)
						// Proceeding anyway, not a fatal error for the application itself
					} else {
						slog.Info("Referral code processed successfully")
					}
				}
			}
		}

		// TODO(C3): auto-approve temporal — la feature de aprobación por admin
		// aún no está contratada. Cambiar a "pending" cuando exista la UI de revisión.
		status := "approved"

		sql := `
		INSERT INTO driver_applications (
			user_id, full_name, address, phone_number, emergency_number, device_type,
			vehicle_type, passenger_capacity, driver_category,
			driver_license_url, tlc_license_url, car_registration_url,
			vehicle_inspection_url, tlc_diamond_url, insurance_files_urls,
			profile_photo_url, vehicle_photos_urls, additional_info,
			status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 
			NULLIF($16, ''), 
			$17, $18::jsonb,
			$19
		) RETURNING id`

		var newID string
		err = database.Pool.QueryRow(r.Context(), sql,
			userID, fullName, address, phoneNumber, emergencyNumber, deviceType,
			vehicleType, passengerCapacity, driverCategory,
			driverLicenseURL, tlcLicenseURL, carRegistrationURL,
			vehicleInspectionURL, tlcDiamondURL, insuranceFilesURLs,
			profilePhotoURL, vehiclePhotosURLs,
			additionalInfoStr,
			status,
		).Scan(&newID)

		if err != nil {
			// Roll back uploaded files so a failed insert doesn't leave orphans in Storage (audit A1).
			orphans := append([]string{
				driverLicenseURL, tlcLicenseURL, carRegistrationURL,
				vehicleInspectionURL, tlcDiamondURL, profilePhotoURL,
			}, vehiclePhotosURLs...)
			orphans = append(orphans, insuranceFilesURLs...)
			storage.CleanupUploads(orphans, cfg)

			// Lost the race against a concurrent registration (UNIQUE on user_id).
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				http.Error(w, "Application already submitted", http.StatusConflict)
				return
			}

			slog.Error("Failed to insert application", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		go func() {
			ctx := context.Background()

			// Welcome email to driver
			var driverEmail string
			if err := database.Pool.QueryRow(ctx,
				"SELECT email FROM profiles WHERE id = $1", userID,
			).Scan(&driverEmail); err != nil {
				slog.Error("Failed to fetch driver email for welcome mail", "user_id", userID, "error", err)
			} else {
				subject := "Welcome to G4 Car Service!"
				body := email.WelcomeTemplate(fullName)
				for attempt := 0; attempt < 3; attempt++ {
					if err := email.SendEmail([]string{driverEmail}, subject, body, cfg); err == nil {
						break
					}
					slog.Warn("Welcome email attempt failed", "attempt", attempt+1, "user_id", userID)
					time.Sleep(time.Duration(attempt+1) * 5 * time.Second)
				}
			}

			// Admin notification
			rows, err := database.Pool.Query(ctx,
				"SELECT email FROM profiles WHERE role = 'admin'")
			if err != nil {
				slog.Error("Failed to fetch admin emails for notification", "error", err)
				return
			}
			defer rows.Close()

			var adminEmails []string
			for rows.Next() {
				var e string
				if err := rows.Scan(&e); err == nil {
					adminEmails = append(adminEmails, e)
				}
			}

			if len(adminEmails) == 0 {
				return
			}

			adminData := email.AdminNotifData{
				FullName:     fullName,
				Category:     driverCategory,
				Phone:        phoneNumber,
				RegisteredAt: time.Now(),
				DashboardURL: "https://g4-drivers-sign-up-frontend.vercel.app/admin",
			}
			if err := email.SendEmail(
				adminEmails,
				"New Driver Registration — G4 Car Service",
				email.AdminNotificationTemplate(adminData),
				cfg,
			); err != nil {
				slog.Error("Failed to send admin notification email", "error", err)
			}
		}()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": newID, "status": "success"})
	}
}
