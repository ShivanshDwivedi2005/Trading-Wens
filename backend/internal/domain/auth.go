package domain

type User struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	DisplayName      string `json:"display_name,omitempty"`
	EmailConfirmedAt string `json:"email_confirmed_at,omitempty"`
}

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type AuthResult struct {
	User    User     `json:"user"`
	Session *Session `json:"session,omitempty"`
}
