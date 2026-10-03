// Package home serves Home: the Banners, the Season hero card and the Ayah of the day.
package home

import "time"

// Season is Season in the app. Only one is on at a time.
type Season string

// The seasons, named as the Dart enum values.
const (
	Ramadan      Season = "ramadan"
	BoiMela      Season = "boiMela"
	Admission    Season = "admission"
	BackToSchool Season = "backToSchool"
)

// Seasons lists the valid season names.
var Seasons = []Season{Ramadan, BoiMela, Admission, BackToSchool}

// IsSeason reports whether name is a season.
func IsSeason(name string) bool {
	for _, s := range Seasons {
		if string(s) == name {
			return true
		}
	}
	return false
}

type day struct{ y, m, d int }

func (a day) before(b day) bool {
	if a.y != b.y {
		return a.y < b.y
	}
	if a.m != b.m {
		return a.m < b.m
	}
	return a.d < b.d
}

// ramadan lists each year Ramadan runs, both ends included. It moves about 11 days a year, so
// each year is listed (port of SeasonFixtures.ramadan; seeded to 2028).
var ramadan = [][2]day{
	{{2026, 2, 18}, {2026, 3, 19}},
	{{2027, 2, 8}, {2027, 3, 9}},
	{{2028, 1, 28}, {2028, 2, 26}},
}

// byMonth: the other seasons fill whole months, every year.
var byMonth = map[time.Month]Season{
	time.January:  BackToSchool,
	time.February: BoiMela,
	time.October:  Admission,
	time.November: Admission,
	time.December: Admission,
}

// ActiveOn is SeasonPicker.activeOn: the override when Staff set one, else Ramadan when the day
// falls in it, else the Season of the month; "" outside every Season. date is read as a calendar
// day in its own location.
func ActiveOn(date time.Time, override Season) Season {
	if override != "" {
		return override
	}
	d := day{date.Year(), int(date.Month()), date.Day()}
	for _, w := range ramadan {
		if !d.before(w[0]) && !w[1].before(d) {
			return Ramadan
		}
	}
	return byMonth[date.Month()]
}

// SeasonInfo is SeasonModel: the hero card of a Season.
type SeasonInfo struct {
	Season       Season `json:"season"`
	TitleEn      string `json:"titleEn"`
	TitleBn      string `json:"titleBn"`
	SubtitleEn   string `json:"subtitleEn"`
	SubtitleBn   string `json:"subtitleBn"`
	Seed         int    `json:"seed"`
	CollectionID string `json:"collectionId"`
}

// Info is what each Season hero card says (SeasonFixtures.info).
var Info = map[Season]SeasonInfo{
	Ramadan: {Ramadan, "Ramadan reading", "রমজানের পাঠ", "Quran, Seerah and Hadith for the blessed month",
		"বরকতময় মাসের জন্য কুরআন, সীরাত ও হাদিস", 3, "col-ramadan"},
	BoiMela: {BoiMela, "Boi Mela is here", "বইমেলা চলছে", "This year’s fair picks, in Bangla",
		"এ বছরের মেলার বাছাই বই, বাংলায়", 1, "col-boi-mela"},
	Admission: {Admission, "Admission season", "ভর্তি মৌসুম", "Guides and question banks for every test",
		"প্রতিটি ভর্তি পরীক্ষার গাইড ও প্রশ্নব্যাংক", 0, "col-admission"},
	BackToSchool: {BackToSchool, "Back to school", "স্কুলে ফেরা", "Textbooks and grammar for the new year",
		"নতুন বছরের পাঠ্যবই ও ব্যাকরণ", 2, "col-back-to-school"},
}
