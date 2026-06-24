package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var languageCodes = map[string]string{
	"hindi":      "hi",
	"hi":         "hi",
	"spanish":    "es",
	"es":         "es",
	"french":     "fr",
	"fr":         "fr",
	"german":     "de",
	"de":         "de",
	"italian":    "it",
	"it":         "it",
	"japanese":   "ja",
	"ja":         "ja",
	"chinese":    "zh",
	"zh":         "zh",
	"russian":    "ru",
	"ru":         "ru",
	"arabic":     "ar",
	"ar":         "ar",
	"portuguese": "pt",
	"pt":         "pt",
	"korean":     "ko",
	"ko":         "ko",
	"dutch":      "nl",
	"nl":         "nl",
	"turkish":    "tr",
	"tr":         "tr",
	"swedish":    "sv",
	"sv":         "sv",
	"polish":     "pl",
	"pl":         "pl",
	"vietnamese": "vi",
	"vi":         "vi",
	"latin":      "la",
	"la":         "la",
	"greek":      "el",
	"el":         "el",
}

type translationResponse struct {
	ResponseData struct {
		TranslatedText string `json:"translatedText"`
	} `json:"responseData"`
}

// Translate translates text from English to the target language.
func Translate(text, targetLang string) (string, error) {
	langCode := strings.ToLower(strings.TrimSpace(targetLang))
	if code, ok := languageCodes[langCode]; ok {
		langCode = code
	}

	params := url.Values{}
	params.Set("q", text)
	params.Set("langpair", "en|"+langCode)

	reqURL := "https://api.mymemory.translated.net/get?" + params.Encode()
	client := &http.Client{Timeout: 8 * time.Second}

	resp, err := client.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("translation network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("translation read error: %w", err)
	}

	var res translationResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("translation parse error: %w", err)
	}

	translated := strings.TrimSpace(res.ResponseData.TranslatedText)
	if translated == "" {
		return "", fmt.Errorf("no translation found")
	}

	return translated, nil
}
