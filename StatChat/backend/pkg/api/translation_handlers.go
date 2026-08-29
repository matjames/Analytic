package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"statchat/pkg/translation"
)

func translationLanguagesHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, translation.SupportedLanguages())
}

func translateTextHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text           string `json:"text"`
		SourceLanguage string `json:"sourceLanguage"`
		TargetLanguage string `json:"targetLanguage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" || len(req.Text) > 10000 {
		writeError(w, http.StatusBadRequest, "text is required and must be no larger than 10000 characters")
		return
	}
	result, err := translation.Translate(req.Text, req.SourceLanguage, req.TargetLanguage)
	if errors.Is(err, translation.ErrUnsupportedLanguage) {
		writeError(w, http.StatusBadRequest, "unsupported source or target language")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "translation failed")
		return
	}
	writeJSON(w, result)
}
