package commonApiStructs

type RequestPage struct {
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
	Keyword string `json:"keyword"`
}

type RequestDeleteIds[T int | uint] struct {
	Ids []T `json:"ids"`
}

type SortRequestItem struct {
	Id   uint `json:"id"`
	Sort uint `json:"sort"`
}

type SortRequest struct {
	SortItems []SortRequestItem `json:"sortItems"`
}
