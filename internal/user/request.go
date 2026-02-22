package user

type CreateUserRequest struct {
    Name string `json:"name" example:"Alice"`
    Email string `json:"email" example:"alice@example.com"`
}

type UpdateUserRequest struct {
    Name *string `json:"name,omitempty" example:"Bob"`
    Emaile *string `json:"email,omitempty" example:"bob@example.com"`
}
