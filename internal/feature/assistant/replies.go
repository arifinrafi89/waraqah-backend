package assistant

import (
	"fmt"
	"strings"
)

// Port of AssistantReplies (assistant_replies.dart): what the assistant says, in English or Bangla.

func greetingText(bn bool) string {
	if bn {
		return "আসসালামু আলাইকুম! ওয়ারাকাহর পুরো ক্যাটালগ আমার জানা। কোনো বই, বিষয়, লেখক বা বাজেট বলুন।"
	}
	return "Assalamu Alaikum! I know Waraqah's whole catalog. Ask for a book, a topic, an author or a budget."
}

func foundText(bn bool, in Intent) string {
	what := label(bn, in)
	if bn {
		under := ""
		if in.MaxPrice != nil {
			under = fmt.Sprintf(" ৳%d-এর মধ্যে", *in.MaxPrice)
		}
		return "ওয়ারাকাহর ক্যাটালগ থেকে " + what + under + "। প্রতিটিতে ওয়ারাকাহর দাম আর স্টক আছে।"
	}
	under := ""
	if in.MaxPrice != nil {
		under = fmt.Sprintf(" under ৳%d", *in.MaxPrice)
	}
	return "Here are " + what + under + " from Waraqah's catalog, each with our price and stock."
}

func noneText(bn bool) string {
	if bn {
		return "ওয়ারাকাহর ক্যাটালগে এমন কিছু পাইনি। অন্য নাম, লেখক বা বাজেট দিয়ে দেখুন, অথবা বইটির অনুরোধ করুন।"
	}
	return "I couldn't find that in Waraqah's catalog. Try another title, author or budget, or request the book."
}

func helloText(bn bool) string {
	if bn {
		return "ওয়া আলাইকুমুস সালাম! কী পড়তে চান? বিষয়, লেখক বা বাজেট বলুন।"
	}
	return "Wa alaikum assalam! What would you like to read? Tell me a topic, an author or a budget."
}

func studyText(bn bool) string {
	if bn {
		return `পরীক্ষার প্রস্তুতির জন্য ক্যাটালগের "Admission & Job Prep" আর "School & College" বিভাগ দেখুন, অথবা বিষয় আর ক্লাস বলুন।`
	}
	return "For exam prep, try the Admission & Job Prep and School & College sections, or tell me the subject and class."
}

func sellText(bn bool) string {
	if bn {
		return "পুরোনো বই বিক্রি করতে P2P ট্যাবে লিস্ট করুন, অথবা প্রোফাইল থেকে সেল ব্যাকে ওয়ারাকাহর কাছে বিক্রি করুন।"
	}
	return "To sell a used book, list it in the P2P tab, or sell it back to Waraqah from your Profile for an instant price."
}

func thanksText(bn bool) string {
	if bn {
		return "স্বাগতম! আরও বই খুঁজতে যেকোনো সময় জিজ্ঞেস করুন।"
	}
	return "You're welcome! Ask any time you want another book."
}

func fallbackText(bn bool) string {
	if bn {
		return "আমি ওয়ারাকাহর বই, দাম আর স্টক নিয়ে সাহায্য করতে পারি। একটি বিষয়, লেখক বা বাজেট বলুন।"
	}
	return "I can help with Waraqah's books, prices and stock. Tell me a topic, an author or a budget."
}

// basketText is a basket for the cart: how many Books and what they cost together.
func basketText(bn bool, in Intent, count, total int) string {
	what := label(bn, in)
	if bn {
		under := ""
		if in.MaxPrice != nil {
			under = fmt.Sprintf(" (৳%d-এর মধ্যে)", *in.MaxPrice)
		}
		return fmt.Sprintf("%s%s: %dটি বই, মোট ৳%d। নিচ থেকে সবগুলো কার্টে যোগ করুন।", what, under, count, total)
	}
	under := ""
	if in.MaxPrice != nil {
		under = fmt.Sprintf(" within ৳%d", *in.MaxPrice)
	}
	return fmt.Sprintf("Here's a basket of %s%s: %d books, ৳%d in all. Add them all to your cart below.", what, under, count, total)
}

func label(bn bool, in Intent) string {
	pick := func(bangla, english string) string {
		if bn {
			return bangla
		}
		return english
	}
	var base string
	switch {
	case in.Kind == Seerah:
		base = pick("সীরাহর বই", "books on the Seerah")
	case in.Kind == Hadith:
		base = pick("হাদিসের বই", "Hadith books")
	case in.Kind == Quran:
		base = pick("কুরআনের বই", "Quran books")
	case in.Kind == IslamicHistory:
		base = pick("ইসলামের ইতিহাসের বই", "Islamic history books")
	case in.Kind == AuthorSearch:
		base = pick(in.Query+"-এর বই", "books by "+in.Query)
	case in.IsIslamic():
		base = pick("ইসলামিক বই", "Islamic books")
	case in.ClassLevel != nil:
		base = pick(fmt.Sprintf("%dম শ্রেণির বই", *in.ClassLevel), fmt.Sprintf("books for Class %d", *in.ClassLevel))
	case in.Exam != "":
		base = pick(strings.ToUpper(in.Exam)+" প্রস্তুতির বই", strings.ToUpper(in.Exam)+" books")
	default:
		base = pick("আপনার জন্য কিছু বই", "some books for you")
	}
	switch in.Language {
	case "bangla":
		base += pick(" (বাংলায়)", " in Bangla")
	case "english":
		base += pick(" (ইংরেজিতে)", " in English")
	case "arabic":
		base += pick(" (আরবিতে)", " in Arabic")
	}
	return base
}
