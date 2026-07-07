package dto

type UserCreateRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Name     string `json:"name"     validate:"required"`
	Role     string `json:"role"     validate:"required,oneof=SALES OPS MANAGER ADMIN"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type UserUpdateRequest struct {
	Name   *string `json:"name"   validate:"omitempty"`
	Role   *string `json:"role"   validate:"omitempty,oneof=SALES OPS MANAGER ADMIN"`
	Active *bool   `json:"active" validate:"omitempty"`
}

type ResetPasswordRequest struct {
	Password *string `json:"password" validate:"omitempty,min=8,max=72"`
}

type ResetPasswordResponse struct {
	Password string `json:"password,omitempty"`
}

type UserListResponse struct {
	Items    []UserResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
