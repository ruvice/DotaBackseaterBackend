package model

type Item struct {
	ItemID string `json:"item_id,omitempty"`
}

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
