package user 

type User struct {
    Id    string `json:"id" example:"123"`
    Name  string `json:"name" example:"Alice"`
    Email string `json:"email" example:"alice@example.com"`
}
