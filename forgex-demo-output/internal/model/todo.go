package model

type Todo struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
	Note   string `json:"note"`
}

type CreateTodoRequest struct {
	Title string `json:"title"`
	Note  string `json:"note"`
}

type UpdateTodoRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
	Note  *string `json:"note"`
}
