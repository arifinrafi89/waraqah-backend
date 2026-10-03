package assistant

import "testing"

// Ported from test/smarter_ai_test.dart ("plain words become filters, in English and Bangla").
func TestParse(t *testing.T) {
	a := Parse("Books for Class 9 under ৳1,000", nil)
	if a.ClassLevel == nil || *a.ClassLevel != 9 || a.MaxPrice == nil || *a.MaxPrice != 1000 || !a.Basket {
		t.Errorf("class 9 under 1000: %+v", a)
	}
	b := Parse("৯ম শ্রেণির বই ১০০০ টাকার মধ্যে", nil)
	if b.ClassLevel == nil || *b.ClassLevel != 9 || b.MaxPrice == nil || *b.MaxPrice != 1000 {
		t.Errorf("Bangla class and price: %+v", b)
	}
	c := Parse("short seerah for beginners in Bangla", nil)
	if c.Kind != Seerah || c.Language != "bangla" {
		t.Errorf("seerah in Bangla: %+v", c)
	}
	if got := Parse("HSC ebook", nil); got.Format != "ebook" || got.Exam != "hsc" {
		t.Errorf("HSC ebook: %+v", got)
	}
	if Parse("good morning", nil).SearchesBooks() {
		t.Error("small talk searches no books")
	}
	if got := Parse("Books by Humayun Ahmed", nil); got.Kind != AuthorSearch || got.Query != "Humayun Ahmed" {
		t.Errorf("author: %+v", got)
	}
	if got := Parse("one more please", []string{"Here are some books for you"}); got.Kind != GeneralRecommendations {
		t.Errorf("more after books: %+v", got)
	}
	if got := Parse("class 3 books", nil); got.ClassLevel != nil {
		t.Errorf("classes run 6 to 12: %+v", got)
	}
}

func TestDigits(t *testing.T) {
	if got := digits("মোট ৳1250"); got != "মোট ৳১২৫০" {
		t.Errorf("Bangla text: %q", got)
	}
	if got := digits("Total ৳1250"); got != "Total ৳1250" {
		t.Errorf("English text: %q", got)
	}
}
