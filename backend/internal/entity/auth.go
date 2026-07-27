package entity

type RegisterReq struct {
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8,max=72"`
}

type LoginReq struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutReq struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordReq struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordReq struct {
	Token       string `json:"token"        validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}
