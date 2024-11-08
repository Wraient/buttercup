package internal

// Create these in a new file like models.go or types.go

type JackettResponse struct {
    Results  []Release `json:"Results"`
    Indexers []Indexer `json:"Indexers"`
}

type Release struct {
    Title         string   `json:"Title"`
    Guid          string   `json:"Guid"`
    Size          int64    `json:"Size"`
    PublishDate   string   `json:"PublishDate"`
    Category      []int    `json:"Category"`
    Description   string   `json:"Description"`
    InfoHash      string   `json:"InfoHash"`
    MagnetUri     string   `json:"MagnetUri"`
    Seeders       int      `json:"Seeders"`
    Peers         int      `json:"Peers"`
    TrackerType   string   `json:"TrackerType"`
    Tracker       string   `json:"Tracker"`
    CategoryDesc  string   `json:"CategoryDesc"`
    Files         int      `json:"Files"`
    Genres        []string `json:"Genres,omitempty"`
    Languages     []string `json:"Languages"`
}

type Indexer struct {
    ID          string `json:"ID"`
    Name        string `json:"Name"`
    Status      int    `json:"Status"`
    Results     int    `json:"Results"`
    Error       string `json:"Error,omitempty"`
    ElapsedTime int    `json:"ElapsedTime"`
}

type Torrent struct {
    Title    string
    URI      string
    Size     int64
    Seeders  int
    Leechers int
    Files    []string
    FileIndex int
    SortedFiles []string
}

type User struct {
	Watching   Torrent
	Player     Player
	Resume     bool
}

type Player struct {
	SocketPath string
	PlaybackTime int
	Started      bool
	Duration int
	Speed float64
}

