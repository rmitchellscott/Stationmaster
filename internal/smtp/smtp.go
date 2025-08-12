package smtp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html"
	"html/template"
	"net/smtp"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/database"
)

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// EmailData holds data for email templates
type EmailData struct {
	Username    string
	ResetToken  string
	ResetURL    string
	SiteName    string
	SiteURL     string
	ExpiryHours int
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,64}$`)

func sanitizeUsername(username string) string {
	username = html.UnescapeString(strings.TrimSpace(username))

	if !usernameRegex.MatchString(username) {
		username = regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(username, "")
		if len(username) > 64 {
			username = username[:64]
		}
		if username == "" {
			username = "user"
		}
	}

	return username
}

func validateURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing hostname")
	}

	return parsedURL.String(), nil
}

func validateResetToken(token string) error {
	if token == "" {
		return fmt.Errorf("empty reset token")
	}

	if len(token) < 16 || len(token) > 128 {
		return fmt.Errorf("invalid token length")
	}

	validToken := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validToken.MatchString(token) {
		return fmt.Errorf("invalid token format")
	}

	return nil
}

func sanitizeEmailData(data *EmailData) error {
	data.Username = sanitizeUsername(data.Username)

	if data.ResetToken != "" {
		if err := validateResetToken(data.ResetToken); err != nil {
			return fmt.Errorf("invalid reset token: %w", err)
		}
	}

	if data.SiteURL != "" {
		validatedURL, err := validateURL(data.SiteURL)
		if err != nil {
			return fmt.Errorf("invalid site URL: %w", err)
		}
		data.SiteURL = validatedURL
	}

	if data.ResetURL != "" {
		validatedURL, err := validateURL(data.ResetURL)
		if err != nil {
			return fmt.Errorf("invalid reset URL: %w", err)
		}
		data.ResetURL = validatedURL
	}

	data.SiteName = html.EscapeString(strings.TrimSpace(data.SiteName))
	if data.SiteName == "" {
		data.SiteName = "Stationmaster"
	}

	if data.ExpiryHours <= 0 || data.ExpiryHours > 168 {
		data.ExpiryHours = 24
	}

	return nil
}

// GetSMTPConfig reads SMTP configuration from environment variables
func GetSMTPConfig() (*SMTPConfig, error) {
	host := config.Get("SMTP_HOST", "")
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST not configured")
	}

	portStr := config.Get("SMTP_PORT", "")
	if portStr == "" {
		portStr = "587"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	username := config.Get("SMTP_USERNAME", "")
	password := config.Get("SMTP_PASSWORD", "")
	from := config.Get("SMTP_FROM", "")

	if from == "" {
		return nil, fmt.Errorf("SMTP_FROM not configured")
	}

	useTLS := true
	if tlsStr := config.Get("SMTP_TLS", ""); tlsStr != "" {
		useTLS = strings.ToLower(tlsStr) == "true"
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		UseTLS:   useTLS,
	}, nil
}

// IsSMTPConfigured checks if SMTP is properly configured
func IsSMTPConfigured() bool {
	_, err := GetSMTPConfig()
	return err == nil
}

// SendPasswordResetEmail sends a password reset email
func SendPasswordResetEmail(email, username, resetToken string) error {
	cfg, err := GetSMTPConfig()
	if err != nil {
		return fmt.Errorf("SMTP not configured: %w", err)
	}

	if err := validateResetToken(resetToken); err != nil {
		return fmt.Errorf("invalid reset token: %w", err)
	}

	siteURL := config.Get("SITE_URL", "")
	if siteURL == "" {
		siteURL = "http://localhost:8000"
	}

	validatedSiteURL, err := validateURL(siteURL)
	if err != nil {
		return fmt.Errorf("invalid site URL: %w", err)
	}
	siteURL = validatedSiteURL

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", siteURL, resetToken)

	// Get expiry hours from system settings
	expiryStr, _ := database.GetSystemSetting("password_reset_timeout_hours")
	expiryHours := 24 // Default
	if exp, err := strconv.Atoi(expiryStr); err == nil {
		expiryHours = exp
	}

	emailData := EmailData{
		Username:    sanitizeUsername(username),
		ResetToken:  resetToken,
		ResetURL:    resetURL,
		SiteName:    "Stationmaster",
		SiteURL:     siteURL,
		ExpiryHours: expiryHours,
	}

	// Generate email content
	subject := "Password Reset"
	htmlBody, err := generatePasswordResetHTML(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate email HTML: %w", err)
	}

	textBody := generatePasswordResetText(emailData)

	// Send email
	return sendEmail(cfg, email, subject, textBody, htmlBody)
}

// SendWelcomeEmail sends a welcome email to new users
func SendWelcomeEmail(email, username string) error {
	cfg, err := GetSMTPConfig()
	if err != nil {
		return fmt.Errorf("SMTP not configured: %w", err)
	}

	siteURL := config.Get("SITE_URL", "")
	if siteURL == "" {
		siteURL = "http://localhost:8000"
	}

	validatedSiteURL, err := validateURL(siteURL)
	if err != nil {
		return fmt.Errorf("invalid site URL: %w", err)
	}

	emailData := EmailData{
		Username: sanitizeUsername(username),
		SiteName: "Stationmaster",
		SiteURL:  validatedSiteURL,
	}

	subject := "Welcome to Stationmaster!"
	htmlBody, err := generateWelcomeHTML(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate welcome email HTML: %w", err)
	}

	textBody := generateWelcomeText(emailData)

	return sendEmail(cfg, email, subject, textBody, htmlBody)
}

// sendEmail sends an email using SMTP
func sendEmail(config *SMTPConfig, to, subject, textBody, htmlBody string) error {
	// Create message
	headers := make(map[string]string)
	headers["From"] = config.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "multipart/alternative; boundary=\"boundary123\""

	var message bytes.Buffer
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")

	// Add multipart content
	message.WriteString("--boundary123\r\n")
	message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	message.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	message.WriteString("\r\n")
	message.WriteString(textBody)
	message.WriteString("\r\n")

	message.WriteString("--boundary123\r\n")
	message.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	message.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	message.WriteString("\r\n")
	message.WriteString(htmlBody)
	message.WriteString("\r\n")

	message.WriteString("--boundary123--\r\n")

	// Setup authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Send email
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	return smtp.SendMail(addr, auth, config.From, []string{to}, message.Bytes())
}

// generatePasswordResetHTML generates HTML content for password reset email
func generatePasswordResetHTML(data EmailData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Password Reset - {{.SiteName}}</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; 
            line-height: 1.6; 
            color: oklch(0 0 0); 
            background-color: oklch(1 0 0);
            margin: 0;
            padding: 0;
        }
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            background: oklch(1 0 0);
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
        }
        .header { 
            background: oklch(1 0 0); 
            padding: 32px 24px; 
            text-align: center; 
            border-bottom: 1px solid oklch(0.922 0 0);
            border-radius: 10px 10px 0 0;
        }
        .header h1 {
            margin: 0;
            color: oklch(0.205 0 0);
            font-size: 28px;
            font-weight: 600;
        }
        .content { 
            padding: 32px 24px; 
            background: oklch(1 0 0);
        }
        .content h2 {
            color: oklch(0.205 0 0);
            font-size: 20px;
            font-weight: 600;
            margin: 0 0 24px 0;
        }
        .content p {
            margin: 0 0 16px 0;
            color: oklch(0 0 0);
        }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background: oklch(0.205 0 0); 
            color: oklch(0.985 0 0); 
            text-decoration: none; 
            border-radius: 10px;
            font-weight: 500;
            margin: 24px 0;
            transition: background-color 0.2s ease;
        }
        .button:hover {
            background: oklch(0 0 0);
        }
        .link {
            color: oklch(0.205 0 0);
            text-decoration: none;
            word-break: break-all;
        }
        .link:hover {
            text-decoration: underline;
        }
        .footer { 
            background: oklch(1 0 0); 
            padding: 24px; 
            text-align: center; 
            font-size: 14px; 
            color: oklch(0.556 0 0);
            border-top: 1px solid oklch(0.922 0 0);
        }
        .footer a {
            color: oklch(0.556 0 0);
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
        .warning {
            background: oklch(0.98 0 0);
            border: 1px solid oklch(0.922 0 0);
            border-radius: 8px;
            padding: 16px;
            margin: 24px 0;
        }
        .warning p {
            margin: 0;
            color: oklch(0.556 0 0);
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div style="background: oklch(1 0 0); padding: 40px 20px;">
        <div class="container">
            <div class="header">
                <svg width="200" height="65" viewBox="0 0 988 323" xmlns="http://www.w3.org/2000/svg" style="display: block; margin: 0 auto;">
                    <g transform="matrix(1,0,0,1,-46.3057,-292.948)">
                        <g transform="matrix(1,0,0,1,4.8518,-56.825)">
                            <g transform="matrix(1,0,0,1,-31.1644,-257.495)">
                                <path d="M73.32,930.21C71.87,925.906 72.741,916.643 75.256,916.658C82.758,916.7 90.429,914.261 92.667,908.211C104.082,877.359 142.044,745.526 151.715,711.792L143.462,711.792C140.472,711.792 137.731,710.124 136.357,707.468C134.983,704.811 135.205,701.611 136.933,699.17L199.595,610.646C201.095,608.527 203.529,607.268 206.125,607.268C208.72,607.268 211.155,608.527 212.654,610.646L275.317,699.17C277.045,701.611 277.267,704.811 275.893,707.468C274.518,710.124 271.778,711.792 268.787,711.792L261.071,711.792C271.956,745.103 317.781,903.406 319.054,905.474C326.256,917.184 332.309,915.247 335.813,915.84C340.629,916.655 339.561,925.79 337.131,929.945C310.078,925.574 277.327,922.526 241.427,921.304C241.531,919.185 242.216,917.183 243.542,916.28C247.086,913.867 255.783,915.479 253.95,904.456C252.253,894.255 234.102,844.999 234.102,844.999L177.663,845.185C177.663,845.185 160.86,894.269 157.912,905.003C156.42,910.431 163.718,913.934 169.341,915.814C170.673,916.259 171.308,918.688 171.249,921.284C134.391,922.516 100.832,925.672 73.32,930.21ZM254.324,682.733C250.924,682.678 245.868,681.826 242.355,681.872C241.753,681.88 241.027,685.533 242.321,685.586C247.043,685.779 252.18,686.409 257.177,686.91C260.656,687.258 255.624,682.753 254.324,682.733ZM225.192,757.595C223.754,748.254 215.672,741.091 205.931,741.091C196.19,741.091 188.109,748.254 186.671,757.595L186.637,757.669L176.637,804.131C176.637,806.573 178.62,808.556 181.062,808.556C197.674,807.752 214.253,807.775 230.801,808.556C233.243,808.556 235.226,806.573 235.226,804.131L225.226,757.669L225.192,757.595ZM266.141,768.45C261.509,767.473 260.129,767.096 255.348,767.403C253.384,767.53 252.556,775.669 255.663,775.337C260.407,774.83 263.86,775.87 268.815,776.39C270.583,776.575 268.544,768.957 266.141,768.45ZM158.545,767.773C154.925,767.325 146.237,768.927 143.015,769.234C141.585,769.37 138.458,777.433 140.697,777.012C142.314,776.707 151.922,774.964 158.715,774.833C160.011,774.808 159.835,767.933 158.545,767.773ZM187.926,711.115C184.548,710.784 167.592,712.256 164.096,712.271C162.661,712.278 160.073,717.9 162.321,717.528C164.616,717.149 181.72,715.532 186.716,715.944C189.667,716.188 189.973,711.316 187.926,711.115ZM295.523,859.778C292.789,859.614 285.276,858.139 282.493,858.156C281.35,858.163 280.403,864.541 282.217,864.512C284.949,864.468 295.164,866.544 298.071,866.432C299.102,866.392 296.556,859.84 295.523,859.778ZM259.081,732.514C256.04,731.855 252.139,731.884 248.177,731.756C246.21,731.693 245.275,737.464 248.399,737.438C253.104,737.398 254.266,738.422 259.272,738.082C261.046,737.962 260.691,732.863 259.081,732.514ZM229.495,679.777C226.06,679.146 192.545,678.816 189.102,679.462C186.611,679.929 185.29,684.694 187.567,684.611C204.568,683.991 211.933,684.116 229.485,685.045C230.78,685.113 230.773,680.012 229.495,679.777ZM273.17,857.172C270.332,856.631 257.557,854.994 253.072,855.503C251.573,855.674 250.187,863.966 252.58,863.937C256.185,863.893 269.388,864.687 273.365,865.019C274.722,865.133 274.511,857.428 273.17,857.172ZM163.521,855.749C160.102,854.895 146.543,856.03 142.89,856.807C141.674,857.066 140.962,864.836 142.213,864.762C145.186,864.585 159.003,863.328 162.946,863.495C164.263,863.551 164.81,856.071 163.521,855.749ZM156.856,797.495C153.456,796.88 136.247,799.451 132.707,800.224C131.304,800.53 130.582,807.128 132.86,807.102C136.292,807.062 155.163,805.078 158.814,804.976C160.11,804.94 158.135,797.727 156.856,797.495ZM164.629,736.882C157.754,737.301 156.708,737.398 152.693,738.332C151.295,738.657 149.174,744.296 152.137,744.281C154.193,744.271 161.143,743.523 164.794,743.421C166.09,743.385 165.926,736.803 164.629,736.882ZM233.508,653.004C231.102,652.922 224.566,652.608 222.118,652.617C221.112,652.62 220.466,655.96 222.056,655.836C224.514,655.645 233.523,656.226 236.001,656.415C236.906,656.485 234.416,653.035 233.508,653.004ZM134.063,858.023C130.939,857.351 118.643,859.37 114.995,860.192C113.735,860.475 112.484,866.027 113.785,866.115C116.881,866.325 129.412,864.476 133.175,864.376C134.443,864.343 135.315,858.292 134.063,858.023ZM304.297,889.06C299.256,888.049 279.32,885.26 274.023,886.23C273.003,886.416 272.468,894.41 274.627,894.284C277.111,894.139 297.764,896.773 305.628,898.234C308.666,898.798 306.281,889.459 304.297,889.06ZM244.655,823.169C244.213,821.821 239.995,819.769 238.581,819.769C214.351,817.53 193.826,818.35 169.978,820.607C168.564,820.607 167.416,821.755 167.416,823.169L167.416,824.814C167.416,826.228 171.036,827.252 172.45,827.252C196.088,824.139 217.644,824.226 242.093,827.377C243.507,827.377 244.655,826.228 244.655,824.814L244.655,823.169ZM143.553,885.115C137.937,884.747 114.683,887.653 108.296,889.062C106.026,889.563 103.812,898.171 106.123,897.919C112.387,897.236 135.698,893.375 142.167,893.982C144.439,894.195 145.831,885.264 143.553,885.115ZM287.536,829.98C282.853,829.363 265.215,827.244 260.393,827.888C258.442,828.148 257.005,834.295 260.091,833.808C264.698,833.08 282.383,835.877 287.499,836.547C289.262,836.778 289.303,830.213 287.536,829.98ZM177.913,680.261C174.408,679.929 162.744,680.997 159.132,681.787C157.729,682.094 153.868,687.085 156.13,686.817C162.859,686.017 170.172,685.418 177.919,685.202C179.216,685.166 179.207,680.383 177.913,680.261ZM194.246,659.483C190.783,659.08 181.938,659.734 178.519,659.956C177.086,660.049 177.388,665.27 179.154,665.206C182.583,665.082 190.641,664.672 194.292,664.571C195.588,664.535 195.537,659.633 194.246,659.483ZM211.2,635.03C207.823,634.632 199.288,634.73 195.785,634.887C194.351,634.951 190.922,641.952 193.194,641.791C198.723,641.397 210.113,641.361 211.485,641.515C212.774,641.661 212.491,635.182 211.2,635.03ZM279.156,800.152C274.415,799.432 257.128,797.105 252.355,797.495C250.393,797.655 248.904,805.497 251.998,805.069C256.754,804.41 274.31,806.921 279.251,807.613C281.012,807.859 280.918,800.419 279.156,800.152ZM152.973,828.247C148.163,827.367 130.482,829.602 125.79,830.511C123.857,830.885 122.627,836.623 125.735,836.311C130.32,835.85 147.7,833.589 152.841,833.958C154.615,834.085 154.727,828.567 152.973,828.247Z" style="fill: oklch(0.205 0 0);"/>
                            </g>
                            <g transform="matrix(0.871984,0,0,0.871984,-4.99441,238.549)">
                                <g transform="matrix(288,0,0,288,321.327,495.769)">
                                    <path d="M0.084,-0.619C0.079,-0.633 0.074,-0.644 0.068,-0.652C0.062,-0.659 0.055,-0.665 0.046,-0.668C0.037,-0.671 0.026,-0.672 0.013,-0.672L0,-0.672L0,-0.714L0.275,-0.714L0.275,-0.672L0.252,-0.672C0.232,-0.672 0.217,-0.668 0.207,-0.661C0.197,-0.653 0.192,-0.641 0.192,-0.624C0.192,-0.621 0.192,-0.617 0.193,-0.613C0.194,-0.609 0.195,-0.605 0.196,-0.601C0.197,-0.597 0.199,-0.592 0.2,-0.587L0.31,-0.262C0.317,-0.243 0.323,-0.223 0.328,-0.204C0.333,-0.185 0.338,-0.166 0.343,-0.149C0.348,-0.131 0.352,-0.114 0.355,-0.098C0.358,-0.114 0.362,-0.131 0.367,-0.148C0.371,-0.165 0.376,-0.184 0.382,-0.203C0.387,-0.222 0.394,-0.241 0.401,-0.262L0.511,-0.58C0.513,-0.585 0.515,-0.591 0.516,-0.596C0.517,-0.601 0.518,-0.606 0.519,-0.611C0.52,-0.616 0.52,-0.62 0.52,-0.623C0.52,-0.64 0.515,-0.653 0.504,-0.661C0.493,-0.668 0.476,-0.672 0.454,-0.672L0.431,-0.672L0.431,-0.714L0.675,-0.714L0.675,-0.672L0.656,-0.672C0.643,-0.672 0.633,-0.67 0.624,-0.666C0.615,-0.662 0.608,-0.655 0.601,-0.644C0.594,-0.632 0.587,-0.616 0.58,-0.594L0.373,-0L0.3,-0L0.084,-0.619Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,515.727,495.769)">
                                    <path d="M0.038,-0L0.038,-0.042L0.051,-0.042C0.066,-0.042 0.08,-0.044 0.093,-0.047C0.105,-0.05 0.115,-0.057 0.122,-0.068C0.129,-0.078 0.133,-0.093 0.133,-0.114L0.133,-0.6C0.133,-0.621 0.129,-0.637 0.122,-0.647C0.115,-0.657 0.105,-0.664 0.093,-0.667C0.08,-0.67 0.066,-0.672 0.051,-0.672L0.038,-0.672L0.038,-0.714L0.329,-0.714L0.329,-0.672L0.316,-0.672C0.301,-0.672 0.288,-0.67 0.275,-0.667C0.262,-0.664 0.252,-0.657 0.245,-0.647C0.238,-0.637 0.234,-0.621 0.234,-0.6L0.234,-0.114C0.234,-0.093 0.238,-0.078 0.245,-0.068C0.252,-0.057 0.262,-0.05 0.275,-0.047C0.288,-0.044 0.301,-0.042 0.316,-0.042L0.329,-0.042L0.329,-0L0.038,-0Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,621.423,495.769)">
                                    <path d="M0,-0L0,-0.042L0.019,-0.042C0.032,-0.042 0.043,-0.044 0.051,-0.048C0.059,-0.052 0.066,-0.06 0.073,-0.071C0.08,-0.082 0.087,-0.099 0.095,-0.12L0.317,-0.714L0.395,-0.714L0.621,-0.095C0.626,-0.081 0.632,-0.07 0.638,-0.063C0.644,-0.055 0.651,-0.05 0.66,-0.047C0.669,-0.044 0.679,-0.042 0.692,-0.042L0.705,-0.042L0.705,-0L0.43,-0L0.43,-0.042L0.453,-0.042C0.473,-0.042 0.488,-0.046 0.498,-0.054C0.508,-0.061 0.513,-0.073 0.513,-0.09C0.513,-0.094 0.513,-0.098 0.512,-0.102C0.511,-0.105 0.511,-0.109 0.51,-0.114C0.509,-0.118 0.507,-0.122 0.505,-0.127L0.465,-0.239L0.202,-0.239L0.164,-0.134C0.162,-0.129 0.16,-0.123 0.159,-0.118C0.158,-0.113 0.157,-0.108 0.156,-0.104C0.155,-0.099 0.155,-0.095 0.155,-0.091C0.155,-0.074 0.161,-0.062 0.172,-0.054C0.183,-0.046 0.199,-0.042 0.221,-0.042L0.244,-0.042L0.244,-0L0,-0ZM0.221,-0.289L0.447,-0.289L0.385,-0.464C0.378,-0.484 0.372,-0.503 0.366,-0.521C0.359,-0.539 0.354,-0.556 0.349,-0.573C0.343,-0.59 0.339,-0.606 0.335,-0.622C0.332,-0.606 0.328,-0.591 0.324,-0.576C0.319,-0.561 0.314,-0.545 0.309,-0.529C0.304,-0.512 0.297,-0.494 0.289,-0.473L0.221,-0.289Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,824.463,495.769)">
                                    <path d="M0.038,-0L0.038,-0.042L0.051,-0.042C0.066,-0.042 0.079,-0.044 0.092,-0.047C0.104,-0.05 0.114,-0.056 0.122,-0.066C0.129,-0.075 0.133,-0.09 0.133,-0.109L0.133,-0.604C0.133,-0.624 0.129,-0.639 0.122,-0.649C0.114,-0.658 0.104,-0.665 0.092,-0.668C0.079,-0.671 0.066,-0.672 0.051,-0.672L0.038,-0.672L0.038,-0.714L0.307,-0.714C0.365,-0.714 0.413,-0.707 0.451,-0.693C0.489,-0.678 0.517,-0.657 0.536,-0.629C0.555,-0.6 0.564,-0.564 0.564,-0.521C0.564,-0.486 0.557,-0.456 0.542,-0.432C0.527,-0.407 0.509,-0.388 0.487,-0.373C0.464,-0.358 0.441,-0.347 0.417,-0.339L0.554,-0.122C0.571,-0.095 0.588,-0.075 0.604,-0.062C0.62,-0.049 0.638,-0.042 0.659,-0.042L0.662,-0.042L0.662,-0L0.648,-0C0.607,-0 0.575,-0.002 0.552,-0.007C0.529,-0.011 0.51,-0.02 0.496,-0.033C0.482,-0.046 0.467,-0.065 0.452,-0.09L0.317,-0.315L0.234,-0.315L0.234,-0.109C0.234,-0.09 0.238,-0.075 0.246,-0.066C0.253,-0.056 0.263,-0.05 0.276,-0.047C0.288,-0.044 0.301,-0.042 0.316,-0.042L0.329,-0.042L0.329,-0L0.038,-0ZM0.304,-0.362C0.363,-0.362 0.403,-0.375 0.425,-0.401C0.446,-0.427 0.457,-0.466 0.457,-0.518C0.457,-0.554 0.452,-0.583 0.442,-0.605C0.432,-0.626 0.416,-0.642 0.393,-0.652C0.37,-0.661 0.34,-0.666 0.302,-0.666L0.234,-0.666L0.234,-0.362L0.304,-0.362Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,1004.75,495.769)">
                                    <path d="M0.159,-0L0.159,-0.042L0.182,-0.042C0.197,-0.042 0.21,-0.044 0.223,-0.047C0.235,-0.05 0.245,-0.056 0.253,-0.066C0.26,-0.075 0.264,-0.09 0.264,-0.109L0.264,-0.298L0.079,-0.619C0.072,-0.632 0.065,-0.642 0.059,-0.65C0.052,-0.657 0.045,-0.663 0.037,-0.667C0.029,-0.67 0.019,-0.672 0.008,-0.672L-0.005,-0.672L-0.005,-0.714L0.27,-0.714L0.27,-0.672L0.233,-0.672C0.215,-0.672 0.203,-0.669 0.198,-0.663C0.192,-0.656 0.189,-0.649 0.189,-0.64C0.189,-0.631 0.191,-0.621 0.195,-0.612C0.199,-0.603 0.203,-0.594 0.207,-0.587L0.281,-0.453C0.292,-0.432 0.302,-0.412 0.311,-0.392C0.319,-0.372 0.326,-0.354 0.331,-0.339C0.337,-0.353 0.346,-0.37 0.357,-0.391C0.368,-0.412 0.38,-0.432 0.391,-0.453L0.455,-0.568C0.462,-0.579 0.466,-0.59 0.469,-0.601C0.472,-0.611 0.473,-0.62 0.473,-0.628C0.473,-0.643 0.468,-0.654 0.458,-0.661C0.447,-0.668 0.432,-0.672 0.413,-0.672L0.384,-0.672L0.384,-0.714L0.628,-0.714L0.628,-0.672L0.616,-0.672C0.607,-0.672 0.598,-0.67 0.59,-0.665C0.581,-0.66 0.572,-0.652 0.564,-0.641C0.555,-0.63 0.544,-0.614 0.533,-0.594L0.365,-0.298L0.365,-0.114C0.365,-0.093 0.369,-0.078 0.376,-0.068C0.383,-0.057 0.393,-0.05 0.406,-0.047C0.419,-0.044 0.432,-0.042 0.447,-0.042L0.47,-0.042L0.47,-0L0.159,-0Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                            </g>
                        </g>
                    </g>
                </svg>
            </div>
            <div class="content">
                <h2>Password Reset Request</h2>
                <p>Hello <strong>{{.Username}}</strong>,</p>
                <p>We received a request to reset your password for your {{.SiteName}} account.</p>
                <p>Click the button below to reset your password:</p>
                <div style="text-align: center;">
                    <a href="{{.ResetURL}}" class="button">Reset Password</a>
                </div>
                <p>If the button doesn't work, copy and paste this link into your browser:</p>
                <p><a href="{{.ResetURL}}" class="link">{{.ResetURL}}</a></p>
                <div class="warning">
                    <p><strong>Important:</strong> This link will expire in {{.ExpiryHours}} hours for security reasons.</p>
                </div>
                <p>If you didn't request this password reset, please ignore this email. Your password will remain unchanged and your account is secure.</p>
            </div>
            <div class="footer">
                <p>This email was sent by {{.SiteName}} • <a href="{{.SiteURL}}">{{.SiteURL}}</a></p>
            </div>
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("password_reset").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generatePasswordResetText generates plain text content for password reset email
func generatePasswordResetText(data EmailData) string {
	return fmt.Sprintf(`Password Reset Request - %s

Hello %s,

We received a request to reset your password for your %s account.

To reset your password, please visit the following link:
%s

This link will expire in %d hours.

If you didn't request this password reset, please ignore this email. Your password will remain unchanged.

--
This email was sent by %s (%s)
`, data.SiteName, data.Username, data.SiteName, data.ResetURL, data.ExpiryHours, data.SiteName, data.SiteURL)
}

// generateWelcomeHTML generates HTML content for welcome email
func generateWelcomeHTML(data EmailData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to {{.SiteName}}</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; 
            line-height: 1.6; 
            color: oklch(0 0 0); 
            background-color: oklch(1 0 0);
            margin: 0;
            padding: 0;
        }
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            background: oklch(1 0 0);
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
        }
        .header { 
            background: oklch(1 0 0); 
            padding: 32px 24px; 
            text-align: center; 
            border-bottom: 1px solid oklch(0.922 0 0);
            border-radius: 10px 10px 0 0;
        }
        .header h1 {
            margin: 0;
            color: oklch(0.205 0 0);
            font-size: 28px;
            font-weight: 600;
        }
        .content { 
            padding: 32px 24px; 
            background: oklch(1 0 0);
        }
        .content h2 {
            color: oklch(0.205 0 0);
            font-size: 20px;
            font-weight: 600;
            margin: 0 0 24px 0;
        }
        .content p {
            margin: 0 0 16px 0;
            color: oklch(0 0 0);
        }
        .content ul {
            margin: 16px 0;
            padding-left: 24px;
            color: oklch(0 0 0);
        }
        .content li {
            margin: 8px 0;
        }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            background: oklch(0.205 0 0); 
            color: oklch(0.985 0 0); 
            text-decoration: none; 
            border-radius: 10px;
            font-weight: 500;
            margin: 24px 0;
            transition: background-color 0.2s ease;
        }
        .button:hover {
            background: oklch(0 0 0);
        }
        .footer { 
            background: oklch(1 0 0); 
            padding: 24px; 
            text-align: center; 
            font-size: 14px; 
            color: oklch(0.556 0 0);
            border-top: 1px solid oklch(0.922 0 0);
        }
        .footer a {
            color: oklch(0.556 0 0);
            text-decoration: none;
        }
        .footer a:hover {
            text-decoration: underline;
        }
        .feature-box {
            background: oklch(0.98 0 0);
            border: 1px solid oklch(0.922 0 0);
            border-radius: 8px;
            padding: 20px;
            margin: 24px 0;
        }
    </style>
</head>
<body>
    <div style="background: oklch(1 0 0); padding: 40px 20px;">
        <div class="container">
            <div class="header">
                <svg width="200" height="65" viewBox="0 0 988 323" xmlns="http://www.w3.org/2000/svg" style="display: block; margin: 0 auto;">
                    <g transform="matrix(1,0,0,1,-46.3057,-292.948)">
                        <g transform="matrix(1,0,0,1,4.8518,-56.825)">
                            <g transform="matrix(1,0,0,1,-31.1644,-257.495)">
                                <path d="M73.32,930.21C71.87,925.906 72.741,916.643 75.256,916.658C82.758,916.7 90.429,914.261 92.667,908.211C104.082,877.359 142.044,745.526 151.715,711.792L143.462,711.792C140.472,711.792 137.731,710.124 136.357,707.468C134.983,704.811 135.205,701.611 136.933,699.17L199.595,610.646C201.095,608.527 203.529,607.268 206.125,607.268C208.72,607.268 211.155,608.527 212.654,610.646L275.317,699.17C277.045,701.611 277.267,704.811 275.893,707.468C274.518,710.124 271.778,711.792 268.787,711.792L261.071,711.792C271.956,745.103 317.781,903.406 319.054,905.474C326.256,917.184 332.309,915.247 335.813,915.84C340.629,916.655 339.561,925.79 337.131,929.945C310.078,925.574 277.327,922.526 241.427,921.304C241.531,919.185 242.216,917.183 243.542,916.28C247.086,913.867 255.783,915.479 253.95,904.456C252.253,894.255 234.102,844.999 234.102,844.999L177.663,845.185C177.663,845.185 160.86,894.269 157.912,905.003C156.42,910.431 163.718,913.934 169.341,915.814C170.673,916.259 171.308,918.688 171.249,921.284C134.391,922.516 100.832,925.672 73.32,930.21ZM254.324,682.733C250.924,682.678 245.868,681.826 242.355,681.872C241.753,681.88 241.027,685.533 242.321,685.586C247.043,685.779 252.18,686.409 257.177,686.91C260.656,687.258 255.624,682.753 254.324,682.733ZM225.192,757.595C223.754,748.254 215.672,741.091 205.931,741.091C196.19,741.091 188.109,748.254 186.671,757.595L186.637,757.669L176.637,804.131C176.637,806.573 178.62,808.556 181.062,808.556C197.674,807.752 214.253,807.775 230.801,808.556C233.243,808.556 235.226,806.573 235.226,804.131L225.226,757.669L225.192,757.595ZM266.141,768.45C261.509,767.473 260.129,767.096 255.348,767.403C253.384,767.53 252.556,775.669 255.663,775.337C260.407,774.83 263.86,775.87 268.815,776.39C270.583,776.575 268.544,768.957 266.141,768.45ZM158.545,767.773C154.925,767.325 146.237,768.927 143.015,769.234C141.585,769.37 138.458,777.433 140.697,777.012C142.314,776.707 151.922,774.964 158.715,774.833C160.011,774.808 159.835,767.933 158.545,767.773ZM187.926,711.115C184.548,710.784 167.592,712.256 164.096,712.271C162.661,712.278 160.073,717.9 162.321,717.528C164.616,717.149 181.72,715.532 186.716,715.944C189.667,716.188 189.973,711.316 187.926,711.115ZM295.523,859.778C292.789,859.614 285.276,858.139 282.493,858.156C281.35,858.163 280.403,864.541 282.217,864.512C284.949,864.468 295.164,866.544 298.071,866.432C299.102,866.392 296.556,859.84 295.523,859.778ZM259.081,732.514C256.04,731.855 252.139,731.884 248.177,731.756C246.21,731.693 245.275,737.464 248.399,737.438C253.104,737.398 254.266,738.422 259.272,738.082C261.046,737.962 260.691,732.863 259.081,732.514ZM229.495,679.777C226.06,679.146 192.545,678.816 189.102,679.462C186.611,679.929 185.29,684.694 187.567,684.611C204.568,683.991 211.933,684.116 229.485,685.045C230.78,685.113 230.773,680.012 229.495,679.777ZM273.17,857.172C270.332,856.631 257.557,854.994 253.072,855.503C251.573,855.674 250.187,863.966 252.58,863.937C256.185,863.893 269.388,864.687 273.365,865.019C274.722,865.133 274.511,857.428 273.17,857.172ZM163.521,855.749C160.102,854.895 146.543,856.03 142.89,856.807C141.674,857.066 140.962,864.836 142.213,864.762C145.186,864.585 159.003,863.328 162.946,863.495C164.263,863.551 164.81,856.071 163.521,855.749ZM156.856,797.495C153.456,796.88 136.247,799.451 132.707,800.224C131.304,800.53 130.582,807.128 132.86,807.102C136.292,807.062 155.163,805.078 158.814,804.976C160.11,804.94 158.135,797.727 156.856,797.495ZM164.629,736.882C157.754,737.301 156.708,737.398 152.693,738.332C151.295,738.657 149.174,744.296 152.137,744.281C154.193,744.271 161.143,743.523 164.794,743.421C166.09,743.385 165.926,736.803 164.629,736.882ZM233.508,653.004C231.102,652.922 224.566,652.608 222.118,652.617C221.112,652.62 220.466,655.96 222.056,655.836C224.514,655.645 233.523,656.226 236.001,656.415C236.906,656.485 234.416,653.035 233.508,653.004ZM134.063,858.023C130.939,857.351 118.643,859.37 114.995,860.192C113.735,860.475 112.484,866.027 113.785,866.115C116.881,866.325 129.412,864.476 133.175,864.376C134.443,864.343 135.315,858.292 134.063,858.023ZM304.297,889.06C299.256,888.049 279.32,885.26 274.023,886.23C273.003,886.416 272.468,894.41 274.627,894.284C277.111,894.139 297.764,896.773 305.628,898.234C308.666,898.798 306.281,889.459 304.297,889.06ZM244.655,823.169C244.213,821.821 239.995,819.769 238.581,819.769C214.351,817.53 193.826,818.35 169.978,820.607C168.564,820.607 167.416,821.755 167.416,823.169L167.416,824.814C167.416,826.228 171.036,827.252 172.45,827.252C196.088,824.139 217.644,824.226 242.093,827.377C243.507,827.377 244.655,826.228 244.655,824.814L244.655,823.169ZM143.553,885.115C137.937,884.747 114.683,887.653 108.296,889.062C106.026,889.563 103.812,898.171 106.123,897.919C112.387,897.236 135.698,893.375 142.167,893.982C144.439,894.195 145.831,885.264 143.553,885.115ZM287.536,829.98C282.853,829.363 265.215,827.244 260.393,827.888C258.442,828.148 257.005,834.295 260.091,833.808C264.698,833.08 282.383,835.877 287.499,836.547C289.262,836.778 289.303,830.213 287.536,829.98ZM177.913,680.261C174.408,679.929 162.744,680.997 159.132,681.787C157.729,682.094 153.868,687.085 156.13,686.817C162.859,686.017 170.172,685.418 177.919,685.202C179.216,685.166 179.207,680.383 177.913,680.261ZM194.246,659.483C190.783,659.08 181.938,659.734 178.519,659.956C177.086,660.049 177.388,665.27 179.154,665.206C182.583,665.082 190.641,664.672 194.292,664.571C195.588,664.535 195.537,659.633 194.246,659.483ZM211.2,635.03C207.823,634.632 199.288,634.73 195.785,634.887C194.351,634.951 190.922,641.952 193.194,641.791C198.723,641.397 210.113,641.361 211.485,641.515C212.774,641.661 212.491,635.182 211.2,635.03ZM279.156,800.152C274.415,799.432 257.128,797.105 252.355,797.495C250.393,797.655 248.904,805.497 251.998,805.069C256.754,804.41 274.31,806.921 279.251,807.613C281.012,807.859 280.918,800.419 279.156,800.152ZM152.973,828.247C148.163,827.367 130.482,829.602 125.79,830.511C123.857,830.885 122.627,836.623 125.735,836.311C130.32,835.85 147.7,833.589 152.841,833.958C154.615,834.085 154.727,828.567 152.973,828.247Z" style="fill: oklch(0.205 0 0);"/>
                            </g>
                            <g transform="matrix(0.871984,0,0,0.871984,-4.99441,238.549)">
                                <g transform="matrix(288,0,0,288,321.327,495.769)">
                                    <path d="M0.084,-0.619C0.079,-0.633 0.074,-0.644 0.068,-0.652C0.062,-0.659 0.055,-0.665 0.046,-0.668C0.037,-0.671 0.026,-0.672 0.013,-0.672L0,-0.672L0,-0.714L0.275,-0.714L0.275,-0.672L0.252,-0.672C0.232,-0.672 0.217,-0.668 0.207,-0.661C0.197,-0.653 0.192,-0.641 0.192,-0.624C0.192,-0.621 0.192,-0.617 0.193,-0.613C0.194,-0.609 0.195,-0.605 0.196,-0.601C0.197,-0.597 0.199,-0.592 0.2,-0.587L0.31,-0.262C0.317,-0.243 0.323,-0.223 0.328,-0.204C0.333,-0.185 0.338,-0.166 0.343,-0.149C0.348,-0.131 0.352,-0.114 0.355,-0.098C0.358,-0.114 0.362,-0.131 0.367,-0.148C0.371,-0.165 0.376,-0.184 0.382,-0.203C0.387,-0.222 0.394,-0.241 0.401,-0.262L0.511,-0.58C0.513,-0.585 0.515,-0.591 0.516,-0.596C0.517,-0.601 0.518,-0.606 0.519,-0.611C0.52,-0.616 0.52,-0.62 0.52,-0.623C0.52,-0.64 0.515,-0.653 0.504,-0.661C0.493,-0.668 0.476,-0.672 0.454,-0.672L0.431,-0.672L0.431,-0.714L0.675,-0.714L0.675,-0.672L0.656,-0.672C0.643,-0.672 0.633,-0.67 0.624,-0.666C0.615,-0.662 0.608,-0.655 0.601,-0.644C0.594,-0.632 0.587,-0.616 0.58,-0.594L0.373,-0L0.3,-0L0.084,-0.619Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,515.727,495.769)">
                                    <path d="M0.038,-0L0.038,-0.042L0.051,-0.042C0.066,-0.042 0.08,-0.044 0.093,-0.047C0.105,-0.05 0.115,-0.057 0.122,-0.068C0.129,-0.078 0.133,-0.093 0.133,-0.114L0.133,-0.6C0.133,-0.621 0.129,-0.637 0.122,-0.647C0.115,-0.657 0.105,-0.664 0.093,-0.667C0.08,-0.67 0.066,-0.672 0.051,-0.672L0.038,-0.672L0.038,-0.714L0.329,-0.714L0.329,-0.672L0.316,-0.672C0.301,-0.672 0.288,-0.67 0.275,-0.667C0.262,-0.664 0.252,-0.657 0.245,-0.647C0.238,-0.637 0.234,-0.621 0.234,-0.6L0.234,-0.114C0.234,-0.093 0.238,-0.078 0.245,-0.068C0.252,-0.057 0.262,-0.05 0.275,-0.047C0.288,-0.044 0.301,-0.042 0.316,-0.042L0.329,-0.042L0.329,-0L0.038,-0Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,621.423,495.769)">
                                    <path d="M0,-0L0,-0.042L0.019,-0.042C0.032,-0.042 0.043,-0.044 0.051,-0.048C0.059,-0.052 0.066,-0.06 0.073,-0.071C0.08,-0.082 0.087,-0.099 0.095,-0.12L0.317,-0.714L0.395,-0.714L0.621,-0.095C0.626,-0.081 0.632,-0.07 0.638,-0.063C0.644,-0.055 0.651,-0.05 0.66,-0.047C0.669,-0.044 0.679,-0.042 0.692,-0.042L0.705,-0.042L0.705,-0L0.43,-0L0.43,-0.042L0.453,-0.042C0.473,-0.042 0.488,-0.046 0.498,-0.054C0.508,-0.061 0.513,-0.073 0.513,-0.09C0.513,-0.094 0.513,-0.098 0.512,-0.102C0.511,-0.105 0.511,-0.109 0.51,-0.114C0.509,-0.118 0.507,-0.122 0.505,-0.127L0.465,-0.239L0.202,-0.239L0.164,-0.134C0.162,-0.129 0.16,-0.123 0.159,-0.118C0.158,-0.113 0.157,-0.108 0.156,-0.104C0.155,-0.099 0.155,-0.095 0.155,-0.091C0.155,-0.074 0.161,-0.062 0.172,-0.054C0.183,-0.046 0.199,-0.042 0.221,-0.042L0.244,-0.042L0.244,-0L0,-0ZM0.221,-0.289L0.447,-0.289L0.385,-0.464C0.378,-0.484 0.372,-0.503 0.366,-0.521C0.359,-0.539 0.354,-0.556 0.349,-0.573C0.343,-0.59 0.339,-0.606 0.335,-0.622C0.332,-0.606 0.328,-0.591 0.324,-0.576C0.319,-0.561 0.314,-0.545 0.309,-0.529C0.304,-0.512 0.297,-0.494 0.289,-0.473L0.221,-0.289Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,824.463,495.769)">
                                    <path d="M0.038,-0L0.038,-0.042L0.051,-0.042C0.066,-0.042 0.079,-0.044 0.092,-0.047C0.104,-0.05 0.114,-0.056 0.122,-0.066C0.129,-0.075 0.133,-0.09 0.133,-0.109L0.133,-0.604C0.133,-0.624 0.129,-0.639 0.122,-0.649C0.114,-0.658 0.104,-0.665 0.092,-0.668C0.079,-0.671 0.066,-0.672 0.051,-0.672L0.038,-0.672L0.038,-0.714L0.307,-0.714C0.365,-0.714 0.413,-0.707 0.451,-0.693C0.489,-0.678 0.517,-0.657 0.536,-0.629C0.555,-0.6 0.564,-0.564 0.564,-0.521C0.564,-0.486 0.557,-0.456 0.542,-0.432C0.527,-0.407 0.509,-0.388 0.487,-0.373C0.464,-0.358 0.441,-0.347 0.417,-0.339L0.554,-0.122C0.571,-0.095 0.588,-0.075 0.604,-0.062C0.62,-0.049 0.638,-0.042 0.659,-0.042L0.662,-0.042L0.662,-0L0.648,-0C0.607,-0 0.575,-0.002 0.552,-0.007C0.529,-0.011 0.51,-0.02 0.496,-0.033C0.482,-0.046 0.467,-0.065 0.452,-0.09L0.317,-0.315L0.234,-0.315L0.234,-0.109C0.234,-0.09 0.238,-0.075 0.246,-0.066C0.253,-0.056 0.263,-0.05 0.276,-0.047C0.288,-0.044 0.301,-0.042 0.316,-0.042L0.329,-0.042L0.329,-0L0.038,-0ZM0.304,-0.362C0.363,-0.362 0.403,-0.375 0.425,-0.401C0.446,-0.427 0.457,-0.466 0.457,-0.518C0.457,-0.554 0.452,-0.583 0.442,-0.605C0.432,-0.626 0.416,-0.642 0.393,-0.652C0.37,-0.661 0.34,-0.666 0.302,-0.666L0.234,-0.666L0.234,-0.362L0.304,-0.362Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                                <g transform="matrix(288,0,0,288,1004.75,495.769)">
                                    <path d="M0.159,-0L0.159,-0.042L0.182,-0.042C0.197,-0.042 0.21,-0.044 0.223,-0.047C0.235,-0.05 0.245,-0.056 0.253,-0.066C0.26,-0.075 0.264,-0.09 0.264,-0.109L0.264,-0.298L0.079,-0.619C0.072,-0.632 0.065,-0.642 0.059,-0.65C0.052,-0.657 0.045,-0.663 0.037,-0.667C0.029,-0.67 0.019,-0.672 0.008,-0.672L-0.005,-0.672L-0.005,-0.714L0.27,-0.714L0.27,-0.672L0.233,-0.672C0.215,-0.672 0.203,-0.669 0.198,-0.663C0.192,-0.656 0.189,-0.649 0.189,-0.64C0.189,-0.631 0.191,-0.621 0.195,-0.612C0.199,-0.603 0.203,-0.594 0.207,-0.587L0.281,-0.453C0.292,-0.432 0.302,-0.412 0.311,-0.392C0.319,-0.372 0.326,-0.354 0.331,-0.339C0.337,-0.353 0.346,-0.37 0.357,-0.391C0.368,-0.412 0.38,-0.432 0.391,-0.453L0.455,-0.568C0.462,-0.579 0.466,-0.59 0.469,-0.601C0.472,-0.611 0.473,-0.62 0.473,-0.628C0.473,-0.643 0.468,-0.654 0.458,-0.661C0.447,-0.668 0.432,-0.672 0.413,-0.672L0.384,-0.672L0.384,-0.714L0.628,-0.714L0.628,-0.672L0.616,-0.672C0.607,-0.672 0.598,-0.67 0.59,-0.665C0.581,-0.66 0.572,-0.652 0.564,-0.641C0.555,-0.63 0.544,-0.614 0.533,-0.594L0.365,-0.298L0.365,-0.114C0.365,-0.093 0.369,-0.078 0.376,-0.068C0.383,-0.057 0.393,-0.05 0.406,-0.047C0.419,-0.044 0.432,-0.042 0.447,-0.042L0.47,-0.042L0.47,-0L0.159,-0Z" style="fill: oklch(0.205 0 0); fill-rule: nonzero;"/>
                                </g>
                            </g>
                        </g>
                    </g>
                </svg>
                <p style="margin: 16px 0 0 0; color: oklch(0.556 0 0); font-size: 18px; font-weight: 500;">Welcome to Aviary!</p>
            </div>
            <div class="content">
                <h2>Hello {{.Username}},</h2>
                <p>Welcome to {{.SiteName}}! Your account has been created successfully.</p>
                <p>{{.SiteName}} is a web-based document uploader that automatically downloads and sends ePubs, images, and PDFs to your reMarkable tablet.</p>
                <div class="feature-box">
                    <p><strong>You can now:</strong></p>
                    <ul>
                        <li>Configure your sending preferences</li>
                        <li>Upload documents from URLs or local files</li>
                        <li>Manage your API keys for programmatic access</li>
                    </ul>
                </div>
                <div style="text-align: center;">
                    <a href="{{.SiteURL}}" class="button">Go to {{.SiteName}}</a>
                </div>
            </div>
            <div class="footer">
                <p>This email was sent by {{.SiteName}} • <a href="{{.SiteURL}}">{{.SiteURL}}</a></p>
            </div>
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("welcome").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateWelcomeText generates plain text content for welcome email
func generateWelcomeText(data EmailData) string {
	return fmt.Sprintf(`Welcome to %s!

Hello %s,

Welcome to %s! Your account has been created successfully.

%s is a web-based document uploader that automatically downloads and sends PDFs to your reMarkable tablet.

You can now:
- Configure your sending preferences
- Upload documents from URLs or local files
- Manage your API keys for programmatic access

Visit %s to get started!

--
This email was sent by %s (%s)
`, data.SiteName, data.Username, data.SiteName, data.SiteName, data.SiteURL, data.SiteName, data.SiteURL)
}

// TestSMTPConnection tests the SMTP connection
func TestSMTPConnection() error {
	config, err := GetSMTPConfig()
	if err != nil {
		return err
	}

	// Setup authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Test connection
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Quit()

	// Start TLS if configured
	if config.UseTLS {
		tlsConfig := &tls.Config{ServerName: config.Host}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Test auth
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	return nil
}
