package main

type Game struct {
	ID      string  `json:"id"`
	Ruleset Ruleset `json:"ruleset"`
	Map     string  `json:"map"`
	Source  string  `json:"source"`
	Timeout int     `json:"timeout"`
}

type Ruleset struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Settings Settings `json:"settings"`
}

type Settings struct {
	FoodSpawnChance     int `json:"foodSpawnChance"`
	MinimumFood         int `json:"minimumFood"`
	HazardDamagePerTurn int `json:"hazardDamagePerTurn"`
}

type Board struct {
	Height  int     `json:"height"`
	Width   int     `json:"width"`
	Food    []Point `json:"food"`
	Hazards []Point `json:"hazards"`
	Snakes  []Snake `json:"snakes"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Snake struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Health  int     `json:"health"`
	Body    []Point `json:"body"`
	Latency string  `json:"latency"`
	Head    Point   `json:"head"`
	// Length         int            `json:"length"`
	Shout          string         `json:"shout"`
	Customizations Customizations `json:"customizations"`
}

type Customizations struct {
	Color string `json:"color"`
	Head  string `json:"head"`
	Tail  string `json:"tail"`
}

type BattleSnakeGame struct {
	Game  Game  `json:"game"`
	Turn  int   `json:"turn"`
	Board Board `json:"board"`
	You   Snake `json:"you"`
}
