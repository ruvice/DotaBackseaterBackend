package model

type Item struct {
	ItemID     string     `json:"item_id,omitempty"`
	ItemDetail ItemDetail `json:"detail,omitempty"`
}

type ItemDetail struct {
	Name     string `json:"name,omitempty"`
	ItemName string `json:"item_name,omitempty"`
	Cost     int32  `json:"cost,omitempty"`
}

type ItemsResponse struct {
	Items []Item `json:"items,omitempty"`
}

type ItemMap map[string]ItemDetail

func (i *Item) ItemName() string {
	switch i.ItemID {
	case "1":
		return "Blink Dagger"
	case "2":
		return "BKB"
	case "3":
		return "Mantle of Intelligence"
	}
	return ""
}
