package profile

// Details is ProfileDetailsModel: {name, phone, photo?}; photo is a base64 image.
type Details struct {
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Photo *string `json:"photo"`
}

// Prefs is ProfilePrefsModel: the muted notification groups and the two privacy switches.
type Prefs struct {
	Muted           []string `json:"muted"`
	ProfileVisible  bool     `json:"profileVisible"`
	ActivityVisible bool     `json:"activityVisible"`
}

// DefaultPrefs is what a new reader has.
func DefaultPrefs() Prefs {
	return Prefs{Muted: []string{}, ProfileVisible: true, ActivityVisible: true}
}

// MutableGroups are the notification groups a reader can mute (NotificationGroup in the app).
var MutableGroups = []string{"orders", "usedBooks", "alerts", "community"}

type ok struct {
	OK bool `json:"ok"`
}

type idBody struct {
	ID string `json:"id"`
}
