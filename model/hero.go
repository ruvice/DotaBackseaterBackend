package model

type Hero struct {
	Name     string `json:"name,omitempty"`
	HeroName string `json:"hero_name,omitempty"`
	ID       string `json:"id,omitempty"`
}

type HeroResponse struct {
	Items []Hero `json:"items,omitempty"`
}

type HeroMap map[string]Hero
