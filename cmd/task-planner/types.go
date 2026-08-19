package main

type tokenStore struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	ClientID     string `json:"client_id"`
	ExpiresAt    int64  `json:"expires_at"`
}

type project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type plan struct {
	ID, Content, Period, ProjectID string
	DueString                      *string
	Priority                       *int16
}
