package goramwordnet

import (
	"path"
	"runtime"
	"testing"
)

const PathToWordnetDataFiles = "./data"

func sourceCodeRelPath(suffix string) string {
	_, fileName, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(fileName), suffix)
}

var wnInstance *Handle
var wnErr error

func init() {
	wnInstance, wnErr = New(sourceCodeRelPath(PathToWordnetDataFiles))
}

func TestParsing(t *testing.T) {
	if wnErr != nil {
		t.Fatalf("Can't initialize: %s", wnErr)
	}
}

func TestBasicLookup(t *testing.T) {
	// very basic test
	found, err := wnInstance.Lookup(Criteria{Matching: "good"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotAdjective := false
	for _, f := range found {
		if f.POS() == Adjective {
			gotAdjective = true
			break
		}
	}
	if !gotAdjective {
		t.Errorf("couldn't find basic adjective form for good")
	}
}

func TestLemma(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "awesome", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for awesome, got %d", len(found))
	}
	if found[0].Lemma() != "amazing" {
		t.Errorf("incorrect lemma for awesome (%s)", found[0].Lemma())
	}
}

func setContains(haystack, needles []string) bool {
	for _, n := range needles {
		found := false
		for _, h := range haystack {
			if n == h {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestSynonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "yummy", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for yummy, got %d", len(found))
	}

	syns := found[0].Synonyms()
	if !setContains(syns, []string{"delicious", "delectable"}) {
		t.Errorf("missing synonyms for yummy")
	}
}

func TestAntonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var antonyms []string
	for _, f := range found {
		as := f.Related(Antonym)
		for _, a := range as {
			antonyms = append(antonyms, a.Word())
		}
	}
	if !setContains(antonyms, []string{"bad", "evil"}) {
		t.Errorf("missing antonyms for good")
	}
}

func TestHypernyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "jab", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hypernyms []string
	for _, f := range found {
		as := f.Related(Hypernym)
		for _, a := range as {
			hypernyms = append(hypernyms, a.Word())
		}
	}
	if !setContains(hypernyms, []string{"punch"}) {
		t.Errorf("missing hypernyms for jab (expected punch, got %v)", hypernyms)
	}
}

func TestHyponyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "food", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hyponyms []string
	for _, f := range found {
		as := f.Related(Hyponym)
		for _, a := range as {
			hyponyms = append(hyponyms, a.Word())
		}
	}
	expected := []string{"chocolate", "cheese", "pasta", "leftovers"}
	if !setContains(hyponyms, expected) {
		t.Errorf("missing hyponyms for candy (expected %v, got %v)", expected, hyponyms)
	}
}

func TestIterate(t *testing.T) {
	count := 0
	wnInstance.Iterate(PartOfSpeechList{Noun}, func(l Lookup) error {
		count++
		return nil
	})
	if count != 82192 {
		t.Errorf("Missing nouns!")
	}
}
