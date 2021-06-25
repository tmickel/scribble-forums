package main

import "net/http"

func setFlash(w http.ResponseWriter, message string, style string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_text",
		Value:    message,
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_style",
		Value:    style,
		HttpOnly: true,
		Secure:   true,
	})
}

func clearFlash(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_text",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_style",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
	})
}
