package schema

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
	SpectatorChannelId string
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
	DiscordId   string `objectbox:"unique"`
	DiscordName string
	MtgoName    string
	Picture     string
}

type Event struct {
	Id           uint64
	Position     int
	Announcement string
	Card1        *Card `objectbox:"link"`
	Card2        *Card `objectbox:"link"`
	Modified     int
	Round        int
}
