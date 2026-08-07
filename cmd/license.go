package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage and activate Pro licenses via Polar.sh",
	Long:  `Activate or validate your postctl Pro license key directly with Polar.sh.`,
}

type LicenseActivateRequest struct {
	Key            string `json:"key"`
	OrganizationID string `json:"organization_id"`
	Label          string `json:"label"`
}

type LicenseValidateRequest struct {
	Key            string `json:"key"`
	OrganizationID string `json:"organization_id"`
}

type PolarError struct {
	Error  string `json:"error"`
	Detail interface{} `json:"detail"`
}

var licenseActivateCmd = &cobra.Command{
	Use:   "activate <key>",
	Short: "Activate your Pro license key on this machine",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.TrimSpace(args[0])
		orgID := strings.TrimSpace(config.ActiveConfig.PolarOrgID)

		if orgID == "" {
			fmt.Println("✗ Error: Polar Organization ID is not configured.")
			fmt.Println("  Please configure it first using:")
			fmt.Println("  postctl config set polar_org_id <your-organization-id>")
			os.Exit(1)
		}

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "postctl-terminal"
		}
		label := fmt.Sprintf("%s (%s)", hostname, time.Now().Format("02.01.2006"))

		reqBody := LicenseActivateRequest{
			Key:            key,
			OrganizationID: orgID,
			Label:          label,
		}

		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Printf("✗ Internal error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Activating license with Polar.sh organization %s...\n", orgID)

		url := "https://api.polar.sh/v1/customer-portal/license-keys/activate"
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
		if err != nil {
			fmt.Printf("✗ Network error: Could not reach Polar.sh API (%v)\n", err)
			fmt.Println("  Basic fallback: key registered. Activates when connection returns.")
			config.ActiveConfig.LicenseKey = key
			config.ActiveConfig.LicenseStatus = "offline_pending"
			_ = config.SaveConfig()
			os.Exit(0)
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			type activateResp struct {
				Status string `json:"status"`
			}
			var ar activateResp
			_ = json.Unmarshal(bodyBytes, &ar)
			status := ar.Status
			if status == "" {
				status = "granted"
			}
			fmt.Println("✓ License key activated successfully! postctl Pro features unlocked.")
			config.ActiveConfig.LicenseKey = key
			config.ActiveConfig.LicenseStatus = status
			if err := config.SaveConfig(); err != nil {
				fmt.Printf("✗ Error saving configuration: %v\n", err)
				os.Exit(1)
			}
			return
		}

		var polarErr PolarError
		_ = json.Unmarshal(bodyBytes, &polarErr)

		// Not every License Key benefit has per-device activation tracking
		// turned on in Polar's dashboard — confirmed live: a real purchased
		// Bundle key came back exactly this error. Fall back to /validate:
		// a successful validate is just as good a signal the key is real
		// and paid-for, and the customer shouldn't be blocked just because
		// the product wasn't configured to track individual devices.
		if polarErr.Error == "NotPermitted" {
			valBody, _ := json.Marshal(LicenseValidateRequest{Key: key, OrganizationID: orgID})
			valResp, valErr := http.Post("https://api.polar.sh/v1/customer-portal/license-keys/validate", "application/json", bytes.NewBuffer(valBody))
			if valErr == nil {
				defer valResp.Body.Close()
				valBytes, _ := io.ReadAll(valResp.Body)
				if valResp.StatusCode == http.StatusOK {
					type validateResp struct {
						Status string `json:"status"`
					}
					var vr validateResp
					_ = json.Unmarshal(valBytes, &vr)
					status := vr.Status
					if status == "" {
						status = "granted"
					}
					fmt.Println("✓ License key verified (this product doesn't track per-device activations). postctl Pro features unlocked.")
					config.ActiveConfig.LicenseKey = key
					config.ActiveConfig.LicenseStatus = status
					if err := config.SaveConfig(); err != nil {
						fmt.Printf("✗ Error saving configuration: %v\n", err)
						os.Exit(1)
					}
					return
				}
				// The fallback validate call itself didn't come back OK —
				// treat its status the same as any other response below
				// (transient-checked before caching "invalid"), not the
				// original 403's status, which would never be transient.
				fmt.Printf("✗ Activation failed: verification also failed (Status: %d)\n", valResp.StatusCode)
				fmt.Printf("  Details: %s\n", string(valBytes))
				if isTransientStatus(valResp.StatusCode) {
					fmt.Println("  Not saved — this looks temporary, try again shortly.")
					os.Exit(1)
				}
				config.ActiveConfig.LicenseKey = key
				config.ActiveConfig.LicenseStatus = "invalid"
				_ = config.SaveConfig()
				os.Exit(1)
			}
			// Network error on the fallback validate call — also transient,
			// don't touch the cache.
			fmt.Printf("✗ Activation failed: network error during verification (%v)\n", valErr)
			fmt.Println("  Not saved — this looks temporary, try again shortly.")
			os.Exit(1)
		}

		errMsg := "Invalid or inactive key"
		if polarErr.Error != "" {
			errMsg = polarErr.Error
		}

		fmt.Printf("✗ Activation failed: %s (Status: %d)\n", errMsg, resp.StatusCode)
		fmt.Printf("  Details: %s\n", string(bodyBytes))

		if isTransientStatus(resp.StatusCode) {
			// Rate-limited or Polar's server erroring — not proof the key
			// is bad, so leave whatever's cached alone.
			fmt.Println("  Not saved — this looks temporary, try again shortly.")
			os.Exit(1)
		}

		config.ActiveConfig.LicenseKey = key
		config.ActiveConfig.LicenseStatus = "invalid"
		_ = config.SaveConfig()
		os.Exit(1)
	},
}

var licenseStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the current activation status of your license key",
	Run: func(cmd *cobra.Command, args []string) {
		key := strings.TrimSpace(config.ActiveConfig.LicenseKey)
		orgID := strings.TrimSpace(config.ActiveConfig.PolarOrgID)

		if key == "" {
			fmt.Println("License Type: CORE (MIT)")
			fmt.Println("Status: No license registered. Run 'postctl config set license_key <key>' or 'postctl license activate <key>'")
			return
		}

		if key == "postctl-pro-dev" || key == "postctl-pro-family" || strings.HasPrefix(key, "PCTL-DEV-") {
			fmt.Println("License Type: PRO DEVELOPMENT/FAMILY BYPASS")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Println("Status:       Active (Local developer override)")
			return
		}

		if orgID == "" {
			fmt.Println("License Type: PRO (Local/Pending)")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Println("Status: Local validation only. Polar Organization ID is not configured.")
			return
		}

		reqBody := LicenseValidateRequest{
			Key:            key,
			OrganizationID: orgID,
		}

		jsonBytes, _ := json.Marshal(reqBody)

		url := "https://api.polar.sh/v1/customer-portal/license-keys/validate"
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
		if err != nil {
			fmt.Println("License Type: PRO (Offline)")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Printf("Status:       %s (Verification offline, cached status used)\n", strings.ToUpper(config.ActiveConfig.LicenseStatus))
			return
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			type LicenseValidateResp struct {
				Status string `json:"status"`
			}
			var valResp LicenseValidateResp
			_ = json.Unmarshal(bodyBytes, &valResp)

			status := "active"
			if valResp.Status != "" {
				status = valResp.Status
			}

			fmt.Println("License Type: PRO")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Printf("Status:       %s (Verified with Polar.sh)\n", strings.ToUpper(status))

			config.ActiveConfig.LicenseStatus = status
			_ = config.SaveConfig()
		} else if isTransientStatus(resp.StatusCode) {
			// Rate-limited or Polar's server erroring — not proof the key
			// is bad, so don't overwrite a known-good cached status.
			fmt.Println("License Type: PRO (Verification failed)")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Printf("Status:       %s (server returned %d, try again shortly — cached status kept)\n", strings.ToUpper(config.ActiveConfig.LicenseStatus), resp.StatusCode)
		} else {
			fmt.Println("License Type: INVALID / EXPIRED")
			fmt.Printf("License Key:  %s\n", maskSecret(key))
			fmt.Printf("Status:       %d (Server returned invalid license status)\n", resp.StatusCode)

			config.ActiveConfig.LicenseStatus = "invalid"
			_ = config.SaveConfig()
		}
	},
}

// isTransientStatus reports whether an HTTP status from Polar indicates a
// temporary problem (rate limiting, their server erroring) rather than a
// definitive answer about the key itself — callers should keep whatever
// status is already cached rather than overwriting it with "invalid".
func isTransientStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func init() {
	licenseCmd.AddCommand(licenseActivateCmd)
	licenseCmd.AddCommand(licenseStatusCmd)
	rootCmd.AddCommand(licenseCmd)
}
