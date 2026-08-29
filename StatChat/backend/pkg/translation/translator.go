package translation

import (
	"errors"
	"regexp"
	"strings"
)

type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Result struct {
	SourceLanguage string `json:"sourceLanguage"`
	TargetLanguage string `json:"targetLanguage"`
	Text           string `json:"text"`
	Provider       string `json:"provider"`
}

var ErrUnsupportedLanguage = errors.New("unsupported language")

var languages = []Language{
	{Code: "en", Name: "English"},
	{Code: "sw", Name: "Kiswahili"},
	{Code: "fr", Name: "French"},
	{Code: "es", Name: "Spanish"},
}

var wordGlossary = map[string]map[string]string{
	"en:sw": {"hello": "habari", "team": "timu", "good": "nzuri", "morning": "asubuhi", "welcome": "karibu", "thank": "asante", "you": "wewe", "please": "tafadhali", "meeting": "mkutano", "today": "leo", "tomorrow": "kesho", "project": "mradi", "data": "data", "report": "ripoti", "health": "afya", "work": "kazi", "share": "shiriki", "review": "hakiki", "ready": "tayari"},
	"en:fr": {"hello": "bonjour", "team": "equipe", "good": "bon", "morning": "matin", "welcome": "bienvenue", "thank": "merci", "you": "vous", "please": "s'il vous plait", "meeting": "reunion", "today": "aujourd'hui", "tomorrow": "demain", "project": "projet", "data": "donnees", "report": "rapport", "health": "sante", "work": "travail", "share": "partager", "review": "reviser", "ready": "pret"},
	"en:es": {"hello": "hola", "team": "equipo", "good": "bueno", "morning": "manana", "welcome": "bienvenido", "thank": "gracias", "you": "usted", "please": "por favor", "meeting": "reunion", "today": "hoy", "tomorrow": "manana", "project": "proyecto", "data": "datos", "report": "informe", "health": "salud", "work": "trabajo", "share": "compartir", "review": "revisar", "ready": "listo"},
}

var phraseGlossary = map[string]map[string]string{
	"en:sw": {"hello team": "habari timu", "good morning": "habari za asubuhi", "thank you": "asante", "the meeting is today": "mkutano ni leo", "are you ready": "uko tayari"},
	"en:fr": {"hello team": "bonjour equipe", "good morning": "bonjour", "thank you": "merci", "the meeting is today": "la reunion est aujourd'hui", "are you ready": "etes-vous pret"},
	"en:es": {"hello team": "hola equipo", "good morning": "buenos dias", "thank you": "gracias", "the meeting is today": "la reunion es hoy", "are you ready": "estas listo"},
}

var tokenPattern = regexp.MustCompile(`[[:alpha:]]+|[^[:alpha:]]+`)

func SupportedLanguages() []Language {
	return append([]Language(nil), languages...)
}

func isSupported(code string) bool {
	for _, language := range languages {
		if language.Code == code {
			return true
		}
	}
	return false
}

func Translate(text, source, target string) (Result, error) {
	text = strings.TrimSpace(text)
	source = strings.ToLower(strings.TrimSpace(source))
	target = strings.ToLower(strings.TrimSpace(target))
	if source == "" || source == "auto" {
		source = "en"
	}
	if !isSupported(source) || !isSupported(target) {
		return Result{}, ErrUnsupportedLanguage
	}
	if text == "" {
		return Result{SourceLanguage: source, TargetLanguage: target, Provider: "local-glossary"}, nil
	}
	if source == target {
		return Result{SourceLanguage: source, TargetLanguage: target, Text: text, Provider: "identity"}, nil
	}
	key := source + ":" + target
	lower := strings.ToLower(text)
	if phrase, ok := phraseGlossary[key][lower]; ok {
		return Result{SourceLanguage: source, TargetLanguage: target, Text: phrase, Provider: "local-glossary"}, nil
	}
	words, ok := wordGlossary[key]
	if !ok {
		return Result{}, ErrUnsupportedLanguage
	}
	translated := tokenPattern.ReplaceAllStringFunc(text, func(token string) string {
		if !regexp.MustCompile(`^[[:alpha:]]+$`).MatchString(token) {
			return token
		}
		replacement, exists := words[strings.ToLower(token)]
		if !exists {
			return token
		}
		if len(token) > 0 && token[0] >= 'A' && token[0] <= 'Z' {
			return strings.ToUpper(replacement[:1]) + replacement[1:]
		}
		return replacement
	})
	return Result{SourceLanguage: source, TargetLanguage: target, Text: translated, Provider: "local-glossary"}, nil
}
