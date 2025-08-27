package schema

import "time"

//go:generate go run github.com/objectbox/objectbox-go/cmd/objectbox-gogen

type Card struct {
	Id     uint64
	Data   string
	CardId string
}

type Draft struct {
	Id                 uint64
	Name               string
	Format             string
	InPerson           bool
	Seats              []*Seat
	UnassignedPacks    []*Pack
	Events             []*Event
	SpectatorChannelId string `objectbox:"index"`
	PickTwo            bool
}

type Pack struct {
	Id            uint64
	Round         int
	OriginalCards []*Card
	Cards         []*Card
}

type Seat struct {
	Id            uint64
	Position      int
	User          *User `objectbox:"link"`
	ReservedUser  *User `objectbox:"link"`
	ScanSound     int
	ErrorSound    int
	Round         int
	Packs         []*Pack
	OriginalPacks []*Pack
	PickedCards   []*Card
}

type User struct {
	Id          uint64
	DiscordId   string `objectbox:"unique,index"`
	DiscordName string
	MtgoName    string
	Picture     string
	Skips       []*Skip
}

type Event struct {
	Id           uint64
	Position     int
	Announcement string
	Card1        *Card `objectbox:"link"`
	Card2        *Card `objectbox:"link"`
	Pack         *Pack `objectbox:"link"`
	Modified     int
	Round        int
}

type Skip struct {
	Id      uint64
	DraftId uint64
}

type RoleMsg struct {
	Id     uint64
	MsgId  string `objectbox:"index"`
	Emoji  string
	RoleId string
}

type PairingMsg struct {
	Id    uint64
	MsgId string `objectbox:"index"`
	Draft *Draft `objectbox:"link"`
	Round int
}

type Result struct {
	Id        uint64
	Draft     *Draft `objectbox:"link"`
	Round     int
	User      *User `objectbox:"link"`
	Win       bool
	Timestamp time.Time `objectbox:"date"`
}
