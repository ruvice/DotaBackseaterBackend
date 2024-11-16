package model

type Item struct {
	Name     string `json:"name,omitempty"`
	ItemName string `json:"item_name,omitempty"`
	Cost     int32  `json:"cost,omitempty"`
	ID       string `json:"id,omitempty"`
}

type ItemsResponse struct {
	Items []Item `json:"items,omitempty"`
}

type ItemMap map[string]Item
