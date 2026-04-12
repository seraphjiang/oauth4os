// Package i18n serves localization files for consent and device screens.
package i18n

import (
	"encoding/json"
	"net/http"
)

var translations = map[string]map[string]string{
	"en": {
		"title":       "Authorization Request",
		"description": "wants to access your account",
		"scopes":      "Requested permissions",
		"approve":     "Approve",
		"deny":        "Deny",
		"device":      "Device Authorization",
		"enter_code":  "Enter the code shown on your device",
	},
	"es": {
		"title":       "Solicitud de autorización",
		"description": "quiere acceder a tu cuenta",
		"scopes":      "Permisos solicitados",
		"approve":     "Aprobar",
		"deny":        "Denegar",
		"device":      "Autorización de dispositivo",
		"enter_code":  "Ingresa el código que aparece en tu dispositivo",
	},
	"fr": {
		"title":       "Demande d'autorisation",
		"description": "souhaite accéder à votre compte",
		"scopes":      "Autorisations demandées",
		"approve":     "Approuver",
		"deny":        "Refuser",
		"device":      "Autorisation de l'appareil",
		"enter_code":  "Entrez le code affiché sur votre appareil",
	},
	"de": {
		"title":       "Autorisierungsanfrage",
		"description": "möchte auf Ihr Konto zugreifen",
		"scopes":      "Angeforderte Berechtigungen",
		"approve":     "Genehmigen",
		"deny":        "Ablehnen",
		"device":      "Geräteautorisierung",
		"enter_code":  "Geben Sie den auf Ihrem Gerät angezeigten Code ein",
	},
	"ja": {
		"title":       "認可リクエスト",
		"description": "があなたのアカウントへのアクセスを要求しています",
		"scopes":      "要求された権限",
		"approve":     "承認",
		"deny":        "拒否",
		"device":      "デバイス認可",
		"enter_code":  "デバイスに表示されたコードを入力してください",
	},
	"zh": {
		"title":       "授权请求",
		"description": "请求访问您的账户",
		"scopes":      "请求的权限",
		"approve":     "批准",
		"deny":        "拒绝",
		"device":      "设备授权",
		"enter_code":  "请输入设备上显示的代码",
	},
}

// Handler serves GET /i18n/consent.json.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(translations)
}
