package plugins

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateImageResponse creates a response for image-type plugins with refresh rate
func CreateImageResponse(imageURL, filename string, refreshRate int) PluginResponse {
	return gin.H{
		"plugin_type":  string(PluginTypeImage),
		"image_url":    imageURL,
		"filename":     filename,
		"refresh_rate": fmt.Sprintf("%d", refreshRate),
	}
}

// CreateImageResponseWithoutRefresh creates a response for image-type plugins without refresh rate
func CreateImageResponseWithoutRefresh(imageURL, filename string) PluginResponse {
	return gin.H{
		"plugin_type": string(PluginTypeImage),
		"image_url":   imageURL,
		"filename":    filename,
	}
}

// CreateImageDataResponse creates a response for image-type plugins with raw image data
func CreateImageDataResponse(imageData []byte, filename string) PluginResponse {
	return gin.H{
		"plugin_type": string(PluginTypeImage),
		"image_data":  imageData,
		"filename":    filename,
	}
}

// CreateDataResponse creates a response for data-type plugins
func CreateDataResponse(data map[string]interface{}, template string, refreshRate int) PluginResponse {
	return gin.H{
		"plugin_type":  string(PluginTypeData),
		"data":         data,
		"template":     template,
		"refresh_rate": fmt.Sprintf("%d", refreshRate),
	}
}

// CreateErrorResponse creates an error response
func CreateErrorResponse(message string) PluginResponse {
	return gin.H{
		"error":     true,
		"message":   message,
		"timestamp": time.Now().UTC(),
	}
}

// IsImageResponse checks if a response is from an image plugin
func IsImageResponse(response PluginResponse) bool {
	pluginType, ok := response["plugin_type"].(string)
	return ok && pluginType == string(PluginTypeImage)
}

// IsDataResponse checks if a response is from a data plugin
func IsDataResponse(response PluginResponse) bool {
	pluginType, ok := response["plugin_type"].(string)
	return ok && pluginType == string(PluginTypeData)
}

// IsErrorResponse checks if a response contains an error
func IsErrorResponse(response PluginResponse) bool {
	if errVal, ok := response["error"]; ok {
		if err, isBool := errVal.(bool); isBool {
			return err
		}
	}
	return false
}

// GetImageURL extracts the image URL from an image response
func GetImageURL(response PluginResponse) (string, bool) {
	if url, ok := response["image_url"].(string); ok {
		return url, true
	}
	return "", false
}

// GetImageData extracts the image data from an image response
func GetImageData(response PluginResponse) ([]byte, bool) {
	if data, ok := response["image_data"].([]byte); ok {
		return data, true
	}
	return nil, false
}

// GetData extracts the data from a data response
func GetData(response PluginResponse) (map[string]interface{}, bool) {
	if data, ok := response["data"].(map[string]interface{}); ok {
		return data, true
	}
	return nil, false
}

// GetTemplate extracts the template from a data response
func GetTemplate(response PluginResponse) (string, bool) {
	if template, ok := response["template"].(string); ok {
		return template, true
	}
	return "", false
}

// GetRefreshRate extracts the refresh rate from any response
func GetRefreshRate(response PluginResponse) (int, bool) {
	if rate, ok := response["refresh_rate"].(string); ok {
		// Try to parse as string first (standard format)
		var intRate int
		if _, err := fmt.Sscanf(rate, "%d", &intRate); err == nil {
			return intRate, true
		}
	}
	
	// Try as direct int (fallback)
	if rate, ok := response["refresh_rate"].(int); ok {
		return rate, true
	}
	
	return 0, false
}

// CreateHTMLResponse creates a response containing HTML data
func CreateHTMLResponse(html string) PluginResponse {
	return gin.H{
		"type":    "html",
		"content": html,
	}
}

// IsHTMLResponse checks if a response is an HTML response
func IsHTMLResponse(response PluginResponse) bool {
	if responseType, ok := response["type"].(string); ok {
		return responseType == "html"
	}
	return false
}

// GetHTMLContent extracts HTML content from a response
func GetHTMLContent(response PluginResponse) (string, bool) {
	if content, ok := response["content"].(string); ok {
		return content, true
	}
	return "", false
}