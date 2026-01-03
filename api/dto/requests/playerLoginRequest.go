package requests

type PlayerLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
