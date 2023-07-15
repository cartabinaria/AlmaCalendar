package unibo

import (
	"github.com/samber/lo"
	"math/rand"
	"testing"
)

func BenchmarkCourses_FindById(b *testing.B) {
	courses := Courses{}
	for i := 0; i < 1000; i++ {
		course := genRandomCourse()
		courses[course.Codice] = course
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		courses.FindById(rand.Int())
	}
}

func genRandomCourse() Course {
	return Course{
		AnnoAccademico:       lo.RandomString(10, lo.LettersCharset),
		Immatricolabile:      lo.RandomString(10, lo.LettersCharset),
		Codice:               rand.Int(),
		Descrizione:          lo.RandomString(10, lo.LettersCharset),
		Url:                  lo.RandomString(10, lo.LettersCharset),
		Campus:               lo.RandomString(10, lo.LettersCharset),
		Ambiti:               lo.RandomString(10, lo.LettersCharset),
		Tipologia:            lo.RandomString(10, lo.LettersCharset),
		DurataAnni:           rand.Int(),
		Internazionale:       rand.Intn(2) == 1,
		InternazionaleTitolo: lo.RandomString(10, lo.LettersCharset),
		InternazionaleLingua: lo.RandomString(10, lo.LettersCharset),
		Lingue:               lo.RandomString(10, lo.LettersCharset),
		Accesso:              lo.RandomString(10, lo.LettersCharset),
		SedeDidattica:        lo.RandomString(10, lo.LettersCharset),
	}
}

func TestCourses_FindById(t *testing.T) {
	courses := Courses{1: {Codice: 1}, 2: {Codice: 2}, 3: {Codice: 3}}

	course, found := courses.FindById(2)
	if !found {
		t.Fatal("course not found")
	}

	if course.Codice != 2 {
		t.Error("wrong course")
	}
}
